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
		return configureClarificationResponse(request, "invalid_lighting_design_import_payload", []string{"parameters.houseId", "parameters.rooms", "parameters.rooms[].items", "parameters.slots", "parameters.items"}), nil
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
		"只使用 Runtime 归一化后的房间、网关槽位、设备槽位和可选设备组结构",
		"设备槽位代表照明设计占位，不代表设备已配网或可被真实控制",
		"执行后通过家庭实体列表验证房间和设备槽位可见",
	}
	risk := operation.RiskR2
	if lightingDesignImportWipesHome(normalized) {
		risk = operation.RiskR3
		preconditions = append(preconditions, "该操作会覆盖/清空家庭设计元数据，调用方应在调用 Runtime 前完成用户确认")
	}
	record, err := operation.NewPreparedWithRisk(profile, region, houseID, intent, request.RequestID, summary, risk, normalized, preconditions, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		"mode":                           "design_sync_metadata",
		"counts":                         lightingDesignImportPlanCounts(normalized),
		"productResolution":              lightingDesignProductResolutionPreview(normalized),
		"persistentWrites":               true,
		"createsDeviceSlots":             true,
		"deviceSlotsArePhysicalBindings": false,
	}
	if lightingDesignImportWipesHome(normalized) {
		preview["clearAll"] = true
		preview["riskNote"] = "会覆盖家庭现有设计元数据"
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
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
	devices, ok := payload["devices"].([]any)
	if !ok {
		return nil
	}
	matched := 0
	unresolved := 0
	samples := []any{}
	for _, raw := range devices {
		device, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		attrs, _ := device["attrs"].(map[string]any)
		item := map[string]any{
			"localName": device["localName"],
			"pid":       device["pid"],
		}
		if value, ok := attrs["materialCode"]; ok {
			item["materialCode"] = value
		}
		if value, ok := attrs["productName"]; ok {
			item["productName"] = value
		}
		if value, ok := attrs["productMatchConfidence"]; ok {
			item["confidence"] = value
		}
		if _, ok := attrs["materialCode"]; ok {
			matched++
		} else {
			unresolved++
		}
		if len(samples) < 8 {
			samples = append(samples, item)
		}
	}
	return map[string]any{
		"matchedDeviceSlots":    matched,
		"unresolvedDeviceSlots": unresolved,
		"catalog":               "runtime_builtin_lighting_design_products",
		"samples":               samples,
	}
}

func lightingDesignImportPlanCounts(payload map[string]any) map[string]int {
	counts := map[string]int{}
	for output, key := range map[string]string{
		"gateways":    "gateways",
		"rooms":       "rooms",
		"devices":     "devices",
		"groups":      "deviceGroups",
		"scenes":      "scenes",
		"automations": "automations",
	} {
		if items, ok := payload[key].([]any); ok {
			counts[output] = len(items)
		}
	}
	return counts
}

func lightingDesignImportWipesHome(payload map[string]any) bool {
	value, ok := payload["clearAll"].(bool)
	return ok && value
}

func lightingDesignImportExecuteResponse(request contract.Request, record operation.Prepared, result api.LightingDesignImportResult) contract.Response {
	return contract.Response{
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
			"verified":                       result.Verified,
			"verifiedBy":                     result.VerifiedBy,
			"persistentWrites":               true,
			"deviceSlotsArePhysicalBindings": false,
			"clearAll":                       result.ClearAll,
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
	}
}
