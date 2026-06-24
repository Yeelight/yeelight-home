package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DestructiveDeleteKind string

const (
	DestructiveDeleteDevice  DestructiveDeleteKind = "device.remove"
	DestructiveDeleteGateway DestructiveDeleteKind = "gateway.delete"
	DestructiveDeleteHome    DestructiveDeleteKind = "home.delete"
)

type DestructiveDeleteCredentials struct {
	Authorization string
	ClientID      string
}

type DestructiveDeleteRequest struct {
	Kind           DestructiveDeleteKind
	HouseID        string
	EntityID       string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    DestructiveDeleteCredentials
}

type DestructiveDeleteResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId,omitempty"`
	Capability string `json:"capability"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Name       string `json:"name,omitempty"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
}

type DestructiveDeleteClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewDestructiveDeleteClient(endpoint Endpoint, client *http.Client) DestructiveDeleteClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DestructiveDeleteClient{endpoint: endpoint, client: client}
}

func (client DestructiveDeleteClient) Run(ctx context.Context, request DestructiveDeleteRequest) (DestructiveDeleteResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	entityID := strings.TrimSpace(request.EntityID)
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DestructiveDeleteResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	var target EntitySummary
	var err error
	var calls int
	switch request.Kind {
	case DestructiveDeleteDevice:
		if houseID == "" {
			return DestructiveDeleteResult{}, fmt.Errorf("house id is required")
		}
		target, calls, err = client.findEntity(ctx, houseID, "device", entityID, credentials)
	case DestructiveDeleteGateway:
		if houseID == "" {
			return DestructiveDeleteResult{}, fmt.Errorf("house id is required")
		}
		target, calls, err = client.findGateway(ctx, houseID, entityID, credentials)
	case DestructiveDeleteHome:
		target, calls, err = client.findHome(ctx, entityID, credentials)
	default:
		return DestructiveDeleteResult{}, fmt.Errorf("unsupported destructive delete kind %q", request.Kind)
	}
	apiCalls += calls
	if err != nil {
		return DestructiveDeleteResult{}, err
	}
	if target.ID == "" {
		return DestructiveDeleteResult{}, fmt.Errorf("%s %s not found before delete", destructiveEntityType(request.Kind), entityID)
	}
	calls, err = client.writeDelete(ctx, request.Kind, firstNonEmpty(houseID, target.HouseID, target.ID), target.ID, credentials)
	apiCalls += calls
	if err != nil {
		return DestructiveDeleteResult{}, err
	}
	deleted, calls, err := client.verifyDeleted(ctx, request.Kind, firstNonEmpty(houseID, target.HouseID, target.ID), target.ID, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return DestructiveDeleteResult{}, err
	}
	if !deleted {
		return DestructiveDeleteResult{}, fmt.Errorf("%s delete verification mismatch", request.Kind)
	}
	return DestructiveDeleteResult{
		Region:     client.endpoint.Region,
		HouseID:    firstNonEmpty(houseID, target.HouseID),
		Capability: string(request.Kind),
		EntityType: destructiveEntityType(request.Kind),
		EntityID:   target.ID,
		Name:       target.Name,
		Verified:   true,
		VerifiedBy: destructiveVerifyWith(request.Kind),
		APICalls:   apiCalls,
	}, nil
}

