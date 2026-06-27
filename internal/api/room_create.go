package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RoomCreateCredentials struct {
	Authorization string
	ClientID      string
}

type RoomCreateRequest struct {
	HouseID        string
	Name           string
	Description    string
	Icon           string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    RoomCreateCredentials
}

type RoomCreateResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	RoomID     string `json:"roomId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type RoomCreateClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewRoomCreateClient(endpoint Endpoint, client *http.Client) RoomCreateClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return RoomCreateClient{endpoint: endpoint, client: client}
}

func (client RoomCreateClient) Run(ctx context.Context, request RoomCreateRequest) (RoomCreateResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	name := strings.TrimSpace(request.Name)
	if houseID == "" {
		return RoomCreateResult{}, fmt.Errorf("house id is required")
	}
	if name == "" {
		return RoomCreateResult{}, fmt.Errorf("room name is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return RoomCreateResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	verifyCalls, err := client.verifyHouseScopedEntityList(ctx, houseID, credentials)
	if err != nil {
		return RoomCreateResult{}, fmt.Errorf("verify house before room create: %w", err)
	}
	apiCalls += verifyCalls
	existing, findCalls, err := client.findRoomByNameWithCallCount(ctx, houseID, name, credentials)
	apiCalls += findCalls
	if err != nil {
		return RoomCreateResult{}, err
	}
	if existing.ID != "" {
		return RoomCreateResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			RoomID:     existing.ID,
			Name:       existing.Name,
			Verified:   true,
			VerifiedBy: existing.Source,
			APICalls:   apiCalls,
		}, nil
	}
	created, err := client.createRoom(ctx, RoomCreateRequest{
		HouseID:     houseID,
		Name:        name,
		Description: request.Description,
		Icon:        request.Icon,
	}, credentials)
	apiCalls++
	if err != nil {
		return RoomCreateResult{}, err
	}
	verified, verifyCalls, err := client.verifyRoomByNameWithCallCount(ctx, houseID, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return RoomCreateResult{}, err
	}
	if verified.ID == "" {
		return RoomCreateResult{}, fmt.Errorf("unknown write result: created room was not found during verification; createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
	}
	return RoomCreateResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		RoomID:     verified.ID,
		Name:       verified.Name,
		Created:    true,
		Verified:   true,
		VerifiedBy: verified.Source,
		APICalls:   apiCalls,
	}, nil
}

func (client RoomCreateClient) verifyHouseScopedEntityList(ctx context.Context, houseID string, credentials requestCredentials) (int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return result.APICalls, err
	}
	if result.APICalls > 0 {
		return result.APICalls, nil
	}
	return houseScopedEntityListCallCount, nil
}

func (client RoomCreateClient) verifyRoomByNameWithCallCount(ctx context.Context, houseID string, name string, credentials requestCredentials, attempts int, interval time.Duration) (roomSummary, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		room, findCalls, err := client.findRoomByNameWithCallCount(ctx, houseID, name, credentials)
		calls += findCalls
		if err != nil || room.ID != "" || attempt == attempts-1 {
			return room, calls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return roomSummary{}, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return roomSummary{}, calls, nil
}

func (client RoomCreateClient) findRoomByName(ctx context.Context, houseID string, name string, credentials requestCredentials) (roomSummary, error) {
	room, _, err := client.findRoomByNameWithCallCount(ctx, houseID, name, credentials)
	return room, err
}

func (client RoomCreateClient) findRoomByNameWithCallCount(ctx context.Context, houseID string, name string, credentials requestCredentials) (roomSummary, int, error) {
	entities, apiCalls, err := listHouseScopedEntityPages(ctx, client.client, client.endpoint, houseID, credentials, entityListCall{
		entityType:  "room",
		method:      http.MethodGet,
		pathPattern: fmt.Sprintf("/v2/thing/manage/house/%s/room/r/info/{pageNo}/%d", houseID, entityListPageSize),
	})
	if err != nil {
		return roomSummary{}, apiCalls, err
	}
	for _, entity := range entities {
		room := roomSummary{
			ID:     entity.ID,
			Name:   entity.Name,
			Source: "room_list",
		}
		if room.Name == name {
			return room, apiCalls, nil
		}
	}
	return roomSummary{}, apiCalls, nil
}

func (client RoomCreateClient) createRoom(ctx context.Context, request RoomCreateRequest, credentials requestCredentials) (writeProbeResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	body, err := BuildRoomCreatePayload(houseID, request.Name, request.Description, request.Icon)
	if err != nil {
		return writeProbeResult{}, err
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+houseID+"/room/w/create", body, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("room create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}

func BuildRoomCreatePayload(houseID string, name string, description string, icon string) (map[string]any, error) {
	trimmedHouseID := strings.TrimSpace(houseID)
	parsedHouseID, err := strconv.ParseInt(trimmedHouseID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("house id must be numeric for room create")
	}
	payload := map[string]any{
		"houseId": float64(parsedHouseID),
		"name":    strings.TrimSpace(name),
	}
	if value := strings.TrimSpace(description); value != "" {
		payload["desc"] = value
	}
	if value := strings.TrimSpace(icon); value != "" {
		payload["icon"] = value
	}
	return payload, nil
}
