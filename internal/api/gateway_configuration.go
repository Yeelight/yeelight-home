package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type GatewayConfigurationCredentials struct {
	Authorization string
	ClientID      string
}

type GatewayConfigurationRequest struct {
	HouseID        string
	GatewayID      string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    GatewayConfigurationCredentials
}

type GatewayConfigurationResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	GatewayID  string `json:"gatewayId"`
	Name       string `json:"name,omitempty"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type GatewayConfigurationClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewGatewayConfigurationClient(endpoint Endpoint, client *http.Client) GatewayConfigurationClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return GatewayConfigurationClient{endpoint: endpoint, client: client}
}

func (client GatewayConfigurationClient) Run(ctx context.Context, request GatewayConfigurationRequest) (GatewayConfigurationResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	gatewayID := strings.TrimSpace(request.GatewayID)
	if houseID == "" {
		return GatewayConfigurationResult{}, fmt.Errorf("house id is required")
	}
	if gatewayID == "" {
		return GatewayConfigurationResult{}, fmt.Errorf("gateway id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return GatewayConfigurationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	current, calls, err := client.readGateway(ctx, houseID, gatewayID, credentials)
	apiCalls += calls
	if err != nil {
		return GatewayConfigurationResult{}, err
	}
	if firstAnyString(current, semantic.FieldID, semantic.FieldGatewayID, semantic.FieldDeviceID) == "" {
		return GatewayConfigurationResult{}, fmt.Errorf("gateway %s not found before write", gatewayID)
	}
	calls, err = client.validateRoomReferences(ctx, houseID, request.Payload, credentials)
	apiCalls += calls
	if err != nil {
		return GatewayConfigurationResult{}, err
	}
	if err := client.write(ctx, houseID, gatewayID, request.Payload, credentials); err != nil {
		return GatewayConfigurationResult{}, err
	}
	apiCalls++
	verified, calls, err := client.verifyAfterWrite(ctx, houseID, gatewayID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return GatewayConfigurationResult{}, err
	}
	if !verified {
		return GatewayConfigurationResult{}, fmt.Errorf("gateway.configure write verification mismatch")
	}
	name := strings.TrimSpace(stringFromAny(request.Payload[semantic.FieldName]))
	if name == "" {
		name = firstAnyString(current, semantic.FieldName, semantic.FieldGatewayName, semantic.FieldDeviceName)
	}
	return GatewayConfigurationResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "gateway.configure",
		GatewayID:  gatewayID,
		Name:       name,
		Verified:   true,
		VerifiedBy: "gateway.detail.get",
		APICalls:   apiCalls,
	}, nil
}

func (client GatewayConfigurationClient) readGateway(ctx context.Context, houseID string, gatewayID string, credentials requestCredentials) (map[string]any, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/info", nil, credentials)
	if err != nil {
		return nil, 1, err
	}
	if !isBusinessOK(response) {
		return nil, 1, fmt.Errorf("gateway.detail.get returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	data, _ := response["data"].(map[string]any)
	return data, 1, nil
}

func (client GatewayConfigurationClient) validateRoomReferences(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	roomIDs := homeSpaceIDList(payload[semantic.FieldRoomIDs])
	if len(roomIDs) == 0 {
		return 0, nil
	}
	entities, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return entities.APICalls, err
	}
	for _, roomID := range roomIDs {
		if !gatewayEntityListHas(entities, "room", roomID) {
			return entities.APICalls, fmt.Errorf("invalid gateway room reference %s", roomID)
		}
	}
	return entities.APICalls, nil
}

func gatewayEntityListHas(entities EntityListResult, entityType string, entityID string) bool {
	for _, entity := range entities.Entities {
		if entity.Type == entityType && entity.ID == entityID {
			return true
		}
	}
	return false
}

func (client GatewayConfigurationClient) write(ctx context.Context, houseID string, gatewayID string, payload map[string]any, credentials requestCredentials) error {
	body := mapWithoutKeys(payload, semantic.FieldHouseID, semantic.FieldGatewayID, semantic.FieldID, semantic.FieldCapability)
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/w/modify", body, credentials)
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("gateway.configure returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client GatewayConfigurationClient) verifyAfterWrite(ctx context.Context, houseID string, gatewayID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		detail, readCalls, err := client.readGateway(ctx, houseID, gatewayID, credentials)
		calls += readCalls
		if err != nil {
			return false, calls, err
		}
		if gatewayDetailMatchesPayload(detail, payload) {
			return true, calls, nil
		}
		if attempt == attempts-1 {
			return false, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, calls, nil
}

func gatewayDetailMatchesPayload(detail map[string]any, payload map[string]any) bool {
	if len(detail) == 0 {
		return false
	}
	if expected := strings.TrimSpace(stringFromAny(payload[semantic.FieldName])); expected != "" {
		if firstAnyString(detail, semantic.FieldName, semantic.FieldGatewayName, semantic.FieldDeviceName) != expected {
			return false
		}
	}
	return true
}
