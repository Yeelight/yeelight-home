package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type SpaceBatchOrganizationKind string

const (
	SpaceBatchDeviceMoveRoom SpaceBatchOrganizationKind = "device.move_room.batch"
)

type SpaceBatchOrganizationRequest struct {
	Kind           SpaceBatchOrganizationKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    SpaceOrganizationCredentials
}

type SpaceBatchOrganizationResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	Capability       string           `json:"capability"`
	ItemCount        int              `json:"itemCount"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	VerifiedEntities EntityListResult `json:"-"`
}

type SpaceBatchOrganizationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewSpaceBatchOrganizationClient(endpoint Endpoint, client *http.Client) SpaceBatchOrganizationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return SpaceBatchOrganizationClient{endpoint: endpoint, client: client}
}

func (client SpaceBatchOrganizationClient) Run(ctx context.Context, request SpaceBatchOrganizationRequest) (SpaceBatchOrganizationResult, error) {
	if request.Kind != SpaceBatchDeviceMoveRoom {
		return SpaceBatchOrganizationResult{}, fmt.Errorf("unsupported space batch organization kind %q", request.Kind)
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return SpaceBatchOrganizationResult{}, fmt.Errorf("house id is required")
	}
	items, err := deviceRoomBatchItems(request.Payload)
	if err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return SpaceBatchOrganizationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	entities, calls, err := client.listEntities(ctx, houseID, credentials)
	apiCalls += calls
	if err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	if err := validateDeviceRoomBatchItems(items, entities); err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	if err := client.writeDeviceRoomBatch(ctx, houseID, items, credentials); err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	apiCalls++
	ok, verifiedEntities, calls, err := client.verifyDeviceRoomBatch(ctx, houseID, items, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	if !ok {
		return SpaceBatchOrganizationResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	if err := detectDeviceMoveRoomBatchSideEffects(entities, verifiedEntities); err != nil {
		return SpaceBatchOrganizationResult{}, err
	}
	return SpaceBatchOrganizationResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		Capability:       string(request.Kind),
		ItemCount:        len(items),
		Verified:         true,
		VerifiedBy:       "entity.list",
		APICalls:         apiCalls,
		VerifiedEntities: verifiedEntities,
	}, nil
}

func (client SpaceBatchOrganizationClient) listEntities(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
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

func (client SpaceBatchOrganizationClient) writeDeviceRoomBatch(ctx context.Context, houseID string, items map[string]string, credentials requestCredentials) error {
	body := map[string]any{
		semantic.FieldHouseID: requestNumberOrStringForAPI(houseID),
		semantic.FieldItems:   items,
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/device/room/w/batch-modify", body, credentials)
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("device.move_room.batch returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client SpaceBatchOrganizationClient) verifyDeviceRoomBatch(ctx context.Context, houseID string, items map[string]string, credentials requestCredentials, attempts int, interval time.Duration) (bool, EntityListResult, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entities, readCalls, err := client.listEntities(ctx, houseID, credentials)
		calls += readCalls
		if err != nil {
			return false, entities, calls, err
		}
		if deviceRoomBatchMatches(items, entities) || attempt == attempts-1 {
			return deviceRoomBatchMatches(items, entities), entities, calls, nil
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

func deviceRoomBatchItems(payload map[string]any) (map[string]string, error) {
	raw, ok := payload[semantic.FieldItems]
	if !ok {
		return nil, fmt.Errorf("device room batch items are required")
	}
	result := map[string]string{}
	switch typed := raw.(type) {
	case map[string]any:
		for deviceID, roomValue := range typed {
			roomID := strings.TrimSpace(stringFromAny(roomValue))
			if strings.TrimSpace(deviceID) == "" || roomID == "" {
				return nil, fmt.Errorf("invalid device room batch item")
			}
			result[strings.TrimSpace(deviceID)] = roomID
		}
	case []any:
		for _, itemValue := range typed {
			item, ok := itemValue.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid device room batch item")
			}
			deviceID := strings.TrimSpace(firstNonEmpty(stringFromAny(item[semantic.FieldDeviceID]), stringFromAny(item[semantic.FieldID]), stringFromAny(item[semantic.FieldEntityID])))
			roomID := strings.TrimSpace(firstNonEmpty(stringFromAny(item[semantic.FieldRoomID]), stringFromAny(item[semantic.FieldTargetRoomID]), stringFromAny(item[semantic.FieldTargetID])))
			if deviceID == "" || roomID == "" {
				return nil, fmt.Errorf("invalid device room batch item")
			}
			result[deviceID] = roomID
		}
	default:
		return nil, fmt.Errorf("invalid device room batch items")
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("device room batch items are required")
	}
	if len(result) > 20 {
		return nil, fmt.Errorf("device room batch limit exceeded")
	}
	return result, nil
}

func validateDeviceRoomBatchItems(items map[string]string, entities EntityListResult) error {
	for deviceID, roomID := range items {
		if _, ok := findSpaceEntity(entities, "device", deviceID); !ok {
			return fmt.Errorf("device %s not found before write", deviceID)
		}
		if _, ok := findSpaceEntity(entities, "room", roomID); !ok {
			return fmt.Errorf("room %s not found before write", roomID)
		}
	}
	return nil
}

func deviceRoomBatchMatches(items map[string]string, entities EntityListResult) bool {
	for deviceID, roomID := range items {
		device, ok := findSpaceEntity(entities, "device", deviceID)
		if !ok || device.RoomID != roomID {
			return false
		}
	}
	return true
}

func detectDeviceMoveRoomBatchSideEffects(before EntityListResult, after EntityListResult) error {
	drops := []string{}
	for _, entityType := range []string{"group", "scene", "automation"} {
		beforeCount := entityCount(before, entityType)
		afterCount := entityCount(after, entityType)
		if afterCount < beforeCount {
			drops = append(drops, fmt.Sprintf("%s %d->%d", entityType, beforeCount, afterCount))
		}
	}
	if len(drops) == 0 {
		return nil
	}
	return fmt.Errorf("device.move_room.batch caused dependent entity count drop: %s", strings.Join(drops, ", "))
}

func entityCount(result EntityListResult, entityType string) int {
	if result.Counts != nil {
		return result.Counts[entityType]
	}
	count := 0
	for _, entity := range result.Entities {
		if entity.Type == entityType {
			count++
		}
	}
	return count
}
