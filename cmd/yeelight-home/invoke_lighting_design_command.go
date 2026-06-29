package main

import (
	"context"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func (app *app) prepareLightingDesign(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
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
	scope, selectedDevices, clarification := lightingDesignScope(request, entities)
	if clarification != nil {
		return *clarification, nil
	}
	capabilityEvidence, capabilityWarnings, capabilityCalls := lightingDesignCapabilities(ctx, endpoint, entities.HouseID, selectedDevices, authorization, clientID)

	unknowns := []string{}
	warnings := append([]string{}, entities.Warnings...)
	warnings = append(warnings, capabilityWarnings...)
	if len(selectedDevices) == 0 {
		unknowns = append(unknowns, "target_device_evidence_unavailable")
	}
	if len(capabilityWarnings) > 0 {
		unknowns = append(unknowns, "some_device_capabilities_unavailable")
	}

	status := "success"
	message := "已生成本地照明设计计划；不会直接修改设备、情景或自动化。"
	if len(unknowns) > 0 {
		status = "partial"
		message = "已生成本地照明设计计划，但仍缺少部分设备或规则证据。"
	}
	result := map[string]any{
		"region":           entities.Region,
		"houseId":          entities.HouseID,
		"planType":         "local_lighting_design",
		"persistentWrites": false,
		"applyIntent":      "lighting.design.apply",
		"applyBehavior":    "caller_authored_actions_required",
		"scope":            scope,
		"deviceEvidence":   capabilityEvidence,
		"unknownEvidence":  unknowns,
		"steps": []string{
			"读取当前家庭实体和设备能力证据",
			"调用方或 Skill 根据用户目标生成明确的设备级动作",
			"如需应用到真实设备，调用 lighting.design.apply 并传入 actions[] 或明确的 power/brightness/colorTemperature/color 参数",
		},
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     message,
		Result:          result,
		Warnings:        warnings,
		TraceID:         "lighting-design-plan-local",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + capabilityCalls,
			"cacheHits": 0,
		},
	}, nil
}

func lightingDesignScope(request contract.Request, entities api.EntityListResult) (map[string]any, []api.EntitySummary, *contract.Response) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		devices := firstEntitiesByType(entities.Entities, "device", 0)
		return map[string]any{"type": "home", "target": map[string]any{"houseId": entities.HouseID}}, devices, nil
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID == "" {
		response := diagnosticClarificationResponse(request, "entity_not_found", target, candidates, []string{"room", "area", "group", "device"}, entityListAPICalls(entities))
		response.TraceID = "lighting-design-clarification"
		return nil, nil, &response
	}
	scope := map[string]any{
		"type":   match.Type,
		"target": entitySummaryMap(match),
	}
	switch match.Type {
	case "device":
		return scope, []api.EntitySummary{match}, nil
	case "room":
		return scope, devicesInRoom(entities.Entities, match.ID, 0), nil
	default:
		scope["limitations"] = []string{"当前 Runtime 只能直接确认房间和设备范围；区域、设备组成员关系仍需后续只读 adapter。"}
		return scope, []api.EntitySummary{}, nil
	}
}

func lightingDesignCapabilities(ctx context.Context, endpoint api.Endpoint, houseID string, devices []api.EntitySummary, authorization string, clientID string) ([]any, []string, int) {
	evidence := []any{}
	warnings := []string{}
	apiCalls := 0
	for _, device := range devices {
		capability, ok, warning := readDeviceCapability(ctx, endpoint, houseID, device.ID, authorization, clientID)
		if !ok {
			warnings = append(warnings, warning)
			continue
		}
		apiCalls++
		evidence = append(evidence, map[string]any{
			"entity":       entitySummaryMap(device),
			"schemaStatus": capability.SchemaStatus,
			"propertyIds":  stateQueryPropertySet(capability.Device),
		})
	}
	return evidence, warnings, apiCalls
}

func firstEntitiesByType(entities []api.EntitySummary, entityType string, limit int) []api.EntitySummary {
	result := []api.EntitySummary{}
	for _, entity := range entities {
		if entity.Type != entityType {
			continue
		}
		result = append(result, entity)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func devicesInRoom(entities []api.EntitySummary, roomID string, limit int) []api.EntitySummary {
	result := []api.EntitySummary{}
	for _, entity := range entities {
		if entity.Type != "device" || entity.RoomID != roomID {
			continue
		}
		result = append(result, entity)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}
