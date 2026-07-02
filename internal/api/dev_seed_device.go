package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type DevSeedDeviceRequest struct {
	HouseID             string
	RoomID              string
	Name                string
	CapabilityProductID int
	DeviceType          int
	ConnectType         int
	Bound               bool
	AllowWriteDev       bool
	VerifyAttempts      int
	VerifyInterval      time.Duration
	Credentials         DevSeedCredentials
}

type DevSeedDeviceResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	DeviceID   string `json:"deviceId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Bound      bool   `json:"bound"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
}

func (client DevSeedClient) EnsureDevice(ctx context.Context, request DevSeedDeviceRequest) (DevSeedDeviceResult, error) {
	if client.endpoint.Region != "dev" {
		return DevSeedDeviceResult{}, fmt.Errorf("dev seed is only allowed for dev region")
	}
	if !request.AllowWriteDev {
		return DevSeedDeviceResult{}, fmt.Errorf("dev seed requires --allow-write-dev")
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return DevSeedDeviceResult{}, fmt.Errorf("house id is required")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return DevSeedDeviceResult{}, fmt.Errorf("device name is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DevSeedDeviceResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	if err := client.verifyHouseScopedEntityList(ctx, houseID, credentials); err != nil {
		return DevSeedDeviceResult{}, fmt.Errorf("verify house before device seed: %w", err)
	}
	if existing := client.findDeviceByName(ctx, houseID, name, credentials); existing.ID != "" {
		return DevSeedDeviceResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			DeviceID:   existing.ID,
			Name:       existing.Name,
			Bound:      request.Bound,
			Verified:   true,
			VerifiedBy: "entity_list",
		}, nil
	}
	created, err := client.createDevice(ctx, request, credentials)
	if err != nil {
		return DevSeedDeviceResult{}, err
	}
	verified := client.verifyDeviceByName(ctx, houseID, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	if verified.ID == "" {
		return DevSeedDeviceResult{}, fmt.Errorf("unknown write result: created device was not found during verification; createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
	}
	return DevSeedDeviceResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		DeviceID:   verified.ID,
		Name:       verified.Name,
		Created:    true,
		Bound:      request.Bound,
		Verified:   true,
		VerifiedBy: "entity_list",
	}, nil
}

func (client DevSeedClient) createDevice(ctx context.Context, request DevSeedDeviceRequest, credentials requestCredentials) (writeProbeResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	productID := request.CapabilityProductID
	if productID <= 0 {
		productID = 1
	}
	deviceType := request.DeviceType
	if deviceType <= 0 {
		deviceType = 1
	}
	connectType := request.ConnectType
	if connectType < 0 {
		connectType = 0
	}
	body := map[string]any{
		semantic.FieldHouseID: requestNumberOrStringForAPI(houseID),
		semantic.FieldName:    strings.TrimSpace(request.Name),
		semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID): productID,
		semantic.FieldType:                        deviceType,
		semantic.FieldConnectType:                 connectType,
		semantic.InternalDeviceBindFlagField():    byteFlag(request.Bound),
		semantic.InternalDeviceVirtualFlagField(): 1,
		semantic.InternalDeviceIdentifierField():  time.Now().UnixNano() % 900000000,
		semantic.FieldMAC:                         devSeedMAC(),
	}
	if roomID := strings.TrimSpace(request.RoomID); roomID != "" {
		body[semantic.FieldRoomID] = requestNumberOrStringForAPI(roomID)
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/device/w/insert", body, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("device create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}

func (client DevSeedClient) verifyDeviceByName(ctx context.Context, houseID string, name string, credentials requestCredentials, attempts int, interval time.Duration) EntitySummary {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	for attempt := 0; attempt < attempts; attempt++ {
		device := client.findDeviceByName(ctx, houseID, name, credentials)
		if device.ID != "" || attempt == attempts-1 {
			return device
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return EntitySummary{}
		case <-timer.C:
		}
	}
	return EntitySummary{}
}

func (client DevSeedClient) findDeviceByName(ctx context.Context, houseID string, name string, credentials requestCredentials) EntitySummary {
	result, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return EntitySummary{}
	}
	for _, entity := range result.Entities {
		if entity.Type == "device" && entity.Name == name {
			return entity
		}
	}
	return EntitySummary{}
}

func byteFlag(enabled bool) int {
	if enabled {
		return 1
	}
	return 0
}

func devSeedMAC() string {
	value := time.Now().UnixNano()
	return fmt.Sprintf("02:CD:%02X:%02X:%02X:%02X", byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}
