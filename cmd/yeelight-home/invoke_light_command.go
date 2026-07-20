package main

import (
	"context"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

const (
	lightWriteVerificationAttempts = 5
	lightWriteVerificationInterval = 300 * time.Millisecond
)

type lightPropertySpec struct {
	propertyID    string
	missingReason string
	messageKey    string
	traceID       string
	resolveValue  func(contract.Request) (any, any, bool)
}

type lightAdjustSpec struct {
	propertyID    string
	missingReason string
	messageKey    string
	traceID       string
	min           int
	max           int
	resolveDelta  func(contract.Request) (int, bool)
}

func (app *app) invokeLightPropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec lightPropertySpec) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	writeValue, expectedValue, ok := spec.resolveValue(request)
	if !ok {
		return lightControlClarificationResponse(request, spec.missingReason, target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if direct, ok := directNodePropertyTarget(request, houseID, target); ok && direct.entityType != "device" {
		execution, err := runNodePropertySet(ctx, endpoint, direct, houseID, spec.propertyID, writeValue, request, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID, Warnings: []string{}}
		return nodePropertySetResponse(request, entities, entitySummaryFromNodeTarget(direct, houseID), execution, expectedValue, 0, semantic.LightPropertyName, i18n.Template(request.Locale, spec.messageKey), spec.traceID), nil
	}
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, err
	}
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	if match.ID == "" {
		return lightControlClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return lightControlClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "device" {
		if !nodePropertySetEntityTypeSupported(match.Type) {
			return lightControlClarificationResponse(request, "target_not_supported_node", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
		}
		execution, err := runNodePropertySet(ctx, endpoint, nodePropertyTarget{
			entityType: match.Type,
			nodeID:     match.ID,
			name:       match.Name,
			roomID:     match.RoomID,
			roomName:   target.roomName,
		}, houseID, spec.propertyID, writeValue, request, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		return nodePropertySetResponse(request, entities, match, execution, expectedValue, entityListAPICalls(entities), semantic.LightPropertyName, i18n.Template(request.Locale, spec.messageKey), spec.traceID), nil
	}
	execution, err := api.NewDevicePropertySetClient(endpoint, nil).Run(ctx, api.DevicePropertySetRequest{
		HouseID:      houseID,
		DeviceID:     match.ID,
		PropertyName: spec.propertyID,
		Value:        writeValue,
		Command:      "set",
		Credentials: api.DevicePropertySetCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	verification, err := queryLightStateUntilExpected(ctx, endpoint, match.ID, spec.propertyID, authorization, clientID, expectedValue)
	if err != nil {
		return contract.Response{}, err
	}
	return lightNumericSetResponse(request, entities, match, execution, verification, expectedValue, spec.messageKey, spec.traceID), nil
}

func (app *app) invokeLightPropertyAdjust(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec lightAdjustSpec) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	delta, ok := spec.resolveDelta(request)
	if !ok {
		return lightControlClarificationResponse(request, spec.missingReason, target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if direct, ok := directNodePropertyTarget(request, houseID, target); ok && direct.entityType != "device" {
		execution, err := runNodePropertyAdjust(ctx, endpoint, direct, houseID, spec.propertyID, delta, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID, Warnings: []string{}}
		return nodePropertyAdjustResponse(request, entities, entitySummaryFromNodeTarget(direct, houseID), execution, delta, 0, semantic.LightPropertyName, i18n.Template(request.Locale, spec.messageKey), spec.traceID), nil
	}
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, err
	}
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	if match.ID == "" {
		return lightControlClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return lightControlClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "device" {
		if !nodePropertySetEntityTypeSupported(match.Type) {
			return lightControlClarificationResponse(request, "target_not_supported_node", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
		}
		execution, err := runNodePropertyAdjust(ctx, endpoint, nodePropertyTarget{
			entityType: match.Type,
			nodeID:     match.ID,
			name:       match.Name,
			roomID:     match.RoomID,
			roomName:   target.roomName,
		}, houseID, spec.propertyID, delta, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		return nodePropertyAdjustResponse(request, entities, match, execution, delta, entityListAPICalls(entities), semantic.LightPropertyName, i18n.Template(request.Locale, spec.messageKey), spec.traceID), nil
	}
	before, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:     match.ID,
		PropertyName: spec.propertyID,
		Credentials: api.StateQueryCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	current, ok := stateNumericValue(before.Value)
	if !ok {
		return lightAdjustUnsupportedStateResponse(request, entities, match, before, spec.traceID), nil
	}
	expected := clampInt(current+delta, spec.min, spec.max)
	execution, err := api.NewDevicePropertyAdjustClient(endpoint, nil).Run(ctx, api.DevicePropertyAdjustRequest{
		DeviceID:     match.ID,
		PropertyName: spec.propertyID,
		Value:        delta,
		Credentials: api.DevicePropertyAdjustCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	verification, err := queryLightStateUntilExpected(ctx, endpoint, match.ID, spec.propertyID, authorization, clientID, float64(expected))
	if err != nil {
		return contract.Response{}, err
	}
	return lightAdjustResponse(request, entities, match, before, execution, verification, delta, expected, spec.messageKey, spec.traceID), nil
}

func queryLightStateUntilExpected(ctx context.Context, endpoint api.Endpoint, deviceID string, propertyID string, authorization string, clientID string, expected any) (api.StateQueryResult, error) {
	var last api.StateQueryResult
	for attempt := 0; attempt < lightWriteVerificationAttempts; attempt++ {
		verification, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
			DeviceID:     deviceID,
			PropertyName: propertyID,
			Credentials: api.StateQueryCredentials{
				Authorization: authorization,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return api.StateQueryResult{}, err
		}
		last = verification
		if lightStateValueMatches(verification.Value, expected) || attempt == lightWriteVerificationAttempts-1 {
			return verification, nil
		}
		timer := time.NewTimer(lightWriteVerificationInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		case <-timer.C:
		}
	}
	return last, nil
}

func lightPowerSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldPower),
		missingReason: "missing_power_value",
		messageKey:    i18n.LightPowerSet,
		traceID:       "light-power-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightPowerValue(request)
			return value, value, ok
		},
	}
}

func lightBrightnessSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness),
		missingReason: "missing_brightness_value",
		messageKey:    i18n.LightBrightnessSet,
		traceID:       "light-brightness-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightIntegerValue(request, 1, 100, semantic.FieldBrightness, semantic.FieldValue)
			return value, float64(value), ok
		},
	}
}

func lightBrightnessAdjustSpec() lightAdjustSpec {
	return lightAdjustSpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness),
		missingReason: "missing_brightness_delta",
		messageKey:    i18n.LightBrightnessAdjusted,
		traceID:       "light-brightness-adjust-command",
		min:           1,
		max:           100,
		resolveDelta: func(request contract.Request) (int, bool) {
			return lightIntegerValue(request, -100, 100, semantic.FieldDelta, semantic.FieldStep, semantic.FieldValue)
		},
	}
}

func lightColorTemperatureSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature),
		missingReason: "missing_color_temperature_value",
		messageKey:    i18n.LightColorTemperatureSet,
		traceID:       "light-color-temperature-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightIntegerValue(request, 2700, 6500, semantic.FieldColorTemperature, semantic.FieldValue)
			return value, float64(value), ok
		},
	}
}

func lightColorSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldColor),
		missingReason: "missing_color_value",
		messageKey:    i18n.LightColorSet,
		traceID:       "light-color-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightColorValue(request)
			return value, float64(value), ok
		},
	}
}

func lightColorTemperatureAdjustSpec() lightAdjustSpec {
	return lightAdjustSpec{
		propertyID:    semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature),
		missingReason: "missing_color_temperature_delta",
		messageKey:    i18n.LightColorTemperatureAdjusted,
		traceID:       "light-color-temperature-adjust-command",
		min:           2700,
		max:           6500,
		resolveDelta: func(request contract.Request) (int, bool) {
			return lightIntegerValue(request, -3800, 3800, semantic.FieldDelta, semantic.FieldStep, semantic.FieldValue)
		},
	}
}
