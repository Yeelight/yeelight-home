package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) prepareLightingDesignApply(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	scope, selectedDevices, clarification := lightingDesignScope(request, entities)
	if clarification != nil {
		return *clarification, nil
	}
	if len(selectedDevices) == 0 {
		return configureClarificationResponseWithGuide(request, "target_device_evidence_unavailable", []string{"targets[0].id", "targets[0].name", "parameters.houseId", "parameters.actions"}, lightingDesignApplyPayloadGuide()), nil
	}
	actions, preview, calls := lightingDesignApplyActions(ctx, endpoint, houseID, selectedDevices, request, authorization, clientID)
	if len(actions) == 0 {
		return configureClarificationResponseWithGuide(request, "explicit_lighting_actions_required", []string{"parameters.actions", "parameters.design.actions", "parameters.power", "parameters.brightness", "parameters.colorTemperature", "parameters.color"}, lightingDesignApplyPayloadGuide()), nil
	}
	payload := map[string]any{
		"houseId": requestNumberOrString(houseID),
		"scope":   scope,
		"actions": actions,
	}
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, "应用照明设计到已验证设备属性", payload, []string{
		"提交前重新读取家庭实体和目标设备",
		"按设备能力只应用 Runtime 已验证支持的灯光属性",
		"本操作只提交已解析的设备属性动作；如需创建情景、自动化或分组，应使用对应独立 intent",
		"提交后逐项读取设备状态验证结果",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, calls), nil
}

func lightingDesignApplyActions(ctx context.Context, endpoint api.Endpoint, houseID string, devices []api.EntitySummary, request contract.Request, authorization string, clientID string) ([]any, map[string]any, int) {
	actions := []any{}
	skipped := []string{}
	apiCalls := 0
	if explicitActions, ok := explicitLightingDesignActions(request, devices); ok {
		preview := map[string]any{
			"actionCount":       len(explicitActions),
			"targetDeviceCount": len(devices),
			"supportedProperties": []string{
				"p",
				"l",
				"ct",
				"c",
			},
			"persistentWrites": true,
			"createdArtifacts": []string{},
			"skipped":          []string{},
			"actionSource":     "explicit_design_actions",
		}
		return explicitActions, preview, apiCalls
	}
	for _, device := range devices {
		capability, ok, warning := readDeviceCapability(ctx, endpoint, houseID, device.ID, authorization, clientID)
		if !ok {
			skipped = append(skipped, warning)
			continue
		}
		apiCalls++
		propertyIDs := stateQueryPropertySet(capability.Device)
		for _, action := range lightingDesignActionsForDevice(device, propertyIDs, request) {
			actions = append(actions, action)
		}
	}
	preview := map[string]any{
		"actionCount":       len(actions),
		"targetDeviceCount": len(devices),
		"supportedProperties": []string{
			"p",
			"l",
			"ct",
			"c",
		},
		"persistentWrites": true,
		"createdArtifacts": []string{},
		"skipped":          skipped,
	}
	return actions, preview, apiCalls
}

func explicitLightingDesignActions(request contract.Request, devices []api.EntitySummary) ([]any, bool) {
	rawActions, ok := requestMapList(request.Parameters["actions"])
	if !ok {
		design, designOK := request.Parameters["design"].(map[string]any)
		if !designOK {
			return nil, false
		}
		rawActions, ok = requestMapList(design["actions"])
		if !ok {
			return nil, false
		}
	}
	devicesByID := map[string]api.EntitySummary{}
	for _, device := range devices {
		if strings.TrimSpace(device.ID) != "" {
			devicesByID[device.ID] = device
		}
	}
	actions := make([]any, 0, len(rawActions))
	for _, raw := range rawActions {
		propertyName := firstNonEmptyString(
			firstRequestString(raw, "propertyName", "property", "propName", "propId"),
			firstRequestString(raw, "name"),
		)
		value, validValue := lightingDesignActionValue(propertyName, raw["value"])
		if !validValue {
			return nil, false
		}
		deviceID := firstRequestString(raw, "deviceId", "id", "entityId", "resId")
		var device api.EntitySummary
		if deviceID == "" && len(devices) == 1 {
			device = devices[0]
			deviceID = device.ID
		} else if matched, ok := devicesByID[deviceID]; ok {
			device = matched
		} else {
			return nil, false
		}
		action := lightingDesignAction(device, propertyName, value)
		if deviceName := firstRequestString(raw, "deviceName", "entityName", "name"); deviceName != "" {
			action["deviceName"] = deviceName
		}
		actions = append(actions, action)
	}
	return actions, len(actions) > 0
}

