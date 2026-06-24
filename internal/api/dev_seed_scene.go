package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DevSeedSceneRequest struct {
	HouseID        string
	Name           string
	Description    string
	Icon           string
	DeviceID       string
	DeviceName     string
	PropertyName   string
	PropertyValue  bool
	AllowWriteDev  bool
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    DevSeedCredentials
}

type DevSeedSceneResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	SceneID    string `json:"sceneId"`
	Name       string `json:"name"`
	Created    bool   `json:"created"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
}

type sceneSummary struct {
	ID     string
	Name   string
	Source string
}

func (client DevSeedClient) EnsureScene(ctx context.Context, request DevSeedSceneRequest) (DevSeedSceneResult, error) {
	if client.endpoint.Region != "dev" {
		return DevSeedSceneResult{}, fmt.Errorf("dev seed is only allowed for dev region")
	}
	if !request.AllowWriteDev {
		return DevSeedSceneResult{}, fmt.Errorf("dev seed requires --allow-write-dev")
	}
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return DevSeedSceneResult{}, fmt.Errorf("house id is required")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return DevSeedSceneResult{}, fmt.Errorf("scene name is required")
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return DevSeedSceneResult{}, fmt.Errorf("device id is required")
	}
	deviceName := strings.TrimSpace(request.DeviceName)
	if deviceName == "" {
		deviceName = deviceID
	}
	propertyName := strings.TrimSpace(request.PropertyName)
	if propertyName == "" {
		propertyName = "p"
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return DevSeedSceneResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	if err := client.verifyHouseScopedEntityList(ctx, houseID, credentials); err != nil {
		return DevSeedSceneResult{}, fmt.Errorf("verify house before scene seed: %w", err)
	}
	existing, err := client.findSceneByName(ctx, houseID, name, credentials)
	if err != nil {
		return DevSeedSceneResult{}, err
	}
	if existing.ID != "" {
		return DevSeedSceneResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			SceneID:    existing.ID,
			Name:       existing.Name,
			Verified:   true,
			VerifiedBy: existing.Source,
		}, nil
	}
	created, err := client.createScene(ctx, request, credentials, propertyName, deviceName)
	if err != nil {
		return DevSeedSceneResult{}, err
	}
	verified, err := client.verifySceneByName(ctx, houseID, name, credentials, request.VerifyAttempts, request.VerifyInterval)
	if err != nil {
		return DevSeedSceneResult{}, err
	}
	if verified.ID == "" {
		return DevSeedSceneResult{}, fmt.Errorf("unknown write result: created scene was not found during verification; createId=%q code=%q message=%q dataType=%s", created.id, created.code, created.message, created.dataType)
	}
	return DevSeedSceneResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		SceneID:    verified.ID,
		Name:       verified.Name,
		Created:    true,
		Verified:   true,
		VerifiedBy: verified.Source,
	}, nil
}

func (client DevSeedClient) verifySceneByName(ctx context.Context, houseID string, name string, credentials requestCredentials, attempts int, interval time.Duration) (sceneSummary, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	for attempt := 0; attempt < attempts; attempt++ {
		scene, err := client.findSceneByName(ctx, houseID, name, credentials)
		if err != nil || scene.ID != "" || attempt == attempts-1 {
			return scene, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return sceneSummary{}, ctx.Err()
		case <-timer.C:
		}
	}
	return sceneSummary{}, nil
}

func (client DevSeedClient) findSceneByName(ctx context.Context, houseID string, name string, credentials requestCredentials) (sceneSummary, error) {
	summary, _, err := MetadataCreateClient{endpoint: client.endpoint, client: client.client}.findMetadataByNameWithCallCount(ctx, metadataCreateSpecs[MetadataKindScene], houseID, name, credentials)
	if err != nil {
		return sceneSummary{}, err
	}
	return sceneSummary{
		ID:     summary.ID,
		Name:   summary.Name,
		Source: summary.Source,
	}, nil
}

func (client DevSeedClient) createScene(ctx context.Context, request DevSeedSceneRequest, credentials requestCredentials, propertyName string, deviceName string) (writeProbeResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	parsedHouseID, err := strconv.ParseInt(houseID, 10, 64)
	if err != nil {
		return writeProbeResult{}, fmt.Errorf("house id must be numeric for scene create")
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	parsedDeviceID, err := strconv.ParseInt(deviceID, 10, 64)
	if err != nil {
		return writeProbeResult{}, fmt.Errorf("device id must be numeric for scene create")
	}
	params := map[string]any{
		"set": map[string]any{propertyName: request.PropertyValue},
	}
	paramsJSON, err := compactJSON(params)
	if err != nil {
		return writeProbeResult{}, err
	}
	body := map[string]any{
		"houseId": parsedHouseID,
		"name":    strings.TrimSpace(request.Name),
		"details": []any{
			map[string]any{
				"typeId":  2,
				"resId":   parsedDeviceID,
				"resName": deviceName,
				"action":  0,
				"rank":    0,
				"params":  paramsJSON,
			},
		},
	}
	if value := strings.TrimSpace(request.Description); value != "" {
		body["desc"] = value
	}
	if value := strings.TrimSpace(request.Icon); value != "" {
		body["icon"] = value
	}
	response, err := callJSON(ctx, client.client, http.MethodPut, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+houseID+"/scene/w/create", body, credentials)
	if err != nil {
		return writeProbeResult{}, err
	}
	if !isBusinessOK(response) {
		return writeProbeResult{}, fmt.Errorf("scene create returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return writeProbeResult{
		id:       responseID(response),
		dataType: responseDataType(response),
		code:     responseScalar(response, "code"),
		message:  responseScalar(response, "message", "msg"),
	}, nil
}

func compactJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode params: %w", err)
	}
	return string(data), nil
}
