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
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareLightingDesignApply(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
			"local profile houseId",
		}), nil
	}
	target := entityGetTargetFromRequest(request)
	var entities api.EntityListResult
	var err error
	if target.id != "" || target.name != "" {
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
		if err != nil {
			return contract.Response{}, err
		}
		entities = resolved.Entities
	} else {
		entities, err = app.loadEntities(ctx, endpoint, profile, region, houseID, authorization, clientID, entityLoadOptions{PreferCache: true})
		if err != nil {
			return contract.Response{}, err
		}
	}
	scope, selectedDevices, clarification := lightingDesignScopeTarget(request, entities, target)
	if clarification != nil {
		return *clarification, nil
	}
	if len(selectedDevices) == 0 {
		return configureClarificationResponseWithGuide(request, "target_device_evidence_unavailable", []string{
			semantic.FieldPath(semantic.ArrayField(semantic.FieldTargets), semantic.FieldID),
			semantic.FieldPath(semantic.ArrayField(semantic.FieldTargets), semantic.FieldName),
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.ParameterPath(semantic.FieldActions),
		}, lightingDesignApplyPayloadGuide()), nil
	}
	actions, preview, calls := lightingDesignApplyActions(ctx, endpoint, houseID, selectedDevices, request, authorization, clientID)
	if len(actions) == 0 {
		return configureClarificationResponseWithGuide(request, "explicit_lighting_actions_required", []string{
			semantic.ParameterPath(semantic.FieldActions),
			semantic.ParameterPath(semantic.FieldDesign, semantic.FieldActions),
			semantic.ParameterPath(semantic.FieldPower),
			semantic.ParameterPath(semantic.FieldBrightness),
			semantic.ParameterPath(semantic.FieldColorTemperature),
			semantic.ParameterPath(semantic.FieldColor),
		}, lightingDesignApplyPayloadGuide()), nil
	}
	payload := map[string]any{
		semantic.FieldHouseID: requestNumberOrString(houseID),
		semantic.FieldScope:   scope,
		semantic.FieldActions: actions,
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
			semantic.FieldActionCount:       len(explicitActions),
			semantic.FieldTargetDeviceCount: len(devices),
			semantic.FieldSupportedProperties: []string{
				semantic.FieldPower,
				semantic.FieldBrightness,
				semantic.FieldColorTemperature,
				semantic.FieldColor,
			},
			semantic.FieldPersistentWrites: true,
			semantic.FieldCreatedArtifacts: []string{},
			semantic.FieldSkipped:          []string{},
			semantic.FieldActionSource:     "explicit_design_actions",
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
		semantic.FieldActionCount:       len(actions),
		semantic.FieldTargetDeviceCount: len(devices),
		semantic.FieldSupportedProperties: []string{
			semantic.FieldPower,
			semantic.FieldBrightness,
			semantic.FieldColorTemperature,
			semantic.FieldColor,
		},
		semantic.FieldPersistentWrites: true,
		semantic.FieldCreatedArtifacts: []string{},
		semantic.FieldSkipped:          skipped,
	}
	return actions, preview, apiCalls
}

func explicitLightingDesignActions(request contract.Request, devices []api.EntitySummary) ([]any, bool) {
	rawActions, ok := requestMapList(request.Parameters[semantic.FieldActions])
	if !ok {
		design, designOK := request.Parameters[semantic.FieldDesign].(map[string]any)
		if !designOK {
			return nil, false
		}
		rawActions, ok = requestMapList(design[semantic.FieldActions])
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
		device, ok := lightingDesignActionDevice(raw, devices, devicesByID)
		if !ok {
			return nil, false
		}
		if set, ok := raw[semantic.FieldSet].(map[string]any); ok && len(set) > 0 {
			for _, property := range []string{semantic.FieldPower, semantic.FieldBrightness, semantic.FieldColorTemperature, semantic.FieldColor} {
				value, exists := set[property]
				if !exists {
					continue
				}
				parsed, validValue := lightingDesignActionValue(property, value)
				if !validValue {
					return nil, false
				}
				actions = append(actions, lightingDesignAction(device, property, parsed))
			}
			continue
		}
		property := firstRequestString(raw, semantic.FieldProperty)
		value, validValue := lightingDesignActionValue(property, raw[semantic.FieldValue])
		if !validValue {
			return nil, false
		}
		actions = append(actions, lightingDesignAction(device, property, value))
	}
	return actions, len(actions) > 0
}

func lightingDesignActionDevice(raw map[string]any, devices []api.EntitySummary, devicesByID map[string]api.EntitySummary) (api.EntitySummary, bool) {
	targetType := strings.TrimSpace(firstRequestString(raw, semantic.FieldTargetType, semantic.FieldEntityType))
	if targetType != "" && targetType != "device" {
		return api.EntitySummary{}, false
	}
	deviceID := firstRequestString(raw, semantic.FieldDeviceID, semantic.FieldTargetID, semantic.FieldEntityID)
	if matched, ok := devicesByID[deviceID]; ok {
		return matched, true
	}
	deviceName := firstRequestString(raw, semantic.FieldDeviceName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName)
	if deviceName != "" {
		match, candidates, _ := findEntity(entityGetTarget{name: deviceName, entityType: "device"}, devices)
		if match.ID != "" && len(candidates) == 1 {
			return match, true
		}
		return api.EntitySummary{}, false
	}
	if deviceID == "" && len(devices) == 1 {
		return devices[0], true
	}
	return api.EntitySummary{}, false
}

