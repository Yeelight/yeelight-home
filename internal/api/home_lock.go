package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HomeLockKind string

const (
	HomeLockAll   HomeLockKind = "home.lock_all"
	HomeUnlockAll HomeLockKind = "home.unlock_all"
)

type HomeLockRequest struct {
	Kind           HomeLockKind
	HouseID        string
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    SpaceOrganizationCredentials
}

type HomeLockResult struct {
	Region      string `json:"region"`
	HouseID     string `json:"houseId"`
	Capability  string `json:"capability"`
	DeviceCount int    `json:"deviceCount"`
	Verified    bool   `json:"verified"`
	VerifiedBy  string `json:"verifiedBy,omitempty"`
	APICalls    int    `json:"apiCalls"`
}

type HomeLockClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeLockClient(endpoint Endpoint, client *http.Client) HomeLockClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeLockClient{endpoint: endpoint, client: client}
}

func (client HomeLockClient) Run(ctx context.Context, request HomeLockRequest) (HomeLockResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return HomeLockResult{}, fmt.Errorf("house id is required")
	}
	path, ok := homeLockPath(request.Kind, houseID)
	if !ok {
		return HomeLockResult{}, fmt.Errorf("unsupported home lock kind %q", request.Kind)
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return HomeLockResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	before, calls, err := client.listEntities(ctx, houseID, credentials)
	apiCalls += calls
	if err != nil {
		return HomeLockResult{}, err
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, nil, credentials)
	apiCalls++
	if err != nil {
		return HomeLockResult{}, err
	}
	if !isBusinessOK(response) {
		return HomeLockResult{}, fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", request.Kind, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	verified, calls, err := client.verifyHouseAccessible(ctx, houseID, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return HomeLockResult{}, err
	}
	if !verified {
		return HomeLockResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return HomeLockResult{
		Region:      client.endpoint.Region,
		HouseID:     houseID,
		Capability:  string(request.Kind),
		DeviceCount: before.Counts["device"],
		Verified:    true,
		VerifiedBy:  "entity.list:house_accessible_after_write_ack",
		APICalls:    apiCalls,
	}, nil
}

func homeLockPath(kind HomeLockKind, houseID string) (string, bool) {
	switch kind {
	case HomeLockAll:
		return "/v1/house/" + pathSegment(houseID) + "/lockall", true
	case HomeUnlockAll:
		return "/v1/house/" + pathSegment(houseID) + "/unlockall", true
	default:
		return "", false
	}
}

func (client HomeLockClient) listEntities(ctx context.Context, houseID string, credentials requestCredentials) (EntityListResult, int, error) {
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

func (client HomeLockClient) verifyHouseAccessible(ctx context.Context, houseID string, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		result, readCalls, err := client.listEntities(ctx, houseID, credentials)
		calls += readCalls
		if err == nil && result.HouseID == houseID {
			return true, calls, nil
		}
		if err != nil && attempt == attempts-1 {
			return false, calls, err
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
