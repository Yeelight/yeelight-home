package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SpaceOrganizationKind string

const (
	SpaceOrganizationRoomRename   SpaceOrganizationKind = "room.rename"
	SpaceOrganizationRoomUpdate   SpaceOrganizationKind = "room.update"
	SpaceOrganizationAreaUpdate   SpaceOrganizationKind = "area.update"
	SpaceOrganizationDeviceRename SpaceOrganizationKind = "device.rename"
	SpaceOrganizationDeviceMove   SpaceOrganizationKind = "device.move"
	SpaceOrganizationGroupUpdate  SpaceOrganizationKind = "group.update"
)

type SpaceOrganizationCredentials struct {
	Authorization string
	ClientID      string
}

type SpaceOrganizationRequest struct {
	Kind           SpaceOrganizationKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    SpaceOrganizationCredentials
}

type SpaceOrganizationResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Name       string `json:"name,omitempty"`
	RoomID     string `json:"roomId,omitempty"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type SpaceOrganizationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewSpaceOrganizationClient(endpoint Endpoint, client *http.Client) SpaceOrganizationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SpaceOrganizationClient{endpoint: endpoint, client: client}
}

func (client SpaceOrganizationClient) Run(ctx context.Context, request SpaceOrganizationRequest) (SpaceOrganizationResult, error) {
	spec, ok := spaceOrganizationSpecs[request.Kind]
	if !ok {
		return SpaceOrganizationResult{}, fmt.Errorf("unsupported space organization kind %q", request.Kind)
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return SpaceOrganizationResult{}, fmt.Errorf("house id is required")
	}
	entityID := strings.TrimSpace(stringFromAny(request.Payload[spec.idKey]))
	if entityID == "" {
		return SpaceOrganizationResult{}, fmt.Errorf("%s id is required", spec.entityType)
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return SpaceOrganizationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	entities, preflightCalls, err := client.listEntities(ctx, houseID, credentials)
	apiCalls += preflightCalls
	if err != nil {
		return SpaceOrganizationResult{}, err
	}
	current, found := findSpaceEntity(entities, spec.entityType, entityID)
	if !found {
		return SpaceOrganizationResult{}, fmt.Errorf("%s %s not found before write", spec.entityType, entityID)
	}
	if spec.kind == SpaceOrganizationRoomUpdate && strings.TrimSpace(stringFromAny(request.Payload[spec.nameKey])) == "" {
		request.Payload[spec.nameKey] = current.Name
	}
	if err := validateSpaceOrganizationReferences(spec, request.Payload, entities); err != nil {
		return SpaceOrganizationResult{}, err
	}
	if err := client.write(ctx, spec, entityID, request.Payload, credentials); err != nil {
		return SpaceOrganizationResult{}, err
	}
	apiCalls++
	verified, verifyCalls, err := client.verifyAfterWrite(ctx, spec, houseID, entityID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return SpaceOrganizationResult{}, err
	}
	if verified.ID == "" {
		return SpaceOrganizationResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return SpaceOrganizationResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: string(request.Kind),
		EntityType: spec.entityType,
		EntityID:   verified.ID,
		Name:       verified.Name,
		RoomID:     verified.RoomID,
		Verified:   true,
		VerifiedBy: "entity.list",
		APICalls:   apiCalls,
	}, nil
}

type spaceOrganizationSpec struct {
	kind             SpaceOrganizationKind
	entityType       string
	idKey            string
	nameKey          string
	targetEntityType string
	targetIDKey      string
	targetOptional   bool
	pathPattern      string
	pathUsesHouseID  bool
}

var spaceOrganizationSpecs = map[SpaceOrganizationKind]spaceOrganizationSpec{
	SpaceOrganizationRoomRename: {
		kind:        SpaceOrganizationRoomRename,
		entityType:  "room",
		idKey:       "roomId",
		nameKey:     "name",
		pathPattern: "/v1/room/{id}/w/update",
	},
	SpaceOrganizationRoomUpdate: {
		kind:        SpaceOrganizationRoomUpdate,
		entityType:  "room",
		idKey:       "roomId",
		nameKey:     "name",
		pathPattern: "/v1/room/{id}/w/update",
	},
	SpaceOrganizationDeviceRename: {
		kind:        SpaceOrganizationDeviceRename,
		entityType:  "device",
		idKey:       "deviceId",
		nameKey:     "name",
		pathPattern: "/v1/device/{id}/w/update",
	},
	SpaceOrganizationDeviceMove: {
		kind:             SpaceOrganizationDeviceMove,
		entityType:       "device",
		idKey:            "deviceId",
		targetEntityType: "room",
		targetIDKey:      "roomId",
		pathPattern:      "/v1/device/{id}/w/update",
	},
	SpaceOrganizationAreaUpdate: {
		kind:            SpaceOrganizationAreaUpdate,
		entityType:      "area",
		idKey:           "areaId",
		nameKey:         "name",
		pathPattern:     "/v2/thing/manage/house/{houseId}/area/{id}/w/modify",
		pathUsesHouseID: true,
	},
	SpaceOrganizationGroupUpdate: {
		kind:             SpaceOrganizationGroupUpdate,
		entityType:       "group",
		idKey:            "groupId",
		nameKey:          "name",
		targetEntityType: "room",
		targetIDKey:      "roomId",
		targetOptional:   true,
		pathPattern:      "/v2/thing/manage/house/{houseId}/group/{id}/w/modify",
		pathUsesHouseID:  true,
	},
}

func (client SpaceOrganizationClient) write(ctx context.Context, spec spaceOrganizationSpec, entityID string, payload map[string]any, credentials requestCredentials) error {
	body := mapWithoutKeys(payload, spec.idKey)
	path := strings.ReplaceAll(spec.pathPattern, "{id}", entityID)
	if spec.pathUsesHouseID {
		path = strings.ReplaceAll(path, "{houseId}", pathSegment(stringFromAny(payload["houseId"])))
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, credentials)
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", spec.kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client SpaceOrganizationClient) verifyAfterWrite(ctx context.Context, spec spaceOrganizationSpec, houseID string, entityID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (EntitySummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entity, readCalls, err := client.findEntityAfterSpaceWrite(ctx, spec, houseID, entityID, credentials)
		calls += readCalls
		if err != nil {
			return EntitySummary{}, calls, err
		}
		if entityMatchesSpacePayload(entity, spec, payload) || attempt == attempts-1 {
			if entityMatchesSpacePayload(entity, spec, payload) {
				return entity, calls, nil
			}
			return EntitySummary{}, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return EntitySummary{}, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return EntitySummary{}, calls, nil
}

func (client SpaceOrganizationClient) findEntityAfterSpaceWrite(ctx context.Context, spec spaceOrganizationSpec, houseID string, entityID string, credentials requestCredentials) (EntitySummary, int, error) {
	if spec.kind == SpaceOrganizationGroupUpdate {
		entity, calls, err := client.findGroupDetailEntity(ctx, houseID, entityID, credentials)
		if err == nil && entity.ID != "" {
			return entity, calls, nil
		}
		if err != nil {
			fallback, fallbackCalls, fallbackErr := client.findEntity(ctx, houseID, spec.entityType, entityID, credentials)
			if fallbackErr != nil {
				return EntitySummary{}, calls + fallbackCalls, err
			}
			return fallback, calls + fallbackCalls, nil
		}
	}
	return client.findEntity(ctx, houseID, spec.entityType, entityID, credentials)
}

func (client SpaceOrganizationClient) findGroupDetailEntity(ctx context.Context, houseID string, groupID string, credentials requestCredentials) (EntitySummary, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/group/"+pathSegment(groupID)+"/r/info", nil, credentials)
	if err != nil {
		return EntitySummary{}, 1, err
	}
	if !isBusinessOK(response) {
		return EntitySummary{}, 1, metadataReadonlyBusinessError("group detail", response)
	}
	item := groupDetailItem(response["data"])
	if len(item) == 0 {
		return EntitySummary{}, 1, nil
	}
	return EntitySummary{
		Type:    "group",
		ID:      firstAnyString(item, "id", "groupId", "meshGroupId", "meshgroupId"),
		Name:    firstAnyString(item, "name", "groupName"),
		HouseID: firstNonEmpty(firstAnyString(item, "houseId"), houseID),
		RoomID:  firstAnyString(item, "roomId"),
	}, 1, nil
}

func groupDetailItem(data any) map[string]any {
	switch typed := data.(type) {
	case map[string]any:
		if detail, ok := typed["detail"].(map[string]any); ok {
			return detail
		}
		return typed
	default:
		return nil
	}
}

func (client SpaceOrganizationClient) findEntity(ctx context.Context, houseID string, entityType string, entityID string, credentials requestCredentials) (EntitySummary, int, error) {
	result, calls, err := client.listEntities(ctx, houseID, credentials)
	if err != nil {
		return EntitySummary{}, calls, err
	}
	entity, _ := findSpaceEntity(result, entityType, entityID)
	return entity, calls, nil
}

func (client SpaceOrganizationClient) listEntities(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntityListResult{}, result.APICalls, err
	}
	return result, result.APICalls, nil
}

func findSpaceEntity(result EntityListResult, entityType string, entityID string) (EntitySummary, bool) {
	for _, entity := range result.Entities {
		if entity.Type == entityType && entity.ID == entityID {
			return entity, true
		}
	}
	return EntitySummary{}, false
}

func validateSpaceOrganizationReferences(spec spaceOrganizationSpec, payload map[string]any, entities EntityListResult) error {
	if spec.targetEntityType != "" {
		targetID := strings.TrimSpace(stringFromAny(payload[spec.targetIDKey]))
		if targetID == "" {
			if spec.targetOptional {
				return nil
			}
			return fmt.Errorf("%s target %s is required", spec.kind, spec.targetIDKey)
		}
		if _, ok := findSpaceEntity(entities, spec.targetEntityType, targetID); !ok {
			return fmt.Errorf("%s %s not found before write", spec.targetEntityType, targetID)
		}
	}
	if spec.kind == SpaceOrganizationRoomUpdate {
		if gatewayID := strings.TrimSpace(stringFromAny(payload["gatewayDeviceId"])); gatewayID != "" {
			if _, ok := findSpaceEntity(entities, "device", gatewayID); !ok {
				return fmt.Errorf("device %s not found before write", gatewayID)
			}
		}
		for _, key := range []string{"gatewayIds", "defaultGatewayIds"} {
			for _, gatewayID := range spaceOrganizationStringIDs(payload[key]) {
				if _, ok := findSpaceEntity(entities, "device", gatewayID); !ok {
					return fmt.Errorf("device %s not found before write", gatewayID)
				}
			}
		}
	}
	if spec.kind != SpaceOrganizationAreaUpdate {
		return nil
	}
	areaID := strings.TrimSpace(stringFromAny(payload[spec.idKey]))
	if parentID := strings.TrimSpace(stringFromAny(payload["parentId"])); parentID != "" {
		if parentID == areaID {
			return fmt.Errorf("area parentId cannot reference itself")
		}
		if _, ok := findSpaceEntity(entities, "area", parentID); !ok {
			return fmt.Errorf("area parent %s not found before write", parentID)
		}
	}
	roomIDs, ok := payload["roomIds"].([]any)
	if !ok {
		return nil
	}
	if len(roomIDs) > 50 {
		return fmt.Errorf("area roomIds limit exceeded")
	}
	for _, roomIDValue := range roomIDs {
		roomID := strings.TrimSpace(stringFromAny(roomIDValue))
		if roomID == "" {
			return fmt.Errorf("area roomId is required")
		}
		if _, ok := findSpaceEntity(entities, "room", roomID); !ok {
			return fmt.Errorf("room %s not found before write", roomID)
		}
	}
	return nil
}

func spaceOrganizationStringIDs(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if id := strings.TrimSpace(stringFromAny(item)); id != "" {
				result = append(result, id)
			}
		}
		return result
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if id := strings.TrimSpace(item); id != "" {
				result = append(result, id)
			}
		}
		return result
	default:
		if id := strings.TrimSpace(stringFromAny(value)); id != "" {
			return []string{id}
		}
		return nil
	}
}

func entityMatchesSpacePayload(entity EntitySummary, spec spaceOrganizationSpec, payload map[string]any) bool {
	if entity.ID == "" {
		return false
	}
	if spec.nameKey != "" {
		expectedName := strings.TrimSpace(stringFromAny(payload[spec.nameKey]))
		if expectedName != "" && entity.Name != expectedName {
			return false
		}
	}
	if spec.targetIDKey != "" {
		expectedRoomID := strings.TrimSpace(stringFromAny(payload[spec.targetIDKey]))
		if expectedRoomID != "" && entity.RoomID != expectedRoomID {
			return false
		}
	}
	return true
}