func lightingDesignActionsForDevice(device api.EntitySummary, propertyIDs []string, request contract.Request) []any {
	actions := []any{}
	explicitPower, hasExplicitPower := lightPowerValue(request)
	explicitBrightness, hasExplicitBrightness := lightIntegerValue(request, 1, 100, semantic.FieldBrightness)
	explicitColorTemperature, hasExplicitColorTemperature := lightIntegerValue(request, 2700, 6500, semantic.FieldColorTemperature)
	explicitColor, hasExplicitColor := lightColorValue(request)
	hasDeviceLevelDesign := false
	if hasPropertyID(propertyIDs, "l", "brightness") {
		if hasExplicitBrightness {
			actions = append(actions, lightingDesignAction(device, semantic.FieldBrightness, explicitBrightness))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "ct", "colorTemperature") {
		if hasExplicitColorTemperature {
			actions = append(actions, lightingDesignAction(device, semantic.FieldColorTemperature, explicitColorTemperature))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "c", "color", "rgb") {
		if hasExplicitColor {
			actions = append(actions, lightingDesignAction(device, semantic.FieldColor, explicitColor))
			hasDeviceLevelDesign = true
		}
	}
	if hasPropertyID(propertyIDs, "p", "power", "on") {
		if hasExplicitPower {
			actions = append([]any{lightingDesignAction(device, semantic.FieldPower, explicitPower)}, actions...)
		} else if hasDeviceLevelDesign {
			actions = append([]any{lightingDesignAction(device, semantic.FieldPower, true)}, actions...)
		}
	}
	return actions
}

func lightingDesignAction(device api.EntitySummary, property string, value any) map[string]any {
	return map[string]any{
		semantic.FieldDeviceID:   device.ID,
		semantic.FieldDeviceName: device.Name,
		semantic.FieldProperty:   semantic.LightPropertyName(property),
		semantic.FieldValue:      value,
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
	actions, ok := requestMapList(record.Payload[semantic.FieldActions])
	if !ok || len(actions) == 0 {
		return executionBlockedResponse(request, "lighting_design_no_verifiable_actions", "照明设计应用操作没有可执行动作。"), nil
	}
	entities, err := app.loadEntities(ctx, endpoint, record.Profile, record.Region, record.HouseID, authorization, clientID, entityLoadOptions{Refresh: true})
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
	deviceID := strings.TrimSpace(requestString(action[semantic.FieldDeviceID]))
	property := strings.TrimSpace(requestString(action[semantic.FieldProperty]))
	value, ok := lightingDesignActionValue(property, action[semantic.FieldValue])
	propertyID, propertyOK := semantic.LightPropertyID(property)
	if deviceID == "" || !ok {
		return nil, 0, fmt.Errorf("invalid lighting design action")
	}
	if !propertyOK {
		return nil, 0, fmt.Errorf("invalid lighting design property")
	}
	match, _, _ := findEntity(entityGetTarget{id: deviceID, entityType: "device"}, entities.Entities)
	if match.ID == "" {
		return nil, 0, fmt.Errorf("lighting design device %s not found before write", deviceID)
	}
	execution, err := api.NewDevicePropertySetClient(endpoint, nil).Run(ctx, api.DevicePropertySetRequest{
		HouseID:      houseID,
		DeviceID:     match.ID,
		PropertyName: propertyID,
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
		PropertyName: propertyID,
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
		semantic.FieldEntity:        entitySummaryMap(match),
		semantic.FieldProperty:      semantic.LightPropertyName(propertyID),
		semantic.FieldExpectedValue: value,
		semantic.FieldVerifiedValue: verification.Value,
		semantic.FieldVerified:      verified,
	}, calls, nil
}

func lightingDesignActionValue(property string, value any) (any, bool) {
	propertyID, ok := semantic.LightPropertyID(property)
	if !ok {
		return nil, false
	}
	switch propertyID {
	case semantic.InternalField(semantic.DomainAction, semantic.FieldPower):
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
	case semantic.InternalField(semantic.DomainAction, semantic.FieldBrightness):
		if parsed, ok := requestInt(value); ok && parsed >= 1 && parsed <= 100 {
			return parsed, true
		}
	case semantic.InternalField(semantic.DomainAction, semantic.FieldColorTemperature):
		if parsed, ok := requestInt(value); ok && parsed >= 2700 && parsed <= 6500 {
			return parsed, true
		}
	case semantic.InternalField(semantic.DomainAction, semantic.FieldColor):
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
			parsed, ok := lightPowerValue(contract.Request{Parameters: map[string]any{semantic.FieldValue: actualTyped}})
			return ok && parsed == expectedTyped
		}
	case int:
		parsed, ok := requestInt(actual)
		return ok && parsed == expectedTyped
	}
	return false
}
