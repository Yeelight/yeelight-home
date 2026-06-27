package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func (app *app) invokeDeviceUnbindPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
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
	challenge := "UNBIND device " + target.ID
	if target.Name != "" {
		challenge += " " + target.Name
	}
	record, err := plan.NewRecordWithRisk(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("解绑设备 %s", firstNonEmptyString(target.Name, target.ID)), plan.RiskR3, challenge, payload, []string{
		"这是 R3 高影响设备解绑计划，普通 plan.commit 会被阻断",
		"必须先在本机终端运行 approveCommand 完成一次性审批",
		"plan.commit 只接受 planId，忽略提交时附带的解绑字段",
		"提交前 Runtime 会重新读取设备并验证仍属于当前家庭",
		"提交后 Runtime 会通过 entity.list 验证设备已消失或已变为未绑定状态",
	}, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	preview := map[string]any{
		"unbindTarget": map[string]any{"type": "device", "id": target.ID, "name": target.Name},
		"options":      map[string]any{"clearMac": clearMac, "unbindRelDevices": unbindRelDevices},
		"impact":       map[string]any{"mode": "r3_device_unbind", "requiresLocalApprove": true},
	}
	return pendingPlanResponseWithPreview(request, record, entities, preview, 0), nil
}

func (app *app) commitDeviceUnbindPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewDeviceUnbindClient(endpoint, nil).Run(ctx, api.DeviceUnbindRequest{
		HouseID:          record.HouseID,
		DeviceID:         valueIDString(record.Payload["deviceId"]),
		ClearMac:         boolFromPlanPayload(record.Payload["clearMac"]),
		UnbindRelDevices: boolFromPlanPayload(record.Payload["unbindRelDevices"]),
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
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return deviceUnbindCommitResponse(request, record, result), nil
}

func boolFromPlanPayload(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}
