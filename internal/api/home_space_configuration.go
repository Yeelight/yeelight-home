package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type HomeSpaceConfigurationKind string

const (
	HomeSpaceHomeUpdate        HomeSpaceConfigurationKind = "home.update"
	HomeSpaceRoomBatchCreate   HomeSpaceConfigurationKind = "room.batch_create"
	HomeSpaceRoomBatchUpdate   HomeSpaceConfigurationKind = "room.batch_update"
	HomeSpaceRoomAreaConfigure HomeSpaceConfigurationKind = "room.area.configure"
)

type HomeSpaceConfigurationCredentials struct {
	Authorization string
	ClientID      string
}

type HomeSpaceConfigurationRequest struct {
	Kind           HomeSpaceConfigurationKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    HomeSpaceConfigurationCredentials
}

type HomeSpaceConfigurationResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	Capability       string           `json:"capability"`
	ItemCount        int              `json:"itemCount,omitempty"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	VerifiedEntities EntityListResult `json:"-"`
}

type HomeSpaceConfigurationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeSpaceConfigurationClient(endpoint Endpoint, client *http.Client) HomeSpaceConfigurationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeSpaceConfigurationClient{endpoint: endpoint, client: client}
}

func (client HomeSpaceConfigurationClient) Run(ctx context.Context, request HomeSpaceConfigurationRequest) (HomeSpaceConfigurationResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return HomeSpaceConfigurationResult{}, fmt.Errorf("house id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return HomeSpaceConfigurationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	calls, err := client.preflight(ctx, request.Kind, houseID, request.Payload, credentials)
	apiCalls += calls
	if err != nil {
		return HomeSpaceConfigurationResult{}, err
	}
	calls, err = client.write(ctx, request.Kind, houseID, request.Payload, credentials)
	apiCalls += calls
	if err != nil {
		return HomeSpaceConfigurationResult{}, err
	}
	ok, verifiedEntities, calls, err := client.verifyAfterWrite(ctx, request.Kind, houseID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return HomeSpaceConfigurationResult{}, err
	}
	if !ok {
		return HomeSpaceConfigurationResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return HomeSpaceConfigurationResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		Capability:       string(request.Kind),
		ItemCount:        homeSpaceConfigurationItemCount(request.Kind, request.Payload),
		Verified:         true,
		VerifiedBy:       homeSpaceConfigurationVerifyWith(request.Kind),
		APICalls:         apiCalls,
		VerifiedEntities: verifiedEntities,
	}, nil
}

func (client HomeSpaceConfigurationClient) preflight(ctx context.Context, kind HomeSpaceConfigurationKind, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	switch kind {
	case HomeSpaceHomeUpdate:
		_, calls, err := client.readHomeDetail(ctx, houseID, credentials)
		return calls, err
	case HomeSpaceRoomBatchCreate, HomeSpaceRoomBatchUpdate, HomeSpaceRoomAreaConfigure:
		entities, calls, err := client.listEntities(ctx, houseID, credentials)
		if err != nil {
			return calls, err
		}
		return calls, validateHomeSpaceConfigurationReferences(kind, payload, entities)
	default:
		return 0, fmt.Errorf("unsupported home space configuration kind %q", kind)
	}
}

func (client HomeSpaceConfigurationClient) write(ctx context.Context, kind HomeSpaceConfigurationKind, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	switch kind {
	case HomeSpaceHomeUpdate:
		body := mapWithoutKeys(payload, semantic.FieldHouseID)
		body[semantic.FieldID] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/w/modify", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("home.update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeSpaceRoomBatchCreate:
		rooms, err := homeSpaceRoomItems(payload)
		if err != nil {
			return 0, err
		}
		calls := 0
		for _, room := range rooms {
			body := mapWithoutKeys(room, semantic.FieldRoomID, semantic.FieldID)
			response, err := callJSONBody(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/room/w/create", body, credentials)
			calls++
			if err != nil {
				return calls, err
			}
			if !isBusinessOK(response) {
				return calls, fmt.Errorf("room.create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
			}
		}
		return calls, nil
	case HomeSpaceRoomBatchUpdate:
		rooms, err := homeSpaceRoomItems(payload)
		if err != nil {
			return 0, err
		}
		response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/room/w/batchupdate", map[string]any{semantic.FieldRooms: rooms}, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("room.batch_update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	case HomeSpaceRoomAreaConfigure:
		roomID := strings.TrimSpace(stringFromAny(payload[semantic.FieldRoomID]))
		body := mapWithoutKeys(payload, semantic.FieldRoomID)
		body[semantic.FieldHouseID] = requestNumberOrStringForAPI(houseID)
		body[semantic.FieldID] = requestNumberOrStringForAPI(roomID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/room/"+pathSegment(roomID)+"/w/areas", body, credentials)
		if err != nil {
			return 1, err
		}
		if !isBusinessOK(response) {
			return 1, fmt.Errorf("room.area.configure returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported home space configuration kind %q", kind)
	}
}

func (client HomeSpaceConfigurationClient) verifyAfterWrite(ctx context.Context, kind HomeSpaceConfigurationKind, houseID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (bool, EntityListResult, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		var ok bool
		var entities EntityListResult
		var readCalls int
		var err error
		switch kind {
		case HomeSpaceHomeUpdate:
			ok, readCalls, err = client.verifyHomeUpdate(ctx, houseID, payload, credentials)
		case HomeSpaceRoomBatchCreate, HomeSpaceRoomBatchUpdate:
			ok, entities, readCalls, err = client.verifyRooms(ctx, houseID, payload, credentials)
		case HomeSpaceRoomAreaConfigure:
			ok, entities, readCalls, err = client.verifyRoomAreaAccessible(ctx, houseID, payload, credentials)
		default:
			return false, EntityListResult{}, calls, fmt.Errorf("unsupported home space configuration kind %q", kind)
		}
		calls += readCalls
		if err != nil || ok || attempt == attempts-1 {
			return ok, entities, calls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, entities, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, EntityListResult{}, calls, nil
}

func (client HomeSpaceConfigurationClient) listEntities(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
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

func (client HomeSpaceConfigurationClient) readHomeDetail(ctx context.Context, houseID string, credentials requestCredentials) (map[string]any, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/r/info", nil, credentials)
	if err != nil {
		return nil, 1, err
	}
	if !isBusinessOK(response) {
		return nil, 1, fmt.Errorf("home.detail.get returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	data, _ := response["data"].(map[string]any)
	return data, 1, nil
}

func (client HomeSpaceConfigurationClient) verifyHomeUpdate(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	detail, calls, err := client.readHomeDetail(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	if expectedName := strings.TrimSpace(stringFromAny(payload[semantic.FieldName])); expectedName != "" {
		return firstAnyString(detail, semantic.FieldName, semantic.FieldHouseName) == expectedName, calls, nil
	}
	return firstAnyString(detail, semantic.FieldID, semantic.FieldHouseID) == houseID || len(detail) > 0, calls, nil
}

func (client HomeSpaceConfigurationClient) verifyRooms(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, EntityListResult, int, error) {
	entities, calls, err := client.listEntities(ctx, houseID, credentials)
	if err != nil {
		return false, entities, calls, err
	}
	rooms, err := homeSpaceRoomItems(payload)
	if err != nil {
		return false, entities, calls, err
	}
	for _, room := range rooms {
		if !homeSpaceRoomMatches(room, entities) {
			return false, entities, calls, nil
		}
	}
	return true, entities, calls, nil
}

func (client HomeSpaceConfigurationClient) verifyRoomAreaAccessible(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, EntityListResult, int, error) {
	entities, calls, err := client.listEntities(ctx, houseID, credentials)
	if err != nil {
		return false, entities, calls, err
	}
	roomID := strings.TrimSpace(stringFromAny(payload[semantic.FieldRoomID]))
	if _, ok := findSpaceEntity(entities, "room", roomID); !ok {
		return false, entities, calls, nil
	}
	for _, areaID := range append(homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs)]), homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs)])...) {
		if _, ok := findSpaceEntity(entities, "area", areaID); !ok {
			return false, entities, calls, nil
		}
	}
	return true, entities, calls, nil
}

func validateHomeSpaceConfigurationReferences(kind HomeSpaceConfigurationKind, payload map[string]any, entities EntityListResult) error {
	switch kind {
	case HomeSpaceRoomBatchCreate:
		rooms, err := homeSpaceRoomItems(payload)
		if err != nil {
			return err
		}
		seenNames := map[string]bool{}
		for _, room := range rooms {
			name := strings.TrimSpace(stringFromAny(room[semantic.FieldName]))
			if name == "" {
				return fmt.Errorf("room name is required")
			}
			if seenNames[name] {
				return fmt.Errorf("duplicate room name")
			}
			seenNames[name] = true
			for _, entity := range entities.Entities {
				if entity.Type == "room" && entity.Name == name {
					return fmt.Errorf("room name already exists")
				}
			}
			if err := validateHomeSpaceRoomGatewayReferences(room, entities); err != nil {
				return err
			}
		}
	case HomeSpaceRoomBatchUpdate:
		rooms, err := homeSpaceRoomItems(payload)
		if err != nil {
			return err
		}
		seenIDs := map[string]bool{}
		for _, room := range rooms {
			roomID := strings.TrimSpace(stringFromAny(firstNonNil(room[semantic.FieldRoomID], room[semantic.FieldID])))
			if roomID == "" {
				return fmt.Errorf("room id is required")
			}
			current, ok := findSpaceEntity(entities, "room", roomID)
			if !ok {
				return fmt.Errorf("room %s not found before write", roomID)
			}
			if seenIDs[roomID] {
				return fmt.Errorf("duplicate room target")
			}
			seenIDs[roomID] = true
			if strings.TrimSpace(stringFromAny(room[semantic.FieldName])) == "" {
				room[semantic.FieldName] = current.Name
			}
			if err := validateHomeSpaceRoomGatewayReferences(room, entities); err != nil {
				return err
			}
		}
	case HomeSpaceRoomAreaConfigure:
		roomID := strings.TrimSpace(stringFromAny(payload[semantic.FieldRoomID]))
		if roomID == "" {
			return fmt.Errorf("room id is required")
		}
		if _, ok := findSpaceEntity(entities, "room", roomID); !ok {
			return fmt.Errorf("room %s not found before write", roomID)
		}
		areaIDs := append(homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs)]), homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs)])...)
		if len(areaIDs) == 0 {
			return fmt.Errorf("room area delta is required")
		}
		seen := map[string]bool{}
		for _, areaID := range areaIDs {
			if seen[areaID] {
				return fmt.Errorf("duplicate area target")
			}
			seen[areaID] = true
			if _, ok := findSpaceEntity(entities, "area", areaID); !ok {
				return fmt.Errorf("area %s not found before write", areaID)
			}
		}
	}
	return nil
}

func validateHomeSpaceRoomGatewayReferences(room map[string]any, entities EntityListResult) error {
	if gatewayID := strings.TrimSpace(stringFromAny(room[semantic.FieldGatewayDeviceID])); gatewayID != "" {
		if _, ok := findSpaceEntity(entities, "device", gatewayID); !ok {
			return fmt.Errorf("device %s not found before write", gatewayID)
		}
	}
	for _, key := range []string{semantic.FieldGatewayIDs, semantic.FieldGatewayDeviceIDs, semantic.FieldDefaultGatewayIDs} {
		for _, gatewayID := range homeSpaceIDList(room[key]) {
			if _, ok := findSpaceEntity(entities, "device", gatewayID); !ok {
				return fmt.Errorf("device %s not found before write", gatewayID)
			}
		}
	}
	return nil
}

func homeSpaceRoomItems(payload map[string]any) ([]map[string]any, error) {
	rawItems, ok := payload[semantic.FieldRooms]
	if !ok {
		rawItems = payload[semantic.FieldItems]
	}
	items, ok := rawItems.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("rooms are required")
	}
	if len(items) > 20 {
		return nil, fmt.Errorf("room batch limit exceeded")
	}
	result := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		room, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("room item must be an object")
		}
		result = append(result, room)
	}
	return result, nil
}

func homeSpaceRoomMatches(room map[string]any, entities EntityListResult) bool {
	roomID := strings.TrimSpace(stringFromAny(firstNonNil(room[semantic.FieldRoomID], room[semantic.FieldID])))
	expectedName := strings.TrimSpace(stringFromAny(room[semantic.FieldName]))
	for _, entity := range entities.Entities {
		if entity.Type != "room" {
			continue
		}
		if roomID != "" && entity.ID != roomID {
			continue
		}
		if expectedName != "" && entity.Name != expectedName {
			continue
		}
		return true
	}
	return false
}

func homeSpaceIDList(value any) []string {
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

func homeSpaceConfigurationItemCount(kind HomeSpaceConfigurationKind, payload map[string]any) int {
	switch kind {
	case HomeSpaceRoomBatchCreate, HomeSpaceRoomBatchUpdate:
		rooms, err := homeSpaceRoomItems(payload)
		if err == nil {
			return len(rooms)
		}
	case HomeSpaceRoomAreaConfigure:
		return len(homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldAddAreaIDs)])) + len(homeSpaceIDList(payload[semantic.InternalField(semantic.DomainCommon, semantic.FieldRemoveAreaIDs)]))
	case HomeSpaceHomeUpdate:
		return 1
	}
	return 0
}

func homeSpaceConfigurationVerifyWith(kind HomeSpaceConfigurationKind) string {
	switch kind {
	case HomeSpaceHomeUpdate:
		return "home.detail.get"
	case HomeSpaceRoomBatchCreate, HomeSpaceRoomBatchUpdate, HomeSpaceRoomAreaConfigure:
		return "entity.list"
	default:
		return "write_after_read"
	}
}
