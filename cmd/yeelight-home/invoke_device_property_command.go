package main

import (
	"context"
	"fmt"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) invokeDevicePropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return devicePropertySetClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	propertyID := devicePropertySetPropertyName(request)
	if propertyID == "" {
		return devicePropertySetClarificationResponse(request, "missing_property", target, nil, 0), nil
	}
	if semantic.PropertySensitive(propertyID) {
		return devicePropertySetSensitivePropertyResponse(request, propertyID), nil
	}
	value, ok := request.Parameters[semantic.FieldValue]
	if !ok {
		return devicePropertySetClarificationResponse(request, "missing_value", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, err
	}
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	if match.ID == "" {
		return devicePropertySetClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return devicePropertySetClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "device" {
		return devicePropertySetClarificationResponse(request, "target_not_device", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
	}
	execution, err := api.NewDevicePropertySetClient(endpoint, nil).Run(ctx, api.DevicePropertySetRequest{
		HouseID:      houseID,
		DeviceID:     match.ID,
		PropertyName: propertyID,
		Value:        value,
		Command:      "set",
		Credentials: api.DevicePropertySetCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	verification, err := queryLightStateUntilExpected(ctx, endpoint, match.ID, propertyID, authorization, clientID, value)
	if err != nil {
		return contract.Response{}, err
	}
	return devicePropertySetResponse(request, entities, match, execution, verification, value), nil
}

func devicePropertySetPropertyName(request contract.Request) string {
	property := firstRequestString(request.Parameters, semantic.FieldProperty, semantic.FieldName)
	if property == "" {
		return ""
	}
	if id, ok := semantic.PropertyID(property); ok {
		return id
	}
	return property
}

func devicePropertySetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult, expected any) contract.Response {
	result := map[string]any{
		semantic.FieldRegion:        entities.Region,
		semantic.FieldHouseID:       entities.HouseID,
		semantic.FieldEntity:        entitySummaryMap(entity),
		semantic.FieldProperty:      semantic.PropertyName(execution.PropertyName),
		semantic.FieldCommand:       execution.Command,
		semantic.FieldSource:        execution.Source,
		semantic.FieldExpectedValue: expected,
		semantic.FieldVerified:      lightStateValueMatches(verification.Value, expected),
		semantic.FieldVerifiedValue: verification.Value,
	}
	if !lightStateValueMatches(verification.Value, expected) {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "partial",
			UserMessage:     fmt.Sprintf("%s 的控制指令已发送，但写后验证未匹配。", entity.Name),
			Result:          result,
			Warnings:        append(entities.Warnings, "write_verification_mismatch"),
			TraceID:         "device-property-set-verification-mismatch",
			Metrics: map[string]any{
				semantic.FieldAPICalls:  entityListAPICalls(entities) + devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification),
				semantic.FieldCacheHits: topologyCacheHits(entities),
			},
			Error: &contract.Error{
				Code:    "write_verification_mismatch",
				Message: "device property value did not match expected value after write",
			},
		}
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已设置 %s 的%s。", entity.Name, semantic.PropertyName(execution.PropertyName)),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         "device-property-set-command",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func devicePropertySetClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
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
		UserMessage:     "请明确要控制的设备、属性和值。",
		Clarification: map[string]any{
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"device"},
			semantic.FieldAcceptedFields: []string{
				semantic.ParameterPath(semantic.FieldDeviceID),
				semantic.ParameterPath(semantic.FieldDeviceName),
				semantic.ParameterPath(semantic.FieldRoomName),
				semantic.ParameterPath(semantic.FieldProperty),
				semantic.ParameterPath(semantic.FieldValue),
			},
			semantic.FieldPayloadGuide: payloadGuideForIntent(request.Intent),
		},
		Warnings: []string{},
		TraceID:  "device-property-set-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func devicePropertySetSensitivePropertyResponse(request contract.Request, property string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     "该属性属于敏感配置，不支持通过普通设备控制写入。",
		Result: map[string]any{
			semantic.FieldProperty:    semantic.PropertyName(property),
			semantic.FieldBlockReason: "sensitive_property_not_writable",
		},
		Warnings: []string{"sensitive_property_not_writable"},
		TraceID:  "device-property-set-sensitive-property",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
		Error: &contract.Error{
			Code:    "sensitive_property_not_writable",
			Message: "sensitive device property is not writable through device.property.set",
		},
	}
}
