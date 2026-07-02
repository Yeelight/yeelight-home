package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareDeviceUnbind(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", deviceUnbindAcceptedFields()), nil
	}
	deviceID := firstValueIDString(request.Parameters, semantic.FieldDeviceID, semantic.FieldID, semantic.FieldEntityID)
	deviceName := firstRequestString(request.Parameters, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName)
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
	var target api.EntitySummary
	if deviceID != "" {
		if found, ok := findEntitySummary(entities, "device", deviceID); ok {
			target = found
		}
	} else if deviceName != "" {
		match, candidates, _ := findEntity(entityGetTarget{name: deviceName, entityType: "device"}, entities.Entities)
		if len(candidates) > 1 {
			response := entityGetClarificationResponse(request, "ambiguous_target", entityGetTarget{name: deviceName, entityType: "device"}, candidates, entityListAPICalls(entities))
			return response, nil
		}
		target = match
	}
	if target.ID == "" {
		return configureClarificationResponse(request, "invalid_device_reference", deviceUnbindAcceptedFields()), nil
	}
	clearMac := requestBoolDefault(request.Parameters[semantic.FieldClearMAC], false)
	unbindRelDevices := requestBoolDefault(request.Parameters[semantic.FieldUnbindRelatedDevices], false)
	payload := map[string]any{
		semantic.FieldHouseID:              requestNumberOrString(houseID),
		semantic.FieldDeviceID:             target.ID,
		semantic.FieldEntityID:             target.ID,
		semantic.FieldName:                 target.Name,
		semantic.FieldClearMAC:             clearMac,
		semantic.FieldUnbindRelatedDevices: unbindRelDevices,
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
		semantic.FieldTarget:  map[string]any{semantic.FieldType: "device", semantic.FieldID: target.ID, semantic.FieldName: target.Name},
		semantic.FieldOptions: map[string]any{semantic.FieldClearMAC: clearMac, semantic.FieldUnbindRelatedDevices: unbindRelDevices},
		semantic.FieldImpact:  map[string]any{semantic.FieldMode: "r3_device_unbind", semantic.FieldCallerShouldConfirm: true, semantic.FieldRuntimeApprovalStateStored: false},
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func deviceUnbindAcceptedFields() []string {
	return semanticParameterPaths(
		semantic.FieldHouseID,
		semantic.FieldDeviceID,
		semantic.FieldDeviceName,
		semantic.FieldEntityName,
		semantic.FieldName,
		semantic.FieldClearMAC,
		semantic.FieldUnbindRelatedDevices,
	)
}

func (app *app) executeDeviceUnbind(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewDeviceUnbindClient(endpoint, nil).Run(ctx, api.DeviceUnbindRequest{
		HouseID:          record.HouseID,
		DeviceID:         valueIDString(record.Payload[semantic.FieldDeviceID]),
		ClearMac:         boolFromExecutionPayload(record.Payload[semantic.FieldClearMAC]),
		UnbindRelDevices: boolFromExecutionPayload(record.Payload[semantic.FieldUnbindRelatedDevices]),
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
