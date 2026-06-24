package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func (app *app) invokeBehaviorExecute(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	intent, forwarded, ok := behaviorLightIntent(request)
	if !ok {
		return behaviorExecuteUnsupportedResponse(request), nil
	}
	switch intent {
	case "light.power.set":
		return app.invokeLightPropertySet(ctx, forwarded, endpoint, houseID, authorization, clientID, lightPowerSpec())
	case "light.brightness.set":
		return app.invokeLightPropertySet(ctx, forwarded, endpoint, houseID, authorization, clientID, lightBrightnessSpec())
	case "light.brightness.adjust":
		return app.invokeLightPropertyAdjust(ctx, forwarded, endpoint, houseID, authorization, clientID, lightBrightnessAdjustSpec())
	case "light.color_temperature.set":
		return app.invokeLightPropertySet(ctx, forwarded, endpoint, houseID, authorization, clientID, lightColorTemperatureSpec())
	case "light.color_temperature.adjust":
		return app.invokeLightPropertyAdjust(ctx, forwarded, endpoint, houseID, authorization, clientID, lightColorTemperatureAdjustSpec())
	case "light.color.set":
		return app.invokeLightPropertySet(ctx, forwarded, endpoint, houseID, authorization, clientID, lightColorSpec())
	default:
		return behaviorExecuteUnsupportedResponse(request), nil
	}
}

func behaviorLightIntent(request contract.Request) (string, contract.Request, bool) {
	forwarded := request
	forwarded.Parameters = copyRequestParameters(request.Parameters)
	if behavior := firstRequestString(forwarded.Parameters, "behavior", "action", "operation"); behavior != "" {
		if intent, ok := behaviorAliasIntent(behavior, forwarded.Parameters); ok {
			return intent, forwarded, true
		}
	}
	command, propName, value, hasValue := behaviorCommandFields(request)
	intent, ok := behaviorPropertyIntent(propName, command)
	if !ok {
		return "", request, false
	}
	if hasValue {
		applyBehaviorValue(forwarded.Parameters, intent, value)
	}
	return intent, forwarded, true
}

func behaviorAliasIntent(behavior string, parameters map[string]any) (string, bool) {
	switch normalizeBehaviorToken(behavior) {
	case "light.power.set", "power.set", "power":
		return "light.power.set", true
	case "turn.on", "turn_on", "on", "open":
		parameters["on"] = true
		return "light.power.set", true
	case "turn.off", "turn_off", "off", "close":
		parameters["on"] = false
		return "light.power.set", true
	case "light.brightness.set", "brightness.set", "brightness":
		return "light.brightness.set", true
	case "light.brightness.adjust", "brightness.adjust":
		return "light.brightness.adjust", true
	case "light.color.temperature.set", "light.color_temperature.set", "color.temperature.set", "color_temperature.set", "ct.set":
		return "light.color_temperature.set", true
	case "light.color.temperature.adjust", "light.color_temperature.adjust", "color.temperature.adjust", "color_temperature.adjust", "ct.adjust":
		return "light.color_temperature.adjust", true
	case "light.color.set", "color.set", "rgb.set":
		return "light.color.set", true
	default:
		return "", false
	}
}

func behaviorCommandFields(request contract.Request) (string, string, any, bool) {
	command := firstRequestString(request.Parameters, "command", "operator")
	propName := firstRequestString(request.Parameters, "propName", "propertyName", "property", "propId")
	value, hasValue := request.Parameters["value"]
	if commandMap, ok := request.Parameters["command"].(map[string]any); ok {
		command = firstNonEmptyString(firstRequestString(commandMap, "command", "operator"), command)
		propName, value, hasValue = behaviorParamFields(commandMap, propName, value, hasValue)
	}
	if controlRequest, ok := request.Parameters["controlRequest"].(map[string]any); ok {
		if nestedCommand, ok := controlRequest["command"].(map[string]any); ok {
			command = firstNonEmptyString(firstRequestString(nestedCommand, "command", "operator"), command)
			propName, value, hasValue = behaviorParamFields(nestedCommand, propName, value, hasValue)
		}
	}
	return command, propName, value, hasValue
}

func behaviorParamFields(command map[string]any, currentProp string, currentValue any, currentHasValue bool) (string, any, bool) {
	propName := currentProp
	value := currentValue
	hasValue := currentHasValue
	params, ok := command["params"].([]any)
	if !ok || len(params) == 0 {
		return propName, value, hasValue
	}
	first, ok := params[0].(map[string]any)
	if !ok {
		return propName, value, hasValue
	}
	propName = firstNonEmptyString(firstRequestString(first, "propName", "propertyName", "property", "propId"), propName)
	if nestedValue, ok := first["value"]; ok {
		value = nestedValue
		hasValue = true
	}
	return propName, value, hasValue
}

func behaviorPropertyIntent(propName string, command string) (string, bool) {
	normalizedProp := normalizeBehaviorToken(propName)
	normalizedCommand := normalizeBehaviorToken(command)
	if normalizedCommand == "" {
		normalizedCommand = "set"
	}
	switch normalizedProp {
	case "p", "power", "on":
		if normalizedCommand == "set" {
			return "light.power.set", true
		}
	case "l", "brightness", "level":
		if normalizedCommand == "set" {
			return "light.brightness.set", true
		}
		if normalizedCommand == "adjust" {
			return "light.brightness.adjust", true
		}
	case "ct", "color.temperature", "color_temperature", "colortemperature":
		if normalizedCommand == "set" {
			return "light.color_temperature.set", true
		}
		if normalizedCommand == "adjust" {
			return "light.color_temperature.adjust", true
		}
	case "c", "color", "rgb":
		if normalizedCommand == "set" {
			return "light.color.set", true
		}
	}
	return "", false
}

func applyBehaviorValue(parameters map[string]any, intent string, value any) {
	switch intent {
	case "light.power.set":
		parameters["value"] = value
	case "light.brightness.adjust", "light.color_temperature.adjust":
		parameters["delta"] = value
	case "light.color.set":
		if _, ok := value.(string); ok {
			parameters["hex"] = value
			return
		}
		parameters["value"] = value
	default:
		parameters["value"] = value
	}
}

func normalizeBehaviorToken(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "-", ".")))
}

func copyRequestParameters(parameters map[string]any) map[string]any {
	copied := make(map[string]any, len(parameters))
	for key, value := range parameters {
		copied[key] = value
	}
	return copied
}

func behaviorExecuteUnsupportedResponse(request contract.Request) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "not_supported",
		UserMessage:     "当前 Runtime 仅支持通过 behavior.execute 委托安全灯光控制：开关、亮度、色温和 RGB 颜色。",
		Result: map[string]any{
			"supportedProperties": []string{"p", "l", "ct", "c"},
			"supportedCommands":   []string{"set", "adjust"},
		},
		Warnings: []string{"unsupported_behavior_execute"},
		TraceID:  "behavior-execute-unsupported",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
		},
		Error: &contract.Error{
			Code:    "unsupported_behavior_execute",
			Message: fmt.Sprintf("behavior.execute cannot safely map request %q to a reviewed Runtime light control", request.RequestID),
		},
	}
}
