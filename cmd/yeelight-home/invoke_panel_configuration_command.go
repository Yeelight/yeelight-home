package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) preparePanelConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", missingHouseIDAcceptedFields()), nil
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
	payload, preconditions, summary, err := buildPanelConfigurationPayload(request, entities)
	if err != nil {
		return panelConfigurationClarificationResponse(request, err.Error()), nil
	}
	if !entityExists(entities, "device", valueIDString(payload[semantic.FieldDeviceID])) {
		return panelConfigurationClarificationResponse(request, "invalid_panel_device_reference"), nil
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func buildPanelConfigurationPayload(request contract.Request, entities api.EntityListResult) (map[string]any, []string, string, error) {
	deviceID, err := resolvePanelConfigurationDeviceID(request, entities)
	if err != nil {
		return nil, nil, "", err
	}
	if strings.TrimSpace(deviceID) == "" {
		return nil, nil, "", fmt.Errorf("invalid_panel_configuration_device")
	}
	switch request.Intent {
	case "panel.button.configure":
		buttons, ok := sanitizePanelButtonItems(request.Parameters[semantic.FieldButtons])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_configure_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID: deviceID,
				semantic.FieldButtons:  buttons,
			}, []string{
				"提交前重新读取面板详情和按键配置",
				"面板设备必须属于当前家庭",
				"提交后通过 panel.get 验证按钮配置",
			}, "配置面板按键", nil
	case "panel.button_event.update":
		eventSource := request.Parameters
		if nestedEvent, ok := request.Parameters[semantic.FieldButtonEvent].(map[string]any); ok {
			eventSource = nestedEvent
		}
		event, ok := sanitizePanelButtonEvent(eventSource)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID:    deviceID,
				semantic.FieldButtonEvent: event,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"提交后通过 panel.get 验证按钮事件配置",
			}, "更新面板按键动作", nil
	case "panel.button_event.batch_update":
		events, ok := sanitizePanelButtonEvents(request.Parameters[semantic.FieldButtonEvents])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_batch_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID:     deviceID,
				semantic.FieldButtonEvents: events,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"单次批量最多更新 20 个按键事件",
				"提交后通过 panel.get 验证按钮事件配置",
			}, "批量更新面板按键动作", nil
	case "panel.button_event.reset":
		buttonEventID := firstRequestString(request.Parameters, semantic.FieldButtonEventID, semantic.FieldEventID, semantic.FieldID)
		if strings.TrimSpace(buttonEventID) == "" {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_reset_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID:      deviceID,
				semantic.FieldButtonEventID: buttonEventID,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"重置会清空该按键事件的现有动作绑定",
				"提交后重新读取 panel.get 确认云端接受重置",
			}, "重置面板按键动作", nil
	case "knob.configure":
		actions, ok := sanitizeKnobDetails(request.Parameters[semantic.FieldActions])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_knob_configure_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID: deviceID,
				semantic.FieldDetails:  actions,
			}, []string{
				"提交前重新读取旋钮详情",
				"旋钮设备必须属于当前家庭",
				"提交后通过 knob.get 验证旋钮配置",
			}, "配置旋钮", nil
	case "knob.reset":
		index, ok := requestInt(request.Parameters[semantic.FieldIndex])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_knob_reset_payload")
		}
		return map[string]any{
				semantic.FieldDeviceID: deviceID,
				semantic.FieldIndex:    index,
			}, []string{
				"提交前重新读取旋钮详情",
				"旋钮设备必须属于当前家庭",
				"重置会清空该旋钮子键位的现有动作绑定",
				"提交后重新读取 knob.get 确认云端接受重置",
			}, "重置旋钮子键位", nil
	default:
		return nil, nil, "", fmt.Errorf("unsupported_panel_configuration_intent")
	}
}

func resolvePanelConfigurationDeviceID(request contract.Request, entities api.EntityListResult) (string, error) {
	deviceID := firstRequestString(request.Parameters, semantic.FieldDeviceID, semantic.FieldPanelID, semantic.FieldKnobID)
	target := entityGetTargetFromRequest(request)
	if deviceID == "" && target.entityType == "device" {
		deviceID = target.id
	}
	if deviceID != "" {
		return deviceID, nil
	}
	deviceName := firstRequestString(request.Parameters, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName)
	if deviceName == "" && target.entityType == "device" {
		deviceName = target.name
	}
	if deviceName == "" {
		return "", fmt.Errorf("invalid_panel_configuration_device")
	}
	match, candidates, _ := findEntity(entityGetTarget{name: deviceName, entityType: "device", roomID: target.roomID, roomName: target.roomName}, entities.Entities)
	if match.ID != "" && len(candidates) == 1 {
		return match.ID, nil
	}
	if len(candidates) > 1 {
		return "", fmt.Errorf("ambiguous_panel_device_reference")
	}
	return "", fmt.Errorf("invalid_panel_device_reference")
}

func sanitizePanelButtonItems(value any) ([]any, bool) {
	rows, ok := value.([]any)
	if !ok || len(rows) == 0 || len(rows) > 64 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		item = normalizeTargetBinding(item, groupTypeCustom, semantic.InternalField(semantic.DomainPanel, semantic.FieldTargetType))
		clean := copyAllowedConfigFields(item, semantic.PanelButtonWriteFields())
		if len(clean) == 0 {
			return nil, false
		}
		result = append(result, clean)
	}
	return result, true
}

