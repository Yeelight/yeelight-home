package main

import (
	"context"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

type lightPropertySpec struct {
	propertyName    string
	missingReason   string
	messageTemplate string
	traceID         string
	resolveValue    func(contract.Request) (any, any, bool)
}

type lightAdjustSpec struct {
	propertyName    string
	missingReason   string
	messageTemplate string
	traceID         string
	min             int
	max             int
	resolveDelta    func(contract.Request) (int, bool)
}

func (app *app) invokeLightPropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec lightPropertySpec) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	writeValue, expectedValue, ok := spec.resolveValue(request)
	if !ok {
		return lightControlClarificationResponse(request, spec.missingReason, target, nil, 0), nil
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
		return lightControlClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return lightControlClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "device" {
		return lightControlClarificationResponse(request, "target_not_device", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
	}
	execution, err := api.NewDevicePropertySetClient(endpoint, nil).Run(ctx, api.DevicePropertySetRequest{
		HouseID:      houseID,
		DeviceID:     match.ID,
		PropertyName: spec.propertyName,
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
	verification, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:     match.ID,
		PropertyName: spec.propertyName,
		Credentials: api.StateQueryCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return lightNumericSetResponse(request, entities, match, execution, verification, expectedValue, spec.messageTemplate, spec.traceID), nil
}

func (app *app) invokeLightPropertyAdjust(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec lightAdjustSpec) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if target.id == "" && target.name == "" {
		return lightControlClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	delta, ok := spec.resolveDelta(request)
	if !ok {
		return lightControlClarificationResponse(request, spec.missingReason, target, nil, 0), nil
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
		return lightControlClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return lightControlClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if match.Type != "device" {
		return lightControlClarificationResponse(request, "target_not_device", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
	}
	before, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:     match.ID,
		PropertyName: spec.propertyName,
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
		PropertyName: spec.propertyName,
		Value:        delta,
		Credentials: api.DevicePropertyAdjustCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	verification, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:     match.ID,
		PropertyName: spec.propertyName,
		Credentials: api.StateQueryCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return lightAdjustResponse(request, entities, match, before, execution, verification, delta, expected, spec.messageTemplate, spec.traceID), nil
}

func lightPowerSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyName:    "p",
		missingReason:   "missing_power_value",
		messageTemplate: "已设置 %s 的开关状态。",
		traceID:         "light-power-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightPowerValue(request)
			return value, value, ok
		},
	}
}

func lightBrightnessSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyName:    "l",
		missingReason:   "missing_brightness_value",
		messageTemplate: "已设置 %s 的亮度。",
		traceID:         "light-brightness-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightIntegerValue(request, 1, 100, "brightness", "level", "value")
			return value, float64(value), ok
		},
	}
}

func lightBrightnessAdjustSpec() lightAdjustSpec {
	return lightAdjustSpec{
		propertyName:    "l",
		missingReason:   "missing_brightness_delta",
		messageTemplate: "已调整 %s 的亮度。",
		traceID:         "light-brightness-adjust-command",
		min:             1,
		max:             100,
		resolveDelta: func(request contract.Request) (int, bool) {
			return lightIntegerValue(request, -100, 100, "delta", "brightnessDelta", "brightness_delta", "step", "value")
		},
	}
}

func lightColorTemperatureSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyName:    "ct",
		missingReason:   "missing_color_temperature_value",
		messageTemplate: "已设置 %s 的色温。",
		traceID:         "light-color-temperature-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightIntegerValue(request, 2700, 6500, "colorTemperature", "color_temperature", "ct", "value")
			return value, float64(value), ok
		},
	}
}

func lightColorSpec() lightPropertySpec {
	return lightPropertySpec{
		propertyName:    "c",
		missingReason:   "missing_color_value",
		messageTemplate: "已设置 %s 的颜色。",
		traceID:         "light-color-set-command",
		resolveValue: func(request contract.Request) (any, any, bool) {
			value, ok := lightColorValue(request)
			return value, float64(value), ok
		},
	}
}

func lightColorTemperatureAdjustSpec() lightAdjustSpec {
	return lightAdjustSpec{
		propertyName:    "ct",
		missingReason:   "missing_color_temperature_delta",
		messageTemplate: "已调整 %s 的色温。",
		traceID:         "light-color-temperature-adjust-command",
		min:             2700,
		max:             6500,
		resolveDelta: func(request contract.Request) (int, bool) {
			return lightIntegerValue(request, -3800, 3800, "delta", "colorTemperatureDelta", "color_temperature_delta", "ctDelta", "step", "value")
		},
	}
}
