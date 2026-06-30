package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func stateQueryResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, state api.StateQueryResult) contract.Response {
	result := map[string]any{
		"region":     entities.Region,
		"houseId":    entities.HouseID,
		"entity":     entitySummaryMap(entity),
		"source":     state.Source,
		"queryScope": state.QueryScope,
		"rawShape":   state.RawShape,
	}
	if state.PropertyName != "" {
		result["propertyName"] = state.PropertyName
		result["value"] = state.Value
	} else {
		result["properties"] = state.Properties
	}
	if len(state.Skipped) > 0 {
		result["skippedProperties"] = state.Skipped
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
			"apiCalls":  entityListAPICalls(entities) + stateQueryAPICalls(state),
			"cacheHits": topologyCacheHits(entities),
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
			"reason":               reason,
			"target":               target.toMap(),
			"candidates":           preview,
			"supportedEntityTypes": []string{"device"},
		},
		Warnings: []string{},
		TraceID:  "state-query-clarification",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func stateQueryPropertyName(request contract.Request) string {
	return firstRequestString(request.Parameters, "propertyName", "property", "propName")
}

func stateQueryPropertySet(device api.DeviceCapability) []string {
	properties := make([]string, 0, len(device.Properties))
	seen := map[string]bool{}
	appendProperty := func(property api.PropertyCapability) {
		if property.ID == "" || seen[property.ID] {
			return
		}
		seen[property.ID] = true
		properties = append(properties, property.ID)
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

func stateQueryAPICalls(state api.StateQueryResult) int {
	if state.APICalls > 0 {
		return state.APICalls
	}
	return 1
}
