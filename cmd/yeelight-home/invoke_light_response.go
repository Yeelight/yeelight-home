package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func lightPowerSetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult) contract.Response {
	return lightPropertySetResponse(request, entities, entity, execution, verification, executionValueFromRequest(request), "已设置 %s 的开关状态。", "light-power-set-command")
}

func lightNumericSetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult, expected any, messageTemplate string, traceID string) contract.Response {
	return lightPropertySetResponse(request, entities, entity, execution, verification, expected, messageTemplate, traceID)
}

func lightPropertySetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.DevicePropertySetResult, verification api.StateQueryResult, expected any, messageTemplate string, traceID string) contract.Response {
	result := map[string]any{
		semantic.FieldRegion:        entities.Region,
		semantic.FieldHouseID:       entities.HouseID,
		semantic.FieldEntity:        entitySummaryMap(entity),
		semantic.FieldProperty:      semantic.LightPropertyName(execution.PropertyName),
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
			TraceID:         strings.TrimSuffix(traceID, "-command") + "-verification-mismatch",
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
		UserMessage:     fmt.Sprintf(messageTemplate, entity.Name),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func lightAdjustResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, before api.StateQueryResult, execution api.DevicePropertyAdjustResult, verification api.StateQueryResult, delta int, expected int, messageTemplate string, traceID string) contract.Response {
	result := map[string]any{
		semantic.FieldRegion:        entities.Region,
		semantic.FieldHouseID:       entities.HouseID,
		semantic.FieldEntity:        entitySummaryMap(entity),
		semantic.FieldProperty:      semantic.LightPropertyName(execution.PropertyName),
		semantic.FieldCommand:       execution.Command,
		semantic.FieldSource:        execution.Source,
		semantic.FieldBeforeValue:   before.Value,
		semantic.FieldDelta:         delta,
		semantic.FieldExpectedValue: float64(expected),
		semantic.FieldVerified:      lightStateValueMatches(verification.Value, float64(expected)),
		semantic.FieldVerifiedValue: verification.Value,
	}
	if !lightStateValueMatches(verification.Value, float64(expected)) {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "partial",
			UserMessage:     fmt.Sprintf("%s 的调节指令已发送，但写后验证未匹配。", entity.Name),
			Result:          result,
			Warnings:        append(entities.Warnings, "write_verification_mismatch"),
			TraceID:         strings.TrimSuffix(traceID, "-command") + "-verification-mismatch",
			Metrics: map[string]any{
				semantic.FieldAPICalls:  entityListAPICalls(entities) + stateQueryAPICalls(before) + devicePropertyAdjustAPICalls(execution) + stateQueryAPICalls(verification),
				semantic.FieldCacheHits: topologyCacheHits(entities),
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
			semantic.FieldAPICalls:  entityListAPICalls(entities) + stateQueryAPICalls(before) + devicePropertyAdjustAPICalls(execution) + stateQueryAPICalls(verification),
			semantic.FieldCacheHits: topologyCacheHits(entities),
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
			semantic.FieldRegion:   entities.Region,
			semantic.FieldHouseID:  entities.HouseID,
			semantic.FieldEntity:   entitySummaryMap(entity),
			semantic.FieldProperty: semantic.LightPropertyName(before.PropertyName),
			semantic.FieldValue:    before.Value,
			semantic.FieldSource:   before.Source,
		},
		Warnings: append(entities.Warnings, "non_numeric_state"),
		TraceID:  strings.TrimSuffix(traceID, "-command") + "-unsupported-state",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + stateQueryAPICalls(before),
			semantic.FieldCacheHits: topologyCacheHits(entities),
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
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"device"},
			semantic.FieldAcceptedValueFields:  lightAcceptedValueFields(reason),
		},
		Warnings: []string{},
		TraceID:  "light-control-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func lightAcceptedValueFields(reason string) []string {
	switch reason {
	case "missing_brightness_value":
		return []string{semantic.ParameterPath(semantic.FieldBrightness), semantic.ParameterPath(semantic.FieldValue)}
	case "missing_brightness_delta":
		return []string{semantic.ParameterPath(semantic.FieldDelta), semantic.ParameterPath(semantic.FieldStep), semantic.ParameterPath(semantic.FieldValue)}
	case "missing_color_temperature_value":
		return []string{semantic.ParameterPath(semantic.FieldColorTemperature), semantic.ParameterPath(semantic.FieldValue)}
	case "missing_color_temperature_delta":
		return []string{semantic.ParameterPath(semantic.FieldDelta), semantic.ParameterPath(semantic.FieldStep), semantic.ParameterPath(semantic.FieldValue)}
	case "missing_color_value":
		return []string{semantic.ParameterPath(semantic.FieldColor), semantic.ParameterPath(semantic.FieldValue), semantic.ParameterPath(semantic.FieldHex)}
	default:
		return []string{semantic.ParameterPath(semantic.FieldPower), semantic.ParameterPath(semantic.FieldValue)}
	}
}

func lightPowerValue(request contract.Request) (bool, bool) {
	for _, key := range []string{semantic.FieldPower, semantic.FieldValue} {
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
	for _, key := range []string{semantic.FieldColor, semantic.FieldValue} {
		value, ok := request.Parameters[key]
		if !ok {
			continue
		}
		parsed, ok := requestRGBColorValue(value)
		if !ok {
			return 0, false
		}
		return parsed, true
	}
	for _, key := range []string{semantic.FieldHex} {
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

func requestRGBColorValue(value any) (int, bool) {
	if parsed, ok := requestStrictInteger(value); ok && parsed >= 0 && parsed <= 16777215 {
		return parsed, true
	}
	if text, ok := value.(string); ok {
		if parsed, ok := parseRGBHex(text); ok {
			return parsed, true
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && parsed >= 0 && parsed <= 16777215 {
			return parsed, true
		}
		return 0, false
	}
	components, ok := value.(map[string]any)
	if !ok {
		return 0, false
	}
	red, ok := rgbComponent(components, semantic.FieldRed)
	if !ok {
		return 0, false
	}
	green, ok := rgbComponent(components, semantic.FieldGreen)
	if !ok {
		return 0, false
	}
	blue, ok := rgbComponent(components, semantic.FieldBlue)
	if !ok {
		return 0, false
	}
	return red<<16 | green<<8 | blue, true
}

func rgbComponent(values map[string]any, key string) (int, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	parsed, ok := requestInteger(value)
	if !ok || parsed < 0 || parsed > 255 {
		return 0, false
	}
	return parsed, true
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

func lightStateValueMatches(actual any, expected any) bool {
	if actual == expected {
		return true
	}
	switch typed := expected.(type) {
	case float64:
		if current, ok := stateNumericValue(actual); ok {
			return float64(current) == typed
		}
	case int:
		if current, ok := stateNumericValue(actual); ok {
			return current == typed
		}
	case bool:
		if current, ok := actual.(bool); ok {
			return current == typed
		}
	case string:
		return strings.TrimSpace(requestString(actual)) == strings.TrimSpace(typed)
	}
	return false
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
