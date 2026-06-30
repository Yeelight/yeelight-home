package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DeviceUnbindCredentials struct {
	Authorization string
	ClientID      string
}

type DeviceUnbindRequest struct {
	HouseID          string
	DeviceID         string
	ClearMac         bool
	UnbindRelDevices bool
	VerifyAttempts   int
	VerifyInterval   time.Duration
	Credentials      DeviceUnbindCredentials
}

type DeviceUnbindResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	DeviceID         string           `json:"deviceId"`
	Name             string           `json:"name,omitempty"`
	ClearMac         bool             `json:"clearMac"`
	UnbindRelDevices bool             `json:"unbindRelDevices"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	VerifiedEntities EntityListResult `json:"-"`
}

type DeviceUnbindClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewDeviceUnbindClient(endpoint Endpoint, client *http.Client) DeviceUnbindClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return DeviceUnbindClient{endpoint: endpoint, client: client}
}

func (client DeviceUnbindClient) Run(ctx context.Context, request DeviceUnbindRequest) (DeviceUnbindResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	deviceID := strings.TrimSpace(request.DeviceID)
	if houseID == "" {
		return DeviceUnbindResult{}, fmt.Errorf("house id is required")
	}
	if deviceID == "" {
		return DeviceUnbindResult{}, fmt.Errorf("device id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DeviceUnbindResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	before, _, calls, err := client.findDevice(ctx, houseID, deviceID, credentials)
	apiCalls := calls
	if err != nil {
		return DeviceUnbindResult{}, err
	}
	if before.ID == "" {
		return DeviceUnbindResult{}, fmt.Errorf("device %s not found before unbind", deviceID)
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/device/"+pathSegment(deviceID)+"/w/unbind", map[string]any{
		"clearMac":         request.ClearMac,
		"unbindRelDevices": request.UnbindRelDevices,
	}, credentials)
	apiCalls++
	if err != nil {
		return DeviceUnbindResult{}, err
	}
	if !isBusinessOK(response) {
		return DeviceUnbindResult{}, fmt.Errorf("device.unbind returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	ok, verifiedBy, verifiedEntities, calls, err := client.verifyDeviceUnbound(ctx, houseID, deviceID, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return DeviceUnbindResult{}, err
	}
	if !ok {
		return DeviceUnbindResult{}, fmt.Errorf("device.unbind verification mismatch")
	}
	return DeviceUnbindResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		DeviceID:         before.ID,
		Name:             before.Name,
		ClearMac:         request.ClearMac,
		UnbindRelDevices: request.UnbindRelDevices,
		Verified:         true,
		VerifiedBy:       verifiedBy,
		APICalls:         apiCalls,
		VerifiedEntities: verifiedEntities,
	}, nil
}

func (client DeviceUnbindClient) findDevice(ctx context.Context, houseID string, deviceID string, credentials requestCredentials) (EntitySummary, EntityListResult, int, error) {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}, result, result.APICalls, err
	}
	for _, entity := range result.Entities {
		if entity.Type == "device" && entity.ID == deviceID {
			return entity, result, result.APICalls, nil
		}
	}
	return EntitySummary{}, result, result.APICalls, nil
}

func (client DeviceUnbindClient) verifyDeviceUnbound(ctx context.Context, houseID string, deviceID string, credentials requestCredentials, attempts int, interval time.Duration) (bool, string, EntityListResult, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entity, entities, readCalls, err := client.findDevice(ctx, houseID, deviceID, credentials)
		calls += readCalls
		if err != nil {
			return false, "", entities, calls, err
		}
		if entity.ID == "" {
			return true, "entity.list:missing", entities, calls, nil
		}
		if entity.Bind != nil && !*entity.Bind {
			return true, "entity.list:bind=false", entities, calls, nil
		}
		if attempt == attempts-1 {
			return false, "", entities, calls, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, "", entities, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, "", EntityListResult{}, calls, nil
}
