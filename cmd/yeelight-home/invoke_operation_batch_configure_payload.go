package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) buildOperationBatchStepPlanPayload(ctx context.Context, request contract.Request, endpoint api.Endpoint, _ string, _ string, houseID string, authorization string, clientID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	switch {
	case request.Intent == "room.create":
		return operationBatchRoomCreatePayload(request, houseID, entities)
	case request.Intent == "area.create":
		return operationBatchMetadataCreatePayload(request, houseID, entities, areaCreateSpec(), "区域")
	case request.Intent == "group.create":
		return operationBatchMetadataCreatePayload(request, houseID, entities, groupCreateSpec(), "设备组")
	case request.Intent == "scene.create":
		return operationBatchMetadataCreatePayload(request, houseID, entities, sceneCreateSpec(), "情景")
	case request.Intent == "automation.create":
		return operationBatchMetadataCreatePayload(request, houseID, entities, automationCreateSpec(), "自动化")
	case request.Intent == "home.sort.configure" || strings.HasPrefix(request.Intent, "favorite."):
		return operationBatchHomeOrganizationPayload(ctx, request, endpoint, houseID, authorization, clientID, entities)
	case request.Intent == "home.update" || request.Intent == "room.batch_create" || request.Intent == "room.batch_update" || request.Intent == "room.area.configure":
		return operationBatchHomeSpacePayload(request, houseID, entities)
	case request.Intent == "room.rename" || request.Intent == "room.update" || request.Intent == "area.update" || request.Intent == "device.rename" || request.Intent == "device.move" || request.Intent == "group.update":
		return operationBatchSpaceOrganizationPayload(request, houseID, entities)
	case request.Intent == "device.move_room.batch":
		return operationBatchSpaceBatchPayload(request, houseID, entities)
	case request.Intent == "scene.update":
		return operationBatchSceneUpdatePayload(request, houseID, entities)
	case request.Intent == "automation.update":
		return operationBatchAutomationUpdatePayload(request, houseID, entities)
	case request.Intent == "automation.enable" || request.Intent == "automation.disable":
		return operationBatchAutomationStatusPayload(request, houseID, entities)
	case request.Intent == "gateway.configure":
		return operationBatchGatewayConfigurePayload(request, houseID, entities)
	case request.Intent == "panel.button.configure" || request.Intent == "panel.button_event.update" || request.Intent == "panel.button_event.batch_update" || request.Intent == "knob.configure":
		return operationBatchPanelConfigurePayload(request, houseID, entities)
	case request.Intent == "lighting.design.import" || request.Intent == "device.slot.create":
		return operationBatchLightingDesignImportPayload(request, houseID)
	default:
		return nil, nil, "", nil, 0, "operation_batch_contains_unsupported_intent", nil
	}
}

func operationBatchRoomCreatePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	roomName := roomCreateName(request)
	if strings.TrimSpace(roomName) == "" {
		return nil, nil, "", nil, 0, "missing_room_name", nil
	}
	for _, entity := range entities.Entities {
		if entity.Type == "room" && entity.Name == roomName {
			return nil, nil, "", nil, 0, "room_name_already_exists", nil
		}
	}
	if reason := validateConfigureCreatePayload("room", nil, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	payload, err := api.BuildRoomCreatePayload(houseID, roomName, firstRequestString(request.Parameters, "description", "desc"), firstRequestString(request.Parameters, "icon"))
	if err != nil {
		return nil, nil, "", nil, 0, "invalid_room_create_payload", nil
	}
	return payload, nil, fmt.Sprintf("创建房间 %s", roomName), executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchMetadataCreatePayload(request contract.Request, houseID string, entities api.EntityListResult, spec configureCreateSpec, label string) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, err := spec.buildPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, spec.invalidReason, nil
	}
	if reason := validateConfigureCreatePayload(spec.entityType, payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	name := planPayloadString(payload, "name")
	for _, entity := range entities.Entities {
		if entity.Type == spec.entityType && entity.Name == name {
			return nil, nil, "", nil, 0, fmt.Sprintf("%s_name_already_exists", spec.entityType), nil
		}
	}
	return payload, spec.preconditions, fmt.Sprintf("创建%s %s", label, name), executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchHomeOrganizationPayload(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, preconditions, summary, err := buildHomeOrganizationPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateHomeOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	if request.Intent != "home.sort.configure" {
		return payload, preconditions, summary, executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
	}
	preview, calls, err := homeSortPreview(ctx, endpoint, houseID, authorization, clientID, payload)
	if err != nil {
		preview = map[string]any{"previewUnavailable": true, "warning": "home_sort_preview_unavailable", "plannedItems": len(payloadItems(payload))}
		calls = 1
	}
	return payload, preconditions, summary, preview, calls, "", nil
}

func operationBatchHomeSpacePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, preconditions, summary, err := buildHomeSpaceConfigurationPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateHomeSpaceConfigurationPayload(request.Intent, payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	return payload, preconditions, summary, homeSpaceConfigurationPreview(request.Intent, payload, entities), 0, "", nil
}

func operationBatchSpaceOrganizationPayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, preconditions, summary, err := buildSpaceOrganizationPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateSpaceOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	return payload, preconditions, summary, spaceOrganizationPreview(request.Intent, payload, entities), 0, "", nil
}

func operationBatchSpaceBatchPayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, preconditions, summary, err := buildSpaceBatchOrganizationPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateSpaceBatchOrganizationPayload(request.Intent, payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	return payload, preconditions, summary, spaceBatchOrganizationPreview(request.Intent, payload, entities), 0, "", nil
}

func operationBatchSceneUpdatePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, err := buildSceneUpdatePayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateSceneUpdatePayload(payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	name := firstNonEmptyString(planPayloadString(payload, "name"), valueIDString(payload["sceneId"]))
	return payload, nil, fmt.Sprintf("更新情景 %s", name), executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchAutomationUpdatePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	automation, reason := automationStatusTarget(request, entities)
	if reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	payload, err := buildAutomationUpdatePayload(request, houseID, automation.ID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateAutomationUpdatePayload(payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	name := firstNonEmptyString(planPayloadString(payload, "name"), automation.Name)
	return payload, nil, fmt.Sprintf("更新自动化 %s", name), executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchAutomationStatusPayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	automation, reason := automationStatusTarget(request, entities)
	if reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	payload := map[string]any{"houseId": requestNumberOrString(houseID), "automationId": automation.ID}
	return payload, nil, automationStatusSummary(request.Intent, automation), executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchGatewayConfigurePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, err := buildGatewayConfigurationPayload(request, houseID)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if reason := validateGatewayConfigurationPayload(payload, entities); reason != "" {
		return nil, nil, "", nil, 0, reason, nil
	}
	if !entityExists(entities, "device", valueIDString(payload["gatewayId"])) {
		return nil, nil, "", nil, 0, "invalid_gateway_reference", nil
	}
	return payload, nil, "更新网关配置", executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchPanelConfigurePayload(request contract.Request, houseID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string, error) {
	payload, preconditions, summary, err := buildPanelConfigurationPayload(request)
	if err != nil {
		return nil, nil, "", nil, 0, err.Error(), nil
	}
	if !entityExists(entities, "device", valueIDString(payload["deviceId"])) {
		return nil, nil, "", nil, 0, "invalid_panel_device_reference", nil
	}
	return payload, preconditions, summary, executionPayloadPreview(operation.Prepared{HouseID: houseID, Payload: payload}), 0, "", nil
}

func operationBatchLightingDesignImportPayload(request contract.Request, houseID string) (map[string]any, []string, string, map[string]any, int, string, error) {
	normalized, err := api.NormalizeLightingDesignImportPayload(houseID, request.Parameters)
	if err != nil {
		return nil, nil, "", nil, 0, "invalid_lighting_design_import_payload", nil
	}
	if lightingDesignImportWipesHome(normalized) {
		return nil, nil, "", nil, 0, "operation_batch_contains_r3_lighting_design_import", nil
	}
	summary := "导入照明设计并预建设备槽位"
	if request.Intent == "device.slot.create" {
		summary = "创建设备预留槽位"
	}
	preview := map[string]any{"counts": lightingDesignImportPlanCounts(normalized), "createsDeviceSlots": true, "deviceSlotsArePhysicalBindings": false}
	return normalized, nil, summary, preview, 0, "", nil
}
