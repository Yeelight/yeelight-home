package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func (app *app) preparePanelConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	payload, preconditions, summary, err := buildPanelConfigurationPayload(request)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), panelConfigurationAcceptedFields(request.Intent)), nil
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
	if !entityExists(entities, "device", valueIDString(payload["deviceId"])) {
		return configureClarificationResponse(request, "invalid_panel_device_reference", panelConfigurationAcceptedFields(request.Intent)), nil
	}
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponse(request, record, entities), nil
}

func buildPanelConfigurationPayload(request contract.Request) (map[string]any, []string, string, error) {
	target := entityGetTargetFromRequest(request)
	deviceID := firstNonEmptyString(target.id, firstRequestString(request.Parameters, "deviceId", "deviceID", "id"))
	if strings.TrimSpace(deviceID) == "" {
		return nil, nil, "", fmt.Errorf("invalid_panel_configuration_device")
	}
	switch request.Intent {
	case "panel.button.configure":
		buttons, ok := sanitizePanelButtonItems(request.Parameters["buttons"])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_configure_payload")
		}
		return map[string]any{
				"deviceId": deviceID,
				"buttons":  buttons,
			}, []string{
				"提交前重新读取面板详情和按键配置",
				"面板设备必须属于当前家庭",
				"提交后通过 panel.get 验证按钮配置",
			}, "配置面板按键", nil
	case "panel.button_event.update":
		event, ok := sanitizePanelButtonEvent(request.Parameters)
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_payload")
		}
		return map[string]any{
				"deviceId":    deviceID,
				"buttonEvent": event,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"提交后通过 panel.get 验证按钮事件配置",
			}, "更新面板按键动作", nil
	case "panel.button_event.batch_update":
		events, ok := sanitizePanelButtonEvents(request.Parameters["buttonEvents"])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_batch_payload")
		}
		return map[string]any{
				"deviceId":     deviceID,
				"buttonEvents": events,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"单次批量最多更新 20 个按键事件",
				"提交后通过 panel.get 验证按钮事件配置",
			}, "批量更新面板按键动作", nil
	case "panel.button_event.reset":
		buttonEventID := firstRequestString(request.Parameters, "buttonEventId", "eventId", "id")
		if strings.TrimSpace(buttonEventID) == "" {
			return nil, nil, "", fmt.Errorf("invalid_panel_button_event_reset_payload")
		}
		return map[string]any{
				"deviceId":      deviceID,
				"buttonEventId": buttonEventID,
			}, []string{
				"提交前重新读取面板详情和按键事件配置",
				"面板设备必须属于当前家庭",
				"重置会清空该按键事件的现有动作绑定",
				"提交后重新读取 panel.get 确认云端接受重置",
			}, "重置面板按键动作", nil
	case "knob.configure":
		details, ok := sanitizeKnobDetails(request.Parameters["details"])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_knob_configure_payload")
		}
		return map[string]any{
				"deviceId": deviceID,
				"details":  details,
			}, []string{
				"提交前重新读取旋钮详情",
				"旋钮设备必须属于当前家庭",
				"提交后通过 knob.get 验证旋钮配置",
			}, "配置旋钮", nil
	case "knob.reset":
		index, ok := requestInt(request.Parameters["index"])
		if !ok {
			return nil, nil, "", fmt.Errorf("invalid_knob_reset_payload")
		}
		return map[string]any{
				"deviceId": deviceID,
				"index":    index,
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
		clean := copyAllowedConfigFields(item, []string{"id", "deviceId", "name", "alias", "keyValue", "index", "resId", "resType", "visible", "icon", "sort", "type", "extend"})
		if len(clean) == 0 {
			return nil, false
		}
		result = append(result, clean)
	}
	return result, true
}

func sanitizeKnobDetails(value any) ([]any, bool) {
	rows, ok := value.([]any)
	if !ok || len(rows) == 0 || len(rows) > 32 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		clean := copyAllowedConfigFields(item, []string{"id", "index", "configType", "mode", "model", "resId", "typeId", "resType", "resIndex", "resName", "param", "sens", "action", "property", "value", "details"})
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
	buttonEventID := firstRequestString(item, "buttonEventId", "eventId", "id")
	if strings.TrimSpace(buttonEventID) == "" {
		return nil, false
	}
	details, ok := sanitizePanelButtonEventDetails(item["details"])
	if !ok {
		return nil, false
	}
	clean := map[string]any{
		"buttonEventId": buttonEventID,
		"details":       details,
	}
	if alias, ok := item["alias"].(string); ok && strings.TrimSpace(alias) != "" {
		clean["alias"] = strings.TrimSpace(alias)
	}
	return clean, true
}

func sanitizePanelButtonEventDetails(value any) ([]any, bool) {
	rows, ok := value.([]any)
	if !ok || len(rows) == 0 || len(rows) > 32 {
		return nil, false
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			return nil, false
		}
		clean := copyAllowedConfigFields(item, []string{"id", "roomId", "resId", "typeId", "resType", "idx", "params", "rank", "resName", "repeatType", "repeatValue", "startTime", "endTime", "action", "property", "value", "delay", "duration"})
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
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.buttons"}
	case "panel.button_event.update":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.buttonEventId", "parameters.alias", "parameters.details"}
	case "panel.button_event.batch_update":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.buttonEvents[].buttonEventId", "parameters.buttonEvents[].details"}
	case "panel.button_event.reset":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.buttonEventId"}
	case "knob.configure":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.details"}
	case "knob.reset":
		return []string{"parameters.houseId", "parameters.deviceId", "parameters.index"}
	default:
		return []string{"parameters.houseId", "parameters.deviceId"}
	}
}

func (app *app) executePanelConfiguration(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, kind api.PanelConfigurationKind) (contract.Response, error) {
	deviceID := planPayloadString(record.Payload, "deviceId")
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
