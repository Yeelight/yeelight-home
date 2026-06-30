package main

import (
	"context"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) prepareLightingDesignImport(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload := request.Parameters
	if payload == nil {
		payload = map[string]any{}
	}
	normalized, err := api.NormalizeLightingDesignImportPayload(houseID, payload)
	if err != nil {
		return configureClarificationResponseWithGuide(request, "invalid_lighting_design_import_payload", lightingDesignImportAcceptedFields(), lightingDesignImportPayloadGuide()), nil
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
	intent := request.Intent
	summary := "导入照明设计并预建设备槽位"
	if intent == "device.slot.create" {
		summary = "创建设备预留槽位"
	}
	preconditions := []string{
		"执行前重新读取家庭实体列表",
		"只使用 Runtime 归一化后的 HouseMeta: gateway.roomList、deviceList、groupList、sceneList、automationList",
		"设备槽位代表照明设计占位，不代表设备已配网或可被真实控制",
		"执行后通过家庭实体列表验证房间和设备槽位可见",
	}
	risk := operation.RiskR2
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, intent, request.RequestID, summary, risk, normalized, preconditions, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		"mode":                           "house_meta_import",
		"counts":                         lightingDesignImportPayloadCounts(normalized),
		"productResolution":              lightingDesignProductResolutionPreview(normalized),
		"persistentWrites":               true,
		"createsDeviceSlots":             true,
		"deviceSlotsArePhysicalBindings": false,
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func lightingDesignImportAcceptedFields() []string {
	return []string{
		"parameters.houseId",
		"parameters.tempId",
		"parameters.name",
		"parameters.gateway",
		"parameters.gateway.tempId",
		"parameters.gateway.name",
		"parameters.gateway.gatewayDeviceId",
		"parameters.gateway.roomList",
		"parameters.gateway.roomList[].tempId",
		"parameters.gateway.roomList[].name",
		"parameters.gateway.roomList[].deviceList",
		"parameters.gateway.roomList[].deviceList[].tempId",
		"parameters.gateway.roomList[].deviceList[].name",
		"parameters.gateway.roomList[].deviceList[].pid",
		"parameters.gateway.roomList[].deviceList[].roomTempId",
		"parameters.gateway.roomList[].groupList",
		"parameters.gateway.roomList[].groupList[].componentId",
		"parameters.gateway.roomList[].groupList[].deviceTempIdList",
		"parameters.areaList",
		"parameters.sceneList",
		"parameters.sceneList[].details",
		"parameters.automationList",
		"parameters.automationList[].params",
		"parameters.automationList[].actions",
	}
}

func (app *app) executeLightingDesignImport(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewLightingDesignImportClient(endpoint, nil).Run(ctx, api.LightingDesignImportRequest{
		HouseID:        record.HouseID,
		Intent:         record.Intent,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.LightingDesignImportCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return lightingDesignImportExecuteResponse(request, record, result), nil
}

func lightingDesignProductResolutionPreview(payload map[string]any) map[string]any {
	gateway, ok := payload["gateway"].(map[string]any)
	if !ok {
		return nil
	}
	matched := 0
	unresolved := 0
	samples := []any{}
	rooms, _ := gateway["roomList"].([]any)
	for _, rawRoom := range rooms {
		room, ok := rawRoom.(map[string]any)
		if !ok {
			continue
		}
		devices, _ := room["deviceList"].([]any)
		for _, rawDevice := range devices {
			device, ok := rawDevice.(map[string]any)
			if !ok {
				continue
			}
			extra, _ := device["extraMeta"].(map[string]any)
			item := map[string]any{
				"name":       device["name"],
				"pid":        device["pid"],
				"roomTempId": device["roomTempId"],
			}
			if value, ok := extra["materialCode"]; ok {
				item["materialCode"] = value
			}
			if value, ok := extra["productName"]; ok {
				item["productName"] = value
			}
			if device["pid"] != nil {
				matched++
			} else {
				unresolved++
			}
			if len(samples) < 8 {
				samples = append(samples, item)
			}
		}
	}
	return map[string]any{
		"matchedDeviceSlots":    matched,
		"unresolvedDeviceSlots": unresolved,
		"catalog":               "skill_selected_house_meta_products",
		"samples":               samples,
	}
}

func lightingDesignImportPayloadCounts(payload map[string]any) map[string]int {
	counts := map[string]int{
		"gateways":    0,
		"rooms":       0,
		"devices":     0,
		"groups":      0,
		"areas":       0,
		"scenes":      0,
		"automations": 0,
	}
	if areas, ok := payload["areaList"].([]any); ok {
		counts["areas"] = len(areas)
	}
	if scenes, ok := payload["sceneList"].([]any); ok {
		counts["scenes"] = len(scenes)
	}
	if automations, ok := payload["automationList"].([]any); ok {
		counts["automations"] = len(automations)
	}
	gateway, ok := payload["gateway"].(map[string]any)
	if !ok {
		return counts
	}
	counts["gateways"] = 1
	rooms, _ := gateway["roomList"].([]any)
	counts["rooms"] = len(rooms)
	for _, rawRoom := range rooms {
		room, ok := rawRoom.(map[string]any)
		if !ok {
			continue
		}
		devices, _ := room["deviceList"].([]any)
		groups, _ := room["groupList"].([]any)
		counts["devices"] += len(devices)
		counts["groups"] += len(groups)
	}
	return counts
}

func lightingDesignImportExecuteResponse(request contract.Request, record operation.Prepared, result api.LightingDesignImportResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已导入并验证照明设计，设备槽位已作为预建设计占位写入家庭。",
		Result: map[string]any{
			"region":                         result.Region,
			"houseId":                        result.HouseID,
			"capability":                     result.Capability,
			"mode":                           result.Mode,
			"counts":                         result.Counts,
			"mappings":                       result.Mappings,
			"requestKey":                     result.RequestKey,
			"verified":                       result.Verified,
			"verifiedBy":                     result.VerifiedBy,
			"persistentWrites":               true,
			"deviceSlotsArePhysicalBindings": false,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: result.Warnings,
		TraceID:  "lighting-design-import-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}
