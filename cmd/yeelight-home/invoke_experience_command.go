package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func (app *app) invokeSceneTest(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	forwarded := request
	forwarded.Intent = "scene.execute"
	response, err := app.invokeSceneExecute(ctx, forwarded, endpoint, profile, region, houseID, authorization, clientID)
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

func (app *app) invokeSceneExecute(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return sceneExecuteClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	entities, err := app.loadEntities(ctx, endpoint, profile, region, houseID, authorization, clientID, entityLoadOptions{PreferCache: true})
	if err != nil {
		return contract.Response{}, err
	}
	match, candidates, _ := findEntity(target, entities.Entities)
	if match.ID == "" {
		return sceneExecuteClarificationResponse(request, "scene_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return sceneExecuteClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "scene" {
		return sceneExecuteClarificationResponse(request, "target_not_scene", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
	}
	execution, err := api.NewSceneExecuteClient(endpoint, nil).Run(ctx, api.SceneExecuteRequest{
		HouseID: houseID,
		SceneID: match.ID,
		Credentials: api.SceneExecuteCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return sceneExecuteResponse(request, entities, match, execution), nil
}

func (app *app) invokeLightingExperienceApply(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	action, ok := explicitExperienceAction(request)
	if !ok {
		return experienceBlockedResponse(request, "explicit_experience_action_required", "请提供明确的临时灯光动作参数，例如 brightness、colorTemperature 或 color；Runtime 不根据氛围词自动选择动作。"), nil
	}
	forwarded := request
	forwarded.Intent = action.intent
	forwarded.Parameters = copyRequestParameters(request.Parameters)
	for key, value := range action.parameters {
		forwarded.Parameters[key] = value
	}
	response, err := routeExperienceLightAction(app, ctx, forwarded, endpoint, profile, region, houseID, authorization, clientID, action.intent)
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
			"delegatedIntent":  action.intent,
			"temporaryControl": true,
			"persistentWrites": false,
		}
	}
	return response, nil
}

type experienceLightAction struct {
	intent     string
	parameters map[string]any
}

func explicitExperienceAction(request contract.Request) (experienceLightAction, bool) {
	if value, ok := lightIntegerValue(request, 1, 100, "brightness", "level", "l"); ok {
		return experienceLightAction{
			intent: "light.brightness.set",
			parameters: map[string]any{
				"brightness": value,
			},
		}, true
	}
	if value, ok := lightIntegerValue(request, 2700, 6500, "colorTemperature", "color_temperature", "ct"); ok {
		return experienceLightAction{
			intent: "light.color_temperature.set",
			parameters: map[string]any{
				"colorTemperature": value,
			},
		}, true
	}
	if value, ok := lightColorValue(request); ok {
		return experienceLightAction{
			intent: "light.color.set",
			parameters: map[string]any{
				"color": value,
			},
		}, true
	}
	return experienceLightAction{}, false
}

func routeExperienceLightAction(app *app, ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, intent string) (contract.Response, error) {
	switch intent {
	case "light.brightness.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightBrightnessSpec())
	case "light.color_temperature.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightColorTemperatureSpec())
	case "light.color.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightColorSpec())
	default:
		return experienceBlockedResponse(request, "experience_action_not_supported", "该体验动作尚未纳入当前 Runtime 语义能力。"), nil
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
			"blockReason":      code,
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
