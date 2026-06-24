package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DevSeedAutomationRequest struct {
	HouseID        string
	Name           string
	DeviceID       string
	DeviceName     string
	PropertyName   string
	PropertyValue  bool
	AllowWriteDev  bool
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    DevSeedCredentials
}

type DevSeedAutomationResult struct {
	Region       string `json:"region"`
	HouseID      string `json:"houseId"`
	AutomationID string `json:"automationId"`
	Name         string `json:"name"`
	Created      bool   `json:"created"`
	Verified     bool   `json:"verified"`
	VerifiedBy   string `json:"verifiedBy,omitempty"`
	Status       int    `json:"status"`
}

func (client DevSeedClient) EnsureAutomation(ctx context.Context, request DevSeedAutomationRequest) (DevSeedAutomationResult, error) {
	if client.endpoint.Region != "dev" {
		return DevSeedAutomationResult{}, fmt.Errorf("dev seed is only allowed for dev region")
	}
	if !request.AllowWriteDev {
		return DevSeedAutomationResult{}, fmt.Errorf("dev seed requires --allow-write-dev")
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return DevSeedAutomationResult{}, fmt.Errorf("house id is required")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return DevSeedAutomationResult{}, fmt.Errorf("automation name is required")
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return DevSeedAutomationResult{}, fmt.Errorf("device id is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DevSeedAutomationResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	payload, err := client.buildAutomationSeedPayload(request, houseID, name, deviceID)
	if err != nil {
		return DevSeedAutomationResult{}, err
	}
	metadataClient := NewMetadataCreateClient(client.endpoint, client.client)
	result, err := metadataClient.Run(ctx, MetadataCreateRequest{
		Kind:           MetadataKindAutomation,
		HouseID:        houseID,
		Payload:        payload,
		VerifyAttempts: request.VerifyAttempts,
		VerifyInterval: request.VerifyInterval,
		Credentials: MetadataCreateCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return DevSeedAutomationResult{}, err
	}
	return DevSeedAutomationResult{
		Region:       result.Region,
		HouseID:      result.HouseID,
		AutomationID: result.EntityID,
		Name:         result.Name,
		Created:      result.Created,
		Verified:     result.Verified,
		VerifiedBy:   result.VerifiedBy,
		Status:       0,
	}, nil
}

func (client DevSeedClient) buildAutomationSeedPayload(request DevSeedAutomationRequest, houseID string, name string, deviceID string) (map[string]any, error) {
	propertyName := strings.TrimSpace(request.PropertyName)
	if propertyName == "" {
		propertyName = "p"
	}
	deviceName := strings.TrimSpace(request.DeviceName)
	if deviceName == "" {
		deviceName = deviceID
	}
	params := map[string]any{
		"type": "and",
		"conditions": []any{
			map[string]any{
				"type":  "alarm",
				"clock": "23:59:00",
			},
		},
	}
	actionParams, err := compactJSON(map[string]any{
		"set": map[string]any{propertyName: request.PropertyValue},
	})
	if err != nil {
		return nil, err
	}
	parsedDeviceID, err := parseID(deviceID, "device id")
	if err != nil {
		return nil, err
	}
	action := map[string]any{
		"typeId":  2,
		"resId":   parsedDeviceID,
		"resName": deviceName,
		"rank":    0,
		"params":  actionParams,
	}
	return BuildAutomationCreatePayload(
		houseID,
		name,
		"00:00:00",
		"23:59:59",
		2,
		"0x7f",
		params,
		[]map[string]any{action},
		1,
		intPtr(0),
	)
}

func intPtr(value int) *int {
	return &value
}
