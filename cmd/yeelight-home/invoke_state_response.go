package main

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func stateQueryResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, state api.StateQueryResult) contract.Response {
	result := map[string]any{
		semantic.FieldRegion:     entities.Region,
		semantic.FieldHouseID:    entities.HouseID,
		semantic.FieldEntity:     entitySummaryMap(entity),
		semantic.FieldSource:     state.Source,
		semantic.FieldQueryScope: state.QueryScope,
	}
	if state.PropertyName != "" {
		result[semantic.FieldProperty] = semantic.PropertyName(state.PropertyName)
		result[semantic.FieldValue] = state.Value
	} else {
		result[semantic.FieldProperties] = stateQueryPublicProperties(state.Properties)
	}
	if len(state.Skipped) > 0 {
		result[semantic.FieldSkippedProperties] = stateQueryPublicSkippedProperties(state.Skipped)
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已读取 %s 的当前状态。", entity.Name),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         "state-query-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + stateQueryAPICalls(state),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func stateQueryClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
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
		UserMessage:     "请明确要查询状态的设备。",
		Clarification: map[string]any{
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"device"},
		},
		Warnings: []string{},
		TraceID:  "state-query-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func stateQuerySensitivePropertyResponse(request contract.Request, property string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "该属性属于敏感配置，不支持通过状态查询读取。",
		Clarification: map[string]any{
			semantic.FieldReason:   "sensitive_property_not_readable",
			semantic.FieldProperty: semantic.PropertyName(property),
		},
		Warnings: []string{},
		TraceID:  "state-query-sensitive-property",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
}

func stateQueryPropertyName(request contract.Request) string {
	property := firstRequestString(request.Parameters, semantic.FieldProperty)
	if property == "" {
		return ""
	}
	if id, ok := semantic.PropertyID(property); ok {
		return id
	}
	return property
}

func stateQueryPropertySet(device api.DeviceCapability) []string {
	properties := make([]string, 0, len(device.Properties))
	seen := map[string]bool{}
	appendProperty := func(property api.PropertyCapability) {
		propertyID := property.ID
		if normalized, ok := semantic.PropertyID(propertyID); ok {
			propertyID = normalized
		}
		if propertyID == "" || seen[propertyID] || semantic.PropertySensitive(propertyID) {
			return
		}
		seen[propertyID] = true
		properties = append(properties, propertyID)
	}
	for _, property := range device.Properties {
		appendProperty(property)
	}
	for _, component := range device.Components {
		for _, property := range component.Properties {
			appendProperty(property)
		}
	}
	return properties
}

func stateQuerySupportedProperties(device api.DeviceCapability) []string {
	propertyIDs := stateQueryPropertySet(device)
	result := make([]string, 0, len(propertyIDs))
	seen := map[string]bool{}
	for _, propertyID := range propertyIDs {
		publicName := semantic.PropertyName(propertyID)
		if publicName == "" || seen[publicName] {
			continue
		}
		seen[publicName] = true
		result = append(result, publicName)
	}
	return result
}

func stateQueryAPICalls(state api.StateQueryResult) int {
	if state.APICalls > 0 {
		return state.APICalls
	}
	return 1
}

func stateQueryPublicProperties(properties map[string]any) map[string]any {
	if len(properties) == 0 {
		return properties
	}
	result := map[string]any{}
	for key, value := range properties {
		result[semantic.PropertyName(key)] = value
	}
	return result
}

func stateQueryPublicSkippedProperties(properties []string) []string {
	if len(properties) == 0 {
		return properties
	}
	result := make([]string, 0, len(properties))
	seen := map[string]bool{}
	for _, property := range properties {
		name, reason := stateQueryPublicSkippedProperty(property)
		item := name
		if reason != "" {
			item = name + ":" + reason
		}
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func stateQueryPublicSkippedProperty(value string) (string, string) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	property := ""
	if len(parts) > 0 {
		property = strings.TrimSpace(parts[0])
	}
	if property == "" {
		property = "unknownProperty"
	}
	name := ""
	if _, ok := semantic.PropertyID(property); ok {
		name = semantic.PropertyName(property)
	}
	if name == "" {
		name = "unsupportedProperty"
	}
	reason := "unreadable"
	if len(parts) > 1 {
		switch strings.TrimSpace(parts[1]) {
		case "sensitive_property_not_readable":
			reason = "sensitive"
		default:
			reason = "unreadable"
		}
	}
	return name, reason
}