func (client DestructiveDeleteClient) ProbeGateway(ctx context.Context, houseID string, gatewayID string, credentials DestructiveDeleteCredentials) (EntitySummary, int, error) {
	return client.findGateway(ctx, houseID, gatewayID, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
}

func (client DestructiveDeleteClient) writeDelete(ctx context.Context, kind DestructiveDeleteKind, houseID string, entityID string, credentials requestCredentials) (int, error) {
	var method string
	var path string
	var body map[string]any
	switch kind {
	case DestructiveDeleteDevice:
		method = http.MethodDelete
		path = "/v2/thing/manage/house/" + pathSegment(houseID) + "/device/" + pathSegment(entityID) + "/w/info"
	case DestructiveDeleteGateway:
		method = http.MethodDelete
		path = "/v2/thing/manage/house/" + pathSegment(houseID) + "/gateway/" + pathSegment(entityID) + "/w/info"
	case DestructiveDeleteHome:
		method = http.MethodPost
		path = "/v1/house/" + pathSegment(entityID) + "/w/delete"
		body = map[string]any{"id": requestNumberOrStringForAPI(entityID)}
	default:
		return 0, fmt.Errorf("unsupported destructive delete kind %q", kind)
	}
	response, err := callJSON(ctx, client.client, method, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, credentials)
	if err != nil {
		return 1, err
	}
	if !isBusinessOK(response) {
		return 1, fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return 1, nil
}

func (client DestructiveDeleteClient) verifyDeleted(ctx context.Context, kind DestructiveDeleteKind, houseID string, entityID string, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		var target EntitySummary
		var readCalls int
		var err error
		switch kind {
		case DestructiveDeleteDevice:
			target, readCalls, err = client.findEntity(ctx, houseID, "device", entityID, credentials)
		case DestructiveDeleteGateway:
			target, readCalls, err = client.findGatewayFromList(ctx, houseID, entityID, credentials)
		case DestructiveDeleteHome:
			target, readCalls, err = client.findHome(ctx, entityID, credentials)
		default:
			return false, calls, fmt.Errorf("unsupported destructive delete kind %q", kind)
		}
		calls += readCalls
		if err != nil {
			return false, calls, err
		}
		if target.ID == "" {
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

func (client DestructiveDeleteClient) findEntity(ctx context.Context, houseID string, entityType string, entityID string, credentials requestCredentials) (EntitySummary, int, error) {
	if strings.TrimSpace(entityID) == "" {
		return EntitySummary{}, 0, fmt.Errorf("%s id is required", entityType)
	}
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}, result.APICalls, err
	}
	for _, entity := range result.Entities {
		if entity.Type == entityType && entity.ID == entityID {
			return entity, result.APICalls, nil
		}
	}
	return EntitySummary{}, result.APICalls, nil
}

func (client DestructiveDeleteClient) findGateway(ctx context.Context, houseID string, gatewayID string, credentials requestCredentials) (EntitySummary, int, error) {
	if strings.TrimSpace(gatewayID) == "" {
		return EntitySummary{}, 0, fmt.Errorf("gateway id is required")
	}
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/"+pathSegment(gatewayID)+"/r/info", nil, credentials)
	if err != nil {
		return EntitySummary{}, 1, err
	}
	if !isBusinessOK(response) {
		return EntitySummary{}, 1, fmt.Errorf("gateway.detail.get returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	row, _ := response["data"].(map[string]any)
	return EntitySummary{
		Type:    "gateway",
		ID:      firstAnyString(row, "id", "gatewayId", "deviceId"),
		Name:    firstAnyString(row, "name", "gatewayName", "deviceName"),
		HouseID: firstNonEmpty(firstAnyString(row, "houseId"), houseID),
		RoomID:  firstAnyString(row, "roomId"),
		Status:  firstAnyString(row, "status"),
	}, 1, nil
}

func (client DestructiveDeleteClient) findGatewayFromList(ctx context.Context, houseID string, gatewayID string, credentials requestCredentials) (EntitySummary, int, error) {
	if strings.TrimSpace(gatewayID) == "" {
		return EntitySummary{}, 0, fmt.Errorf("gateway id is required")
	}
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/gateway/r/info/1/100", nil, credentials)
	if err != nil {
		return EntitySummary{}, 1, err
	}
	if !isBusinessOK(response) {
		return EntitySummary{}, 1, fmt.Errorf("gateway.list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	for _, entity := range projectEntities("gateway", houseID, response) {
		if entity.ID == gatewayID {
			return entity, 1, nil
		}
	}
	return EntitySummary{}, 1, nil
}

func (client DestructiveDeleteClient) findHome(ctx context.Context, houseID string, credentials requestCredentials) (EntitySummary, int, error) {
	if strings.TrimSpace(houseID) == "" {
		return EntitySummary{}, 0, fmt.Errorf("house id is required")
	}
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}, result.APICalls, err
	}
	for _, entity := range result.Entities {
		if entity.Type == "home" && entity.ID == houseID {
			return entity, result.APICalls, nil
		}
	}
	return EntitySummary{}, result.APICalls, nil
}

func destructiveEntityType(kind DestructiveDeleteKind) string {
	switch kind {
	case DestructiveDeleteDevice:
		return "device"
	case DestructiveDeleteGateway:
		return "gateway"
	case DestructiveDeleteHome:
		return "home"
	default:
		return "unknown"
	}
}

func destructiveVerifyWith(kind DestructiveDeleteKind) string {
	switch kind {
	case DestructiveDeleteGateway:
		return "gateway.list"
	case DestructiveDeleteHome:
		return "home.summary"
	default:
		return "entity.list"
	}
}
