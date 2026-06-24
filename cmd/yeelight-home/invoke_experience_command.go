package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func (app *app) invokeSceneTest(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	forwarded := request
	forwarded.Intent = "scene.execute"
	response, err := app.invoke(ctx, forwarded)
	if err != nil {
		return contract.Response{}, err
	}
	if response.Status == "success" {
		response.UserMessage = strings.Replace(response.UserMessage, "已执行", "已测试执行", 1)
		response.TraceID = "scene-test-command"
		if response.Result == nil {
			response.Result = map[string]any{}
		}
		response.Result["testOnly"] = true
	}
	return response, nil
}

func (app *app) invokeLightingExperienceApply(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	domainCatalog := loadRuntimeLightingCatalog()
	recipe := selectLightingRecipe(request, domainCatalog)
	action, skipped, ok := experienceAction(recipe)
	if !ok {
		return experienceBlockedResponse(request, "experience_recipe_not_executable", "当前体验规则没有可安全执行的临时灯光动作。"), nil
	}
	forwarded := request
	forwarded.Intent = action.intent
	forwarded.Parameters = copyRequestParameters(request.Parameters)
	for key, value := range action.parameters {
		forwarded.Parameters[key] = value
	}
	response, err := routeExperienceLightAction(app, ctx, forwarded, endpoint, houseID, authorization, clientID, action.intent)
	if err != nil {
		return contract.Response{}, err
	}
	if response.Status == "success" || response.Status == "partial" {
		response.UserMessage = "已应用临时灯光体验；没有创建或修改情景、自动化或设备配置。"
		response.TraceID = "lighting-experience-apply-command"
		if response.Result == nil {
			response.Result = map[string]any{}
		}
		response.Result["experience"] = map[string]any{
			"recipe":           recipe,
			"delegatedIntent":  action.intent,
			"temporaryControl": true,
			"persistentWrites": false,
			"skippedActions":   skipped,
		}
	}
	return response, nil
}

type experienceLightAction struct {
	intent     string
	parameters map[string]any
}

func experienceAction(recipe map[string]any) (experienceLightAction, []string, bool) {
	skipped := []string{}
	if value, ok := recipeInt(recipe, "brightness", "brightnessMax", "backgroundBrightnessMax"); ok {
		return experienceLightAction{
			intent: "light.brightness.set",
			parameters: map[string]any{
				"brightness": value,
			},
		}, skipped, true
	}
	if mainLight, ok := recipe["mainLight"].(string); ok && mainLight == "off_or_low" {
		skipped = append(skipped, "mainLight_off_not_applied_globally")
		return experienceLightAction{
			intent: "light.brightness.set",
			parameters: map[string]any{
				"brightness": 10,
			},
		}, skipped, true
	}
	if value, ok := recipeInt(recipe, "colorTemperature"); ok {
		return experienceLightAction{
			intent: "light.color_temperature.set",
			parameters: map[string]any{
				"colorTemperature": value,
			},
		}, skipped, true
	}
	return experienceLightAction{}, skipped, false
}

func recipeInt(recipe map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		value, ok := recipe[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int(math.Round(typed)), true
		case int:
			return typed, true
		case string:
			if parsed, ok := conservativeRecipeInt(typed); ok {
				return parsed, true
			}
		}
	}
	return 0, false
}

func conservativeRecipeInt(value string) (int, bool) {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r < '0' || r > '9'
	})
	for _, part := range parts {
		if part == "" {
			continue
		}
		parsed, err := strconv.Atoi(part)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func routeExperienceLightAction(app *app, ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, intent string) (contract.Response, error) {
	switch intent {
	case "light.brightness.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, authorization, clientID, lightBrightnessSpec())
	case "light.color_temperature.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, authorization, clientID, lightColorTemperatureSpec())
	case "light.color.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, houseID, authorization, clientID, lightColorSpec())
	default:
		return experienceBlockedResponse(request, "experience_action_not_reviewed", "该体验动作尚未通过 Runtime 安全包装审核。"), nil
	}
}

func experienceBlockedResponse(request contract.Request, code string, message string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Result: map[string]any{
			"persistentWrites": false,
		},
		Warnings: []string{code},
		TraceID:  "lighting-experience-blocked",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
		},
		Error: &contract.Error{
			Code:    code,
			Message: fmt.Sprintf("lighting experience request %q was blocked", request.RequestID),
		},
	}
}
