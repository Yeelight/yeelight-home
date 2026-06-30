package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) prepareDeviceUnbind(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "parameters.deviceId"}), nil
	}
	deviceID := firstValueIDString(request.Parameters, "deviceId", "id", "entityId")
	if deviceID == "" {
		return configureClarificationResponse(request, "invalid_device_unbind_payload", []string{"parameters.houseId", "parameters.deviceId"}), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	target, ok := findEntitySummary(entities, "device", deviceID)
	if !ok {
		return configureClarificationResponse(request, "invalid_device_reference", []string{"parameters.houseId", "parameters.deviceId"}), nil
	}
	clearMac := requestBoolDefault(request.Parameters["clearMac"], false)
	unbindRelDevices := requestBoolDefault(request.Parameters["unbindRelDevices"], false)
	payload := map[string]any{
		"houseId":          requestNumberOrString(houseID),
		"deviceId":         target.ID,
		"entityId":         target.ID,
		"name":             target.Name,
		"clearMac":         clearMac,
		"unbindRelDevices": unbindRelDevices,
	}
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("解绑设备 %s", firstNonEmptyString(target.Name, target.ID)), operation.RiskR3, payload, []string{
		"这是 R3 高影响设备解绑操作；调用方应在调用 Runtime 前完成自己的用户确认",
		"执行前 Runtime 会重新读取设备并验证仍属于当前家庭",
		"执行后 Runtime 会通过 entity.list 验证设备已消失或已变为未绑定状态",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		"unbindTarget": map[string]any{"type": "device", "id": target.ID, "name": target.Name},
		"options":      map[string]any{"clearMac": clearMac, "unbindRelDevices": unbindRelDevices},
		"impact":       map[string]any{"mode": "r3_device_unbind", "callerShouldConfirm": true, "runtimeApprovalStateStored": false},
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func (app *app) executeDeviceUnbind(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewDeviceUnbindClient(endpoint, nil).Run(ctx, api.DeviceUnbindRequest{
		HouseID:          record.HouseID,
		DeviceID:         valueIDString(record.Payload["deviceId"]),
		ClearMac:         boolFromExecutionPayload(record.Payload["clearMac"]),
		UnbindRelDevices: boolFromExecutionPayload(record.Payload["unbindRelDevices"]),
		VerifyAttempts:   5,
		VerifyInterval:   time.Second,
		Credentials: api.DeviceUnbindCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return deviceUnbindExecuteResponse(request, record, result), nil
}

func boolFromExecutionPayload(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}
