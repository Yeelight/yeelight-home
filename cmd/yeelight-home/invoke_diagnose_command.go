package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func (app *app) invokeDiagnoseDevice(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	entities, match, clarification, err := resolveDiagnosticTarget(ctx, request, endpoint, houseID, authorization, clientID, "device", []string{"device"})
	if clarification != nil || err != nil {
		return responseOrError(clarification, err)
	}
	capability, capabilityOK, capabilityWarning := readDeviceCapability(ctx, endpoint, entities.HouseID, match.ID, authorization, clientID)
	state, stateOK, stateWarning := readDeviceState(ctx, endpoint, match.ID, capability, capabilityOK, authorization, clientID)

	unknowns := []string{}
	warnings := append([]string{}, entities.Warnings...)
	if capabilityWarning != "" {
		unknowns = append(unknowns, "device_schema_unavailable")
		warnings = append(warnings, capabilityWarning)
	}
	if stateWarning != "" {
		unknowns = append(unknowns, "device_state_unavailable")
		warnings = append(warnings, stateWarning)
	}

	evidence := map[string]any{
		"entity": entitySummaryMap(match),
	}
	if match.Online != nil {
		evidence["online"] = *match.Online
	}
	if match.Status != "" {
		evidence["status"] = match.Status
	}
	if capabilityOK {
		evidence["capabilitySource"] = capability.CapabilitySource
		evidence["schemaStatus"] = capability.SchemaStatus
		evidence["propertyIds"] = stateQueryPropertySet(capability.Device)
	}
	if stateOK {
		evidence["stateSource"] = state.Source
		evidence["stateShape"] = state.RawShape
		if len(state.Properties) > 0 {
			evidence["properties"] = state.Properties
		}
	}

	status := "success"
	message := fmt.Sprintf("已完成 %s 的设备诊断。", match.Name)
	if len(unknowns) > 0 {
		status = "partial"
		message = fmt.Sprintf("已完成 %s 的部分设备诊断，但仍缺少部分证据。", match.Name)
	}
	return diagnosticResponse(request, status, message, "diagnose-device-readonly", entities, evidence, unknowns, warnings, entityListAPICalls(entities)+capabilityAPICalls(capabilityOK)+stateQueryAPICalls(state)), nil
}

func (app *app) invokeDiagnoseGateway(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		response := diagnosticClarificationResponse(request, "missing_target", target, nil, []string{"gateway"}, 0)
		return response, nil
	}
	entities, err := readDiagnosticEntities(ctx, request, endpoint, houseID, authorization, clientID)
	if err != nil {
		return contract.Response{}, err
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	fallback := false
	if match.ID == "" && target.entityType == "gateway" {
		fallbackTarget := target
		fallbackTarget.entityType = ""
		match, candidates, _ = findEntity(fallbackTarget, entities.Entities)
		fallback = match.ID != ""
	}
	if match.ID == "" {
		return diagnosticClarificationResponse(request, "entity_not_found", target, candidates, []string{"gateway"}, entityListAPICalls(entities)), nil
	}
	if match.Type != "gateway" && match.Type != "device" {
		return diagnosticClarificationResponse(request, "target_not_gateway", target, []api.EntitySummary{match}, []string{"gateway"}, entityListAPICalls(entities)), nil
	}

	unknowns := []string{"gateway_child_device_health_unavailable", "gateway_network_quality_unavailable", "gateway_sync_log_unavailable"}
	warnings := append([]string{}, entities.Warnings...)
	if fallback {
		warnings = append(warnings, "gateway_projected_from_device_entity")
		unknowns = append(unknowns, "gateway_entity_type_projection_unavailable")
	}
	evidence := map[string]any{
		"entity": entitySummaryMap(match),
	}
	if match.Online != nil {
		evidence["online"] = *match.Online
	}
	if match.Status != "" {
		evidence["status"] = match.Status
	}
	return diagnosticResponse(request, "partial", fmt.Sprintf("已读取 %s 的网关相关基础信息，但缺少网关专项诊断证据。", match.Name), "diagnose-gateway-readonly", entities, evidence, unknowns, warnings, entityListAPICalls(entities)), nil
}

func (app *app) invokeDiagnoseScene(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	entities, match, clarification, err := resolveDiagnosticTarget(ctx, request, endpoint, houseID, authorization, clientID, "scene", []string{"scene"})
	if clarification != nil || err != nil {
		return responseOrError(clarification, err)
	}
	unknowns := []string{"scene_action_detail_unavailable", "scene_execution_history_unavailable"}
	evidence := map[string]any{
		"entity":             entitySummaryMap(match),
		"executionIntent":    "scene.execute",
		"executionReadiness": "entity_resolved",
	}
	return diagnosticResponse(request, "partial", fmt.Sprintf("已确认情景 %s 存在，但缺少动作明细和执行历史证据。", match.Name), "diagnose-scene-readonly", entities, evidence, unknowns, entities.Warnings, entityListAPICalls(entities)), nil
}

