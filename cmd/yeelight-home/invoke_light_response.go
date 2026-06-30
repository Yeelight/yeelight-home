package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func lightPowerSetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult) contract.Response {
	return lightPropertySetResponse(request, entities, entity, execution, verification, executionValueFromRequest(request), "已设置 %s 的开关状态。", "light-power-set-command")
}

func lightNumericSetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult, expected any, messageTemplate string, traceID string) contract.Response {
	return lightPropertySetResponse(request, entities, entity, execution, verification, expected, messageTemplate, traceID)
}

func lightPropertySetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult, expected any, messageTemplate string, traceID string) contract.Response {
	result := map[string]any{
		"region":        entities.Region,
		"houseId":       entities.HouseID,
		"entity":        entitySummaryMap(entity),
		"propertyName":  execution.PropertyName,
		"command":       execution.Command,
		"source":        execution.Source,
		"rawShape":      execution.RawShape,
		"verified":      verification.Value == expected,
		"verifiedValue": verification.Value,
	}
	if verification.Value != expected {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "partial",
			UserMessage:     fmt.Sprintf("%s 的控制指令已发送，但写后验证未匹配。", entity.Name),
			Result:          result,
			Warnings:        append(entities.Warnings, "write_verification_mismatch"),
			TraceID:         strings.TrimSuffix(traceID, "-command") + "-verification-mismatch",
			Metrics: map[string]any{
				"apiCalls":  entityListAPICalls(entities) + devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification),
				"cacheHits": topologyCacheHits(entities),
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
		UserMessage:     fmt.Sprintf(messageTemplate, entity.Name),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         traceID,
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification),
			"cacheHits": topologyCacheHits(entities),
		},
	}
}

func lightAdjustResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, before api.StateQueryResult, execution api.DevicePropertyAdjustResult, verification api.StateQueryResult, delta int, expected int, messageTemplate string, traceID string) contract.Response {
	result := map[string]any{
		"region":        entities.Region,
		"houseId":       entities.HouseID,
		"entity":        entitySummaryMap(entity),
		"propertyName":  execution.PropertyName,
		"command":       execution.Command,
		"source":        execution.Source,
		"beforeValue":   before.Value,
		"delta":         delta,
		"expectedValue": float64(expected),
		"verified":      verification.Value == float64(expected),
		"verifiedValue": verification.Value,
		"rawShape":      execution.RawShape,
	}
	if verification.Value != float64(expected) {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "partial",
			UserMessage:     fmt.Sprintf("%s 的调节指令已发送，但写后验证未匹配。", entity.Name),
			Result:          result,
			Warnings:        append(entities.Warnings, "write_verification_mismatch"),
			TraceID:         strings.TrimSuffix(traceID, "-command") + "-verification-mismatch",
			Metrics: map[string]any{
				"apiCalls":  entityListAPICalls(entities) + stateQueryAPICalls(before) + devicePropertyAdjustAPICalls(execution) + stateQueryAPICalls(verification),
				"cacheHits": topologyCacheHits(entities),
			},
			Error: &contract.Error{
				Code:    "write_verification_mismatch",
				Message: "device property value did not match expected value after adjustment",
			},
		}
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf(messageTemplate, entity.Name),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         traceID,
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + stateQueryAPICalls(before) + devicePropertyAdjustAPICalls(execution) + stateQueryAPICalls(verification),
			"cacheHits": topologyCacheHits(entities),
		},
	}
}

func lightAdjustUnsupportedStateResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, before api.StateQueryResult, traceID string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "not_supported",
		UserMessage:     fmt.Sprintf("%s 当前属性值不是可验证的数值，已取消调节。", entity.Name),
		Result: map[string]any{
			"region":       entities.Region,
			"houseId":      entities.HouseID,
			"entity":       entitySummaryMap(entity),
			"propertyName": before.PropertyName,
			"value":        before.Value,
			"source":       before.Source,
		},
		Warnings: append(entities.Warnings, "non_numeric_state"),
		TraceID:  strings.TrimSuffix(traceID, "-command") + "-unsupported-state",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + stateQueryAPICalls(before),
			"cacheHits": topologyCacheHits(entities),
		},
		Error: &contract.Error{
			Code:    "non_numeric_state",
			Message: "device property value is not numeric, adjustment verification cannot be planned",
		},
	}
}

func lightControlClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
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
		UserMessage:     "请明确要控制的灯光设备和目标状态。",
		Clarification: map[string]any{
			"reason":               reason,
			"target":               target.toMap(),
			"candidates":           preview,
			"supportedEntityTypes": []string{"device"},
			"acceptedValueFields":  lightAcceptedValueFields(reason),
		},
		Warnings: []string{},
		TraceID:  "light-control-clarification",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func lightAcceptedValueFields(reason string) []string {
	switch reason {
	case "missing_brightness_value":
		return []string{"parameters.brightness", "parameters.level", "parameters.value"}
	case "missing_brightness_delta":
		return []string{"parameters.delta", "parameters.brightnessDelta", "parameters.brightness_delta", "parameters.step", "parameters.value"}
	case "missing_color_temperature_value":
		return []string{"parameters.colorTemperature", "parameters.color_temperature", "parameters.ct", "parameters.value"}
	case "missing_color_temperature_delta":
		return []string{"parameters.delta", "parameters.colorTemperatureDelta", "parameters.color_temperature_delta", "parameters.ctDelta", "parameters.step", "parameters.value"}
	case "missing_color_value":
		return []string{"parameters.color", "parameters.rgb", "parameters.value", "parameters.hex", "parameters.colorHex"}
	default:
		return []string{"parameters.on", "parameters.power", "parameters.value"}
	}
}

func lightPowerValue(request contract.Request) (bool, bool) {
	for _, key := range []string{"on", "power", "value"} {
		value, ok := request.Parameters[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed, true
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "on", "open", "1", "开", "打开", "开启":
				return true, true
			case "false", "off", "close", "0", "关", "关闭":
				return false, true
			}
		}
	}
	return false, false
}

func lightIntegerValue(request contract.Request, min int, max int, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := request.Parameters[key]
		if !ok {
			continue
		}
		parsed, ok := requestInteger(value)
		if !ok || parsed < min || parsed > max {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func lightColorValue(request contract.Request) (int, bool) {
	for _, key := range []string{"color", "rgb", "value"} {
		value, ok := request.Parameters[key]
		if !ok {
			continue
		}
		parsed, ok := requestStrictInteger(value)
		if !ok || parsed < 0 || parsed > 16777215 {
			return 0, false
		}
		return parsed, true
	}
	for _, key := range []string{"hex", "colorHex"} {
		value, ok := request.Parameters[key].(string)
		if !ok {
			continue
		}
		parsed, ok := parseRGBHex(value)
		if !ok {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func requestStrictInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func parseRGBHex(value string) (int, bool) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "#")
	if len(trimmed) != 6 {
		return 0, false
	}
	for _, char := range trimmed {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return 0, false
		}
	}
	parsed, err := strconv.ParseInt(trimmed, 16, 32)
	if err != nil {
		return 0, false
	}
	return int(parsed), true
}

func requestInteger(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case string:
		var result int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &result); err != nil {
			return 0, false
		}
		return result, true
	default:
		return 0, false
	}
}

func stateNumericValue(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func executionValueFromRequest(request contract.Request) any {
	value, _ := lightPowerValue(request)
	return value
}

func devicePropertySetAPICalls(execution api.DevicePropertySetResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}

func devicePropertyAdjustAPICalls(execution api.DevicePropertyAdjustResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}