func lightingDesignActionsForDevice(device api.EntitySummary, propertyIDs []string, request contract.Request) []any {
	actions := []any{}
	explicitPower, hasExplicitPower := lightPowerValue(request)
	explicitBrightness, hasExplicitBrightness := lightIntegerValue(request, 1, 100, "brightness", "level", "l")
	explicitColorTemperature, hasExplicitColorTemperature := lightIntegerValue(request, 2700, 6500, "colorTemperature", "color_temperature", "ct")
	explicitColor, hasExplicitColor := lightColorValue(request)
	hasDeviceLevelDesign := false
	if hasPropertyID(propertyIDs, "l", "brightness") {
		if hasExplicitBrightness {
			actions = append(actions, lightingDesignAction(device, "l", explicitBrightness))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "ct", "colorTemperature") {
		if hasExplicitColorTemperature {
			actions = append(actions, lightingDesignAction(device, "ct", explicitColorTemperature))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "c", "color", "rgb") {
		if hasExplicitColor {
			actions = append(actions, lightingDesignAction(device, "c", explicitColor))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "p", "power", "on") {
		if hasExplicitPower {
			actions = append([]any{lightingDesignAction(device, "p", explicitPower)}, actions...)
		} else if hasDeviceLevelDesign {
			actions = append([]any{lightingDesignAction(device, "p", true)}, actions...)
		}
	}
	return actions
}

func lightingDesignAction(device api.EntitySummary, propertyName string, value any) map[string]any {
	return map[string]any{
		"deviceId":     device.ID,
		"deviceName":   device.Name,
		"propertyName": propertyName,
		"value":        value,
	}
}

func hasPropertyID(ids []string, candidates ...string) bool {
	for _, id := range ids {
		normalized := strings.ToLower(strings.TrimSpace(id))
		for _, candidate := range candidates {
			if normalized == strings.ToLower(candidate) {
				return true
			}
		}
	}
	return false
}

func (app *app) executeLightingDesignApply(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	actions, ok := requestMapList(record.Payload["actions"])
	if !ok || len(actions) == 0 {
		return executionBlockedResponse(request, "lighting_design_no_verifiable_actions", "照明设计应用操作没有可执行动作。"), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: record.HouseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	results := []any{}
	apiCalls := entityListAPICalls(entities)
	for _, action := range actions {
		result, calls, err := app.executeLightingDesignAction(ctx, endpoint, record.HouseID, entities, action, authorization, clientID)
		apiCalls += calls
		if err != nil {
			return contract.Response{}, err
		}
		results = append(results, result)
	}
	return lightingDesignApplyExecuteResponse(request, record, entities, results, apiCalls), nil
}

func (app *app) executeLightingDesignAction(ctx context.Context, endpoint api.Endpoint, houseID string, entities api.EntityListResult, action map[string]any, authorization string, clientID string) (map[string]any, int, error) {
	deviceID := strings.TrimSpace(requestString(action["deviceId"]))
	propertyName := strings.TrimSpace(requestString(action["propertyName"]))
	value, ok := lightingDesignActionValue(propertyName, action["value"])
	if deviceID == "" || !ok {
		return nil, 0, fmt.Errorf("invalid lighting design action")
	}
	match, _, _ := findEntity(entityGetTarget{id: deviceID, entityType: "device"}, entities.Entities)
	if match.ID == "" {
		return nil, 0, fmt.Errorf("lighting design device %s not found before write", deviceID)
	}
	execution, err := api.NewDevicePropertySetClient(endpoint, nil).Run(ctx, api.DevicePropertySetRequest{
		HouseID:      houseID,
		DeviceID:     match.ID,
		PropertyName: propertyName,
		Value:        value,
		Command:      "set",
		Credentials: api.DevicePropertySetCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return nil, devicePropertySetAPICalls(execution), err
	}
	verification, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
		DeviceID:     match.ID,
		PropertyName: propertyName,
		Credentials: api.StateQueryCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	calls := devicePropertySetAPICalls(execution) + stateQueryAPICalls(verification)
	if err != nil {
		return nil, calls, err
	}
	verified := lightingDesignValueVerified(verification.Value, value)
	return map[string]any{
		"entity":        entitySummaryMap(match),
		"propertyName":  propertyName,
		"expectedValue": value,
		"verifiedValue": verification.Value,
		"verified":      verified,
	}, calls, nil
}

func lightingDesignActionValue(propertyName string, value any) (any, bool) {
	switch propertyName {
	case "p":
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
	case "l":
		if parsed, ok := requestInt(value); ok && parsed >= 1 && parsed <= 100 {
			return parsed, true
		}
	case "ct":
		if parsed, ok := requestInt(value); ok && parsed >= 2700 && parsed <= 6500 {
			return parsed, true
		}
	case "c":
		if parsed, ok := requestInt(value); ok && parsed >= 0 && parsed <= 16777215 {
			return parsed, true
		}
	default:
		return nil, false
	}
	return nil, false
}

func lightingDesignValueVerified(actual any, expected any) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}
	switch expectedTyped := expected.(type) {
	case bool:
		switch actualTyped := actual.(type) {
		case bool:
			return actualTyped == expectedTyped
		case string:
			parsed, ok := lightPowerValue(contract.Request{Parameters: map[string]any{"value": actualTyped}})
			return ok && parsed == expectedTyped
		}
	case int:
		parsed, ok := requestInt(actual)
		return ok && parsed == expectedTyped
	}
	return false
}
