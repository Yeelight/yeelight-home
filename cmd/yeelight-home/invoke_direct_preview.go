package main

import (
	"context"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) previewDirectWriteIntent(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, bool, error) {
	switch request.Intent {
	case "light.power.set":
		return app.previewLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightPowerSpec())
	case "light.brightness.set":
		return app.previewLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightBrightnessSpec())
	case "light.color_temperature.set":
		return app.previewLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightColorTemperatureSpec())
	case "light.color.set":
		return app.previewLightPropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID, lightColorSpec())
	case "device.property.set":
		return app.previewDevicePropertySet(ctx, request, endpoint, profile, region, houseID, authorization, clientID)
	case "lighting.experience.apply":
		action, ok := explicitExperienceAction(request)
		if !ok {
			return experienceBlockedResponse(request, "explicit_experience_action_required", "请提供明确的临时灯光动作参数，例如 brightness、colorTemperature 或 color；Runtime 不根据氛围词自动选择动作。"), true, nil
		}
		forwarded := request
		forwarded.Intent = action.intent
		forwarded.Parameters = copyRequestParameters(request.Parameters)
		for key, value := range action.parameters {
			forwarded.Parameters[key] = value
		}
		response, _, err := app.previewDirectWriteIntent(ctx, forwarded, endpoint, profile, region, houseID, authorization, clientID)
		if err != nil {
			return contract.Response{}, true, err
		}
		if response.Result == nil {
			response.Result = map[string]any{}
		}
		response.Result[semantic.FieldExperience] = map[string]any{
			semantic.FieldDelegatedIntent:  action.intent,
			semantic.FieldTemporaryControl: true,
			semantic.FieldPersistentWrites: false,
		}
		response.TraceID = "lighting-experience-apply-preview"
		return response, true, nil
	case "scene.execute", "scene.test":
		return app.previewSceneExecute(ctx, request, endpoint, profile, region, houseID, authorization, clientID)
	default:
		return contract.Response{}, false, nil
	}
}

func (app *app) previewDevicePropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, bool, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return devicePropertySetClarificationResponse(request, "missing_target", target, nil, 0), true, nil
	}
	propertyID := devicePropertySetPropertyName(request)
	if propertyID == "" {
		return devicePropertySetClarificationResponse(request, "missing_property", target, nil, 0), true, nil
	}
	if semantic.PropertySensitive(propertyID) {
		return devicePropertySetSensitivePropertyResponse(request, propertyID), true, nil
	}
	value, ok := request.Parameters[semantic.FieldValue]
	if !ok {
		return devicePropertySetClarificationResponse(request, "missing_value", target, nil, 0), true, nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, true, err
	}
	if response, handled := directPreviewEntityClarification(request, resolved, target, "device", devicePropertySetClarificationResponse); handled {
		return response, true, nil
	}
	return directWritePreviewResponse(request, resolved.Entities, resolved.Match, map[string]any{
		semantic.FieldIntent:   request.Intent,
		semantic.FieldProperty: semantic.PropertyName(propertyID),
		semantic.FieldValue:    value,
		semantic.FieldCommand:  "set",
	}), true, nil
}

func (app *app) previewLightPropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec lightPropertySpec) (contract.Response, bool, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), true, nil
	}
	writeValue, _, ok := spec.resolveValue(request)
	if !ok {
		return lightControlClarificationResponse(request, spec.missingReason, target, nil, 0), true, nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, true, err
	}
	if response, ok := directPreviewEntityClarification(request, resolved, target, "device", lightControlClarificationResponse); ok {
		return response, true, nil
	}
	return directWritePreviewResponse(request, resolved.Entities, resolved.Match, map[string]any{
		semantic.FieldIntent:   request.Intent,
		semantic.FieldProperty: semantic.LightPropertyName(spec.propertyID),
		semantic.FieldValue:    writeValue,
		semantic.FieldCommand:  "set",
	}), true, nil
}

func (app *app) previewSceneExecute(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, bool, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return sceneExecuteClarificationResponse(request, "missing_target", target, nil, 0), true, nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, true, err
	}
	if response, ok := directPreviewEntityClarification(request, resolved, target, "scene", sceneExecuteClarificationResponse); ok {
		return response, true, nil
	}
	response := directWritePreviewResponse(request, resolved.Entities, resolved.Match, map[string]any{
		semantic.FieldIntent:  request.Intent,
		semantic.FieldSceneID: resolved.Match.ID,
	})
	if request.Intent == "scene.test" {
		response.Result[semantic.FieldTestOnly] = true
	}
	return response, true, nil
}

func directPreviewEntityClarification(request contract.Request, resolved entityResolveResult, target entityGetTarget, expectedType string, build func(contract.Request, string, entityGetTarget, []api.EntitySummary, int) contract.Response) (contract.Response, bool) {
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	if match.ID == "" {
		reason := "entity_not_found"
		if expectedType == "scene" {
			reason = "scene_not_found"
		}
		return build(request, reason, target, candidates, entityListAPICalls(entities)), true
	}
	if len(candidates) > 1 && target.id == "" {
		return build(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), true
	}
	if match.Type != expectedType {
		return build(request, "target_not_"+expectedType, target, []api.EntitySummary{match}, entityListAPICalls(entities)), true
	}
	return contract.Response{}, false
}

func directWritePreviewResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, planned map[string]any) contract.Response {
	planned[semantic.FieldEntity] = entitySummaryMap(entity)
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已生成 direct intent dry-run 预览；未调用云端写接口。",
		Result: map[string]any{
			semantic.FieldDryRun:           true,
			semantic.FieldPersistentWrites: false,
			semantic.FieldPlanned:          planned,
		},
		Warnings: appendWarning(entities.Warnings, "dry_run_no_cloud_write"),
		TraceID:  "direct-write-preview",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}