func (app *app) invokeDiagnoseAutomation(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	return app.invokeAutomationExplainWithMode(ctx, request, endpoint, houseID, authorization, clientID, "diagnose")
}

func (app *app) invokeAutomationExplain(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	return app.invokeAutomationExplainWithMode(ctx, request, endpoint, houseID, authorization, clientID, "explain")
}

func (app *app) invokeAutomationExplainWithMode(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, mode string) (contract.Response, error) {
	entities, match, clarification, err := resolveDiagnosticTarget(ctx, request, endpoint, houseID, authorization, clientID, "automation", []string{"automation"})
	if clarification != nil || err != nil {
		return responseOrError(clarification, err)
	}
	unknowns := []string{"automation_trigger_detail_unavailable", "automation_condition_detail_unavailable", "automation_action_detail_unavailable", "automation_history_unavailable"}
	evidence := map[string]any{
		"entity": entitySummaryMap(match),
	}
	if match.Status != "" {
		evidence["status"] = match.Status
	}
	traceID := "diagnose-automation-readonly"
	message := fmt.Sprintf("已读取自动化 %s 的基础状态，但缺少规则明细和历史证据。", match.Name)
	if mode == "explain" {
		traceID = "automation-explain-readonly"
		message = fmt.Sprintf("已根据当前可读信息解释自动化 %s，但规则明细仍需后续只读 adapter。", match.Name)
		evidence["explanationScope"] = "entity_projection_only"
	}
	return diagnosticResponse(request, "partial", message, traceID, entities, evidence, unknowns, entities.Warnings, entityListAPICalls(entities)), nil
}

func resolveDiagnosticTarget(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, expectedType string, supportedTypes []string) (api.EntityListResult, api.EntitySummary, *contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		response := diagnosticClarificationResponse(request, "missing_target", target, nil, supportedTypes, 0)
		return api.EntityListResult{}, api.EntitySummary{}, &response, nil
	}
	entities, err := readDiagnosticEntities(ctx, request, endpoint, houseID, authorization, clientID)
	if err != nil {
		return api.EntityListResult{}, api.EntitySummary{}, nil, err
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID == "" {
		response := diagnosticClarificationResponse(request, "entity_not_found", target, candidates, supportedTypes, entityListAPICalls(entities))
		return entities, api.EntitySummary{}, &response, nil
	}
	if match.Type != expectedType {
		response := diagnosticClarificationResponse(request, "target_not_"+expectedType, target, []api.EntitySummary{match}, supportedTypes, entityListAPICalls(entities))
		return entities, api.EntitySummary{}, &response, nil
	}
	return entities, match, nil, nil
}

func readDiagnosticEntities(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (api.EntityListResult, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	return api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
}

func readDeviceCapability(ctx context.Context, endpoint api.Endpoint, houseID string, deviceID string, authorization string, clientID string) (api.DeviceCapabilitiesResult, bool, string) {
	capability, err := api.NewDeviceCapabilitiesClient(endpoint, nil).Run(ctx, api.DeviceCapabilitiesRequest{
		HouseID:  houseID,
		DeviceID: deviceID,
		Credentials: api.DeviceCapabilitiesCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return api.DeviceCapabilitiesResult{}, false, "device_schema_unavailable"
	}
	return capability, true, ""
}

func readDeviceState(ctx context.Context, endpoint api.Endpoint, deviceID string, capability api.DeviceCapabilitiesResult, capabilityOK bool, authorization string, clientID string) (api.StateQueryResult, bool, string) {
	propertySet := []string{}
	if capabilityOK {
		propertySet = stateQueryPropertySet(capability.Device)
	}
	state, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:    deviceID,
		PropertySet: propertySet,
		Credentials: api.StateQueryCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return api.StateQueryResult{}, false, "device_state_unavailable"
	}
	return state, true, ""
}

func diagnosticResponse(request contract.Request, status string, message string, traceID string, entities api.EntityListResult, evidence map[string]any, unknowns []string, warnings []string, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     message,
		Result: map[string]any{
			"region":          entities.Region,
			"houseId":         entities.HouseID,
			"diagnosticType":  strings.TrimPrefix(traceID, "diagnose-"),
			"evidence":        evidence,
			"unknownEvidence": unknowns,
		},
		Warnings: warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func diagnosticClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, supportedTypes []string, apiCalls int) contract.Response {
	preview := make([]any, 0, len(candidates))
	for index, candidate := range candidates {
		if index >= 5 {
			break
		}
		preview = append(preview, entitySummaryMap(candidate))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请明确要诊断或解释的家庭实体。",
		Clarification: map[string]any{
			"reason":               reason,
			"target":               target.toMap(),
			"candidates":           preview,
			"supportedEntityTypes": supportedTypes,
		},
		Warnings: []string{},
		TraceID:  "diagnostic-clarification",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func responseOrError(response *contract.Response, err error) (contract.Response, error) {
	if err != nil {
		return contract.Response{}, err
	}
	return *response, nil
}

func capabilityAPICalls(ok bool) int {
	if ok {
		return 1
	}
	return 0
}
