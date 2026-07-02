package main

import (
	"context"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareLightingDesign(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
			"local profile houseId",
		}), nil
	}
	target := entityGetTargetFromRequest(request)
	if target.id != "" || target.name != "" {
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
		if err != nil {
			return contract.Response{}, err
		}
		return app.lightingDesignPlanResponse(ctx, request, endpoint, resolved.Entities, target, authorization, clientID)
	}
	entities, err := app.loadEntities(ctx, endpoint, profile, region, houseID, authorization, clientID, entityLoadOptions{PreferCache: true})
	if err != nil {
		return contract.Response{}, err
	}
	return app.lightingDesignPlanResponse(ctx, request, endpoint, entities, target, authorization, clientID)
}

func (app *app) lightingDesignPlanResponse(ctx context.Context, request contract.Request, endpoint api.Endpoint, entities api.EntityListResult, target entityGetTarget, authorization string, clientID string) (contract.Response, error) {
	scope, selectedDevices, clarification := lightingDesignScopeTarget(request, entities, target)
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
		semantic.FieldRegion:           entities.Region,
		semantic.FieldHouseID:          entities.HouseID,
		semantic.FieldPlanType:         "local_lighting_design",
		semantic.FieldPersistentWrites: false,
		semantic.FieldApplyIntent:      "lighting.design.apply",
		semantic.FieldApplyBehavior:    "caller_authored_actions_required",
		semantic.FieldScope:            scope,
		semantic.FieldDeviceEvidence:   capabilityEvidence,
		semantic.FieldUnknownEvidence:  unknowns,
		semantic.FieldSteps: []string{
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
			semantic.FieldAPICalls:  entityListAPICalls(entities) + capabilityCalls,
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}, nil
}

func lightingDesignScope(request contract.Request, entities api.EntityListResult) (map[string]any, []api.EntitySummary, *contract.Response) {
	return lightingDesignScopeTarget(request, entities, entityGetTargetFromRequest(request))
}

func lightingDesignScopeTarget(request contract.Request, entities api.EntityListResult, target entityGetTarget) (map[string]any, []api.EntitySummary, *contract.Response) {
	if target.id == "" && target.name == "" {
		devices := firstEntitiesByType(entities.Entities, "device", 0)
		return map[string]any{
			semantic.FieldType:   "home",
			semantic.FieldTarget: map[string]any{semantic.FieldHouseID: entities.HouseID},
		}, devices, nil
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID == "" {
		response := diagnosticClarificationResponse(request, "entity_not_found", target, candidates, []string{"room", "area", "group", "device"}, entityListAPICalls(entities))
		response.TraceID = "lighting-design-clarification"
		return nil, nil, &response
	}
	scope := map[string]any{
		semantic.FieldType:   match.Type,
		semantic.FieldTarget: entitySummaryMap(match),
	}
	switch match.Type {
	case "device":
		return scope, []api.EntitySummary{match}, nil
	case "room":
		return scope, devicesInRoom(entities.Entities, match.ID, 0), nil
	default:
		scope[semantic.FieldLimitations] = []string{"当前 Runtime 只能直接确认房间和设备范围；区域、设备组成员关系仍需后续只读能力支持。"}
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
			semantic.FieldEntity:              entitySummaryMap(device),
			semantic.FieldSchemaStatus:        capability.SchemaStatus,
			semantic.FieldSupportedProperties: stateQuerySupportedProperties(capability.Device),
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