func sanitizeKnobDetails(value any) ([]any, bool) {
	rows, ok := normalizePanelActionRows(value)
	if !ok || len(rows) == 0 || len(rows) > 32 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		clean := copyAllowedConfigFields(item, semantic.KnobDetailWriteFields())
		if len(clean) == 0 {
			return nil, false
		}
		result = append(result, clean)
	}
	return result, true
}

func sanitizePanelButtonEvents(value any) ([]any, bool) {
	rows, ok := value.([]any)
	if !ok || len(rows) == 0 || len(rows) > 20 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		clean, ok := sanitizePanelButtonEvent(item)
		if !ok {
			return nil, false
		}
		result = append(result, clean)
	}
	return result, true
}

func sanitizePanelButtonEvent(item map[string]any) (map[string]any, bool) {
	buttonEventID := firstRequestString(item, semantic.FieldButtonEventID, semantic.FieldEventID, semantic.FieldID)
	if strings.TrimSpace(buttonEventID) == "" {
		return nil, false
	}
	actions, ok := sanitizePanelButtonEventDetails(item[semantic.FieldActions])
	if !ok {
		return nil, false
	}
	clean := map[string]any{
		semantic.FieldButtonEventID: buttonEventID,
		semantic.FieldDetails:       actions,
	}
	if alias, ok := item[semantic.FieldAlias].(string); ok && strings.TrimSpace(alias) != "" {
		clean[semantic.FieldAlias] = strings.TrimSpace(alias)
	}
	return clean, true
}

func sanitizePanelButtonEventDetails(value any) ([]any, bool) {
	rows, ok := normalizePanelActionRows(value)
	if !ok || len(rows) == 0 || len(rows) > 32 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		clean := copyAllowedConfigFields(item, semantic.PanelEventDetailWriteFields())
		if len(clean) == 0 {
			return nil, false
		}
		result = append(result, clean)
	}
	return result, true
}

func copyAllowedConfigFields(source map[string]any, keys []string) map[string]any {
	result := map[string]any{}
	for _, key := range keys {
		value, ok := source[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				result[key] = strings.TrimSpace(typed)
			}
		case float64, int, int64, bool:
			result[key] = typed
		case map[string]any, []any:
			result[key] = typed
		}
	}
	return result
}

func panelConfigurationAcceptedFields(intent string) []string {
	switch intent {
	case "panel.button.configure":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldButtons)
	case "panel.button_event.update":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldDeviceID), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldEntityName), semantic.ParameterPath(semantic.FieldTargetName), semantic.ParameterPath(semantic.FieldButtonEvent), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.FieldButtonEventID), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.FieldAlias), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.FieldActions), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetType), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetID), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetName), semantic.ParameterPath(semantic.FieldButtonEvent, semantic.ArrayField(semantic.FieldActions), semantic.FieldSet), semantic.ParameterPath(semantic.FieldButtonEventID), semantic.ParameterPath(semantic.FieldAlias), semantic.ParameterPath(semantic.FieldActions), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet)}
	case "panel.button_event.batch_update":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldDeviceID), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldEntityName), semantic.ParameterPath(semantic.FieldTargetName), semanticParameterArrayPath(semantic.FieldButtonEvents, semantic.FieldButtonEventID), semanticParameterArrayPath(semantic.FieldButtonEvents, semantic.FieldActions), semantic.ParameterPath(semantic.ArrayField(semantic.FieldButtonEvents), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetType), semantic.ParameterPath(semantic.ArrayField(semantic.FieldButtonEvents), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetID), semantic.ParameterPath(semantic.ArrayField(semantic.FieldButtonEvents), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetName), semantic.ParameterPath(semantic.ArrayField(semantic.FieldButtonEvents), semantic.ArrayField(semantic.FieldActions), semantic.FieldSet)}
	case "panel.button_event.reset":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldButtonEventID, semantic.FieldIndex)
	case "knob.configure":
		return []string{semantic.ParameterPath(semantic.FieldHouseID), semantic.ParameterPath(semantic.FieldDeviceID), semantic.ParameterPath(semantic.FieldDeviceName), semantic.ParameterPath(semantic.FieldEntityName), semantic.ParameterPath(semantic.FieldTargetName), semantic.ParameterPath(semantic.FieldActions), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetType), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetID), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldTargetName), semanticParameterArrayPath(semantic.FieldActions, semantic.FieldSet)}
	case "knob.reset":
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldIndex)
	default:
		return semanticParameterPaths(semantic.FieldHouseID, semantic.FieldDeviceID, semantic.FieldDeviceName, semantic.FieldEntityName, semantic.FieldTargetName)
	}
}

func panelConfigurationClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, panelConfigurationAcceptedFields(request.Intent), payloadGuideForIntent(request.Intent))
}

func (app *app) executePanelConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.PanelConfigurationKind) (contract.Response, error) {
	deviceID := executionPayloadString(record.Payload, semantic.FieldDeviceID)
	result, err := api.NewPanelConfigurationClient(endpoint, nil).Run(ctx, api.PanelConfigurationRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		DeviceID:       deviceID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.PanelConfigurationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return panelConfigurationExecuteResponse(request, record, result), nil
}
