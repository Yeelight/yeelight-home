package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/i18n"
)

type mcpInvokeArguments struct {
	Locale     string           `json:"locale,omitempty"`
	Utterance  string           `json:"utterance,omitempty"`
	Intent     string           `json:"intent"`
	HomeRef    map[string]any   `json:"homeRef,omitempty"`
	Targets    []map[string]any `json:"targets,omitempty"`
	Parameters map[string]any   `json:"parameters,omitempty"`
	Options    map[string]any   `json:"options,omitempty"`
}

type mcpScopeArguments struct {
	Locale  string `json:"locale,omitempty"`
	HouseID string `json:"houseId,omitempty"`
}

type mcpStateArguments struct {
	Locale     string `json:"locale,omitempty"`
	HouseID    string `json:"houseId,omitempty"`
	DeviceID   string `json:"deviceId,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	RoomName   string `json:"roomName,omitempty"`
}

type mcpLightArguments struct {
	Locale     string `json:"locale,omitempty"`
	HouseID    string `json:"houseId,omitempty"`
	Action     string `json:"action"`
	Value      any    `json:"value"`
	TargetType string `json:"targetType,omitempty"`
	TargetID   string `json:"targetId,omitempty"`
	TargetName string `json:"targetName,omitempty"`
	RoomName   string `json:"roomName,omitempty"`
}

type mcpSceneArguments struct {
	Locale    string `json:"locale,omitempty"`
	HouseID   string `json:"houseId,omitempty"`
	SceneID   string `json:"sceneId,omitempty"`
	SceneName string `json:"sceneName,omitempty"`
}

type mcpExplainArguments struct {
	Locale string `json:"locale,omitempty"`
	Intent string `json:"intent"`
}

func (server *localMCPServer) invokeTool(ctx context.Context, name string, arguments map[string]any) (contract.Response, error) {
	var request contract.Request
	switch name {
	case "yeelight_home_invoke":
		var args mcpInvokeArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		request = server.newRuntimeRequest(args.Locale, args.Utterance, args.Intent, args.HomeRef, args.Targets, args.Parameters, args.Options)
	case "yeelight_home_get_home":
		var args mcpScopeArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		request = server.simpleRequest(args.Locale, "读取当前易来家庭概况", "home.summary", scopeParameters(args.HouseID))
	case "yeelight_home_list_entities":
		var args mcpScopeArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		request = server.simpleRequest(args.Locale, "列出当前易来家庭的房间、设备、设备组和情景", "entity.list", scopeParameters(args.HouseID))
	case "yeelight_home_get_state":
		var args mcpStateArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		parameters := scopeParameters(args.HouseID)
		putNonEmpty(parameters, "deviceId", args.DeviceID)
		putNonEmpty(parameters, "deviceName", args.DeviceName)
		putNonEmpty(parameters, "roomName", args.RoomName)
		if args.DeviceID == "" && args.DeviceName == "" {
			return contract.Response{}, fmt.Errorf("deviceId or deviceName is required")
		}
		request = server.simpleRequest(args.Locale, "查询易来设备的当前状态", "state.query", parameters)
	case "yeelight_home_control_light":
		var args mcpLightArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		built, err := server.lightRequest(args)
		if err != nil {
			return contract.Response{}, err
		}
		request = built
	case "yeelight_home_run_scene":
		var args mcpSceneArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		parameters := scopeParameters(args.HouseID)
		putNonEmpty(parameters, "sceneId", args.SceneID)
		putNonEmpty(parameters, "sceneName", args.SceneName)
		if args.SceneID == "" && args.SceneName == "" {
			return contract.Response{}, fmt.Errorf("sceneId or sceneName is required")
		}
		request = server.simpleRequest(args.Locale, "执行易来家庭情景", "scene.execute", parameters)
	case "yeelight_home_explain":
		var args mcpExplainArguments
		if err := decodeToolArguments(arguments, &args); err != nil {
			return contract.Response{}, err
		}
		if strings.TrimSpace(args.Intent) == "" {
			return contract.Response{}, fmt.Errorf("intent is required")
		}
		request = server.simpleRequest(args.Locale, "解释易来家庭能力的调用方式", "intent.explain", map[string]any{"intent": args.Intent})
	default:
		return contract.Response{}, fmt.Errorf("unknown tool %q", name)
	}
	validated, err := validateRuntimeRequest(request)
	if err != nil {
		return contract.Response{}, err
	}
	response, err := server.app.invokeWithFlags(ctx, validated, server.flags)
	if err != nil {
		return invokeErrorResponse(validated, err), nil
	}
	return response, nil
}

func (server *localMCPServer) lightRequest(args mcpLightArguments) (contract.Request, error) {
	intentByAction := map[string]string{
		"power": "light.power.set", "brightness": "light.brightness.set",
		"color_temperature": "light.color_temperature.set", "color": "light.color.set",
	}
	action := strings.ToLower(strings.TrimSpace(args.Action))
	intent := intentByAction[action]
	if intent == "" {
		return contract.Request{}, fmt.Errorf("action must be power, brightness, color_temperature, or color")
	}
	if args.Value == nil {
		return contract.Request{}, fmt.Errorf("value is required")
	}
	if args.TargetID == "" && args.TargetName == "" {
		return contract.Request{}, fmt.Errorf("targetId or targetName is required")
	}
	parameters := scopeParameters(args.HouseID)
	putNonEmpty(parameters, "targetType", args.TargetType)
	putNonEmpty(parameters, "targetId", args.TargetID)
	putNonEmpty(parameters, "name", args.TargetName)
	putNonEmpty(parameters, "roomName", args.RoomName)
	valueKey := map[string]string{
		"power": "power", "brightness": "brightness",
		"color_temperature": "colorTemperature", "color": "color",
	}[action]
	parameters[valueKey] = args.Value
	return server.simpleRequest(args.Locale, "控制易来灯光", intent, parameters), nil
}

func (server *localMCPServer) simpleRequest(locale string, utterance string, intent string, parameters map[string]any) contract.Request {
	return server.newRuntimeRequest(locale, utterance, intent, nil, nil, parameters, nil)
}

func (server *localMCPServer) newRuntimeRequest(locale, utterance, intent string, homeRef map[string]any, targets []map[string]any, parameters, options map[string]any) contract.Request {
	server.requestSeq++
	if strings.TrimSpace(locale) == "" {
		locale = server.locale
	} else if normalized, ok := i18n.Normalize(locale); ok {
		locale = normalized
	}
	if strings.TrimSpace(utterance) == "" {
		utterance = "通过 Yeelight Home 执行 " + intent
	}
	return contract.Request{
		ContractVersion: contract.Version,
		RequestID:       fmt.Sprintf("mcp-%d", server.requestSeq),
		Locale:          locale,
		Utterance:       utterance,
		Intent:          intent,
		HomeRef:         homeRef,
		Targets:         targets,
		Parameters:      parameters,
		Options:         options,
	}
}

func validateRuntimeRequest(request contract.Request) (contract.Request, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return contract.Request{}, err
	}
	return contract.DecodeRequest(data)
}

func decodeToolArguments(arguments map[string]any, target any) error {
	data, err := json.Marshal(arguments)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}

func scopeParameters(houseID string) map[string]any {
	parameters := map[string]any{}
	putNonEmpty(parameters, "houseId", houseID)
	return parameters
}

func putNonEmpty(target map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		target[key] = strings.TrimSpace(value)
	}
}

func mcpToolResponseResult(response contract.Response) map[string]any {
	text := strings.TrimSpace(response.UserMessage)
	if text == "" {
		text = response.Status
	}
	return map[string]any{
		"content":           []any{map[string]any{"type": "text", "text": text}},
		"structuredContent": response,
		"isError":           mcpResponseIsError(response.Status),
	}
}

func mcpToolErrorResult(err error) map[string]any {
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": err.Error()}},
		"isError": true,
	}
}

func mcpResponseIsError(status string) bool {
	switch status {
	case "error", "auth_required", "not_supported", "blocked":
		return true
	default:
		return false
	}
}
