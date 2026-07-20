package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/lanmcp"
	"github.com/yeelight/yeelight-home/internal/lanruntime"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

type lanRouteResult struct {
	Response contract.Response
	Handled  bool
	Fallback bool
}

func (app *app) tryInvokeLAN(ctx context.Context, request contract.Request, contextInfo runtimeContext) lanRouteResult {
	if contextInfo.ControlMode == controlModeCloud || !lanRoutableIntent(request.Intent) {
		return lanRouteResult{}
	}
	if shouldReturnPreviewOnly(request, cliFlags{values: map[string]string{}}) && lanWriteIntent(request.Intent) {
		return lanRouteResult{Response: lanPreviewResponse(request, contextInfo), Handled: true}
	}
	client, err := lanmcp.NewClient(contextInfo.LANEndpoint, lanmcp.Options{})
	if err != nil {
		return lanRouteFailure(request, contextInfo.ControlMode, err)
	}
	adapter, err := lanruntime.Connect(ctx, lanruntime.Options{Client: client})
	if err != nil {
		return lanRouteFailure(request, contextInfo.ControlMode, err)
	}
	result, err := invokeLANOperation(ctx, adapter, request, contextInfo.HouseID)
	if err != nil {
		return lanRouteFailure(request, contextInfo.ControlMode, err)
	}
	if result.Outcome == lanruntime.OutcomeNotApplied && contextInfo.ControlMode == controlModeLocalPreferred {
		return lanRouteResult{Fallback: true}
	}
	return lanRouteResult{Response: lanOperationResponse(request, result), Handled: true}
}

func invokeLANOperation(ctx context.Context, adapter *lanruntime.Adapter, request contract.Request, defaultHouseID string) (lanruntime.Result, error) {
	target := lanTargetFromRequest(request, defaultHouseID)
	switch request.Intent {
	case "state.query":
		return adapter.Query(ctx, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: stateQueryPropertyName(request)})
	case "state.batch.query":
		queries := lanStateBatchRequests(request, defaultHouseID)
		if len(queries) == 0 {
			return lanruntime.Result{}, fmt.Errorf("at least one state query target is required")
		}
		return adapter.BatchQuery(ctx, queries)
	case "light.power.set":
		return invokeLANLightSet(ctx, adapter, request, target, lightPowerSpec())
	case "light.brightness.set":
		return invokeLANLightSet(ctx, adapter, request, target, lightBrightnessSpec())
	case "light.color_temperature.set":
		return invokeLANLightSet(ctx, adapter, request, target, lightColorTemperatureSpec())
	case "light.color.set":
		return invokeLANLightSet(ctx, adapter, request, target, lightColorSpec())
	case "light.brightness.adjust":
		return invokeLANLightAdjust(ctx, adapter, request, target, lightBrightnessAdjustSpec())
	case "light.color_temperature.adjust":
		return invokeLANLightAdjust(ctx, adapter, request, target, lightColorTemperatureAdjustSpec())
	case "device.property.set", "node.property.set":
		property := devicePropertySetPropertyName(request)
		value, ok := request.Parameters[semantic.FieldValue]
		if property == "" || !ok {
			return lanruntime.Result{}, fmt.Errorf("property and value are required")
		}
		return adapter.Set(ctx, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: property, Value: value})
	case "node.property.toggle":
		property := devicePropertySetPropertyName(request)
		if property == "" || semantic.PropertySensitive(property) {
			return lanruntime.Result{}, fmt.Errorf("a writable property is required")
		}
		return adapter.Toggle(ctx, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: property})
	case "node.properties.set":
		properties, err := lanWritableProperties(request)
		if err != nil {
			return lanruntime.Result{}, err
		}
		return adapter.SetProperties(ctx, lanruntime.PropertiesRequest{RequestID: request.RequestID, Target: target, Properties: properties})
	case "node.property.batch_set":
		property := devicePropertySetPropertyName(request)
		value, ok := request.Parameters[semantic.FieldValue]
		nodeType := firstRequestString(request.Parameters, semantic.FieldNodeType, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldType)
		ids := nodeBatchIDsFromRequest(request, nodeType, firstNonEmptyString(requestHouseID(request), defaultHouseID))
		if property == "" || semantic.PropertySensitive(property) || !ok || len(ids) == 0 {
			return lanruntime.Result{}, fmt.Errorf("node type, ids, writable property, and value are required")
		}
		targets := make([]lanruntime.Target, 0, len(ids))
		for _, id := range ids {
			targets = append(targets, lanruntime.Target{HouseID: firstNonEmptyString(requestHouseID(request), defaultHouseID), Type: nodeType, ID: id})
		}
		return adapter.BatchSet(ctx, lanruntime.BatchPropertyRequest{RequestID: request.RequestID, Targets: targets, Property: property, Value: value})
	case "node.action.execute":
		actionName := firstRequestString(request.Parameters, semantic.FieldActionName, semantic.FieldAction, semantic.FieldName)
		if actionName == "" {
			return lanruntime.Result{}, fmt.Errorf("action name is required")
		}
		return adapter.ExecuteAction(ctx, lanruntime.ActionRequest{
			RequestID: request.RequestID, Target: target, ActionName: actionName,
			Payload: controlPayloadFromRequest(request), Duration: request.Parameters[semantic.FieldDuration], Delay: request.Parameters[semantic.FieldDelay],
		})
	case "lighting.flow.execute":
		flow := firstNonNil(request.Parameters[semantic.FieldFlow], request.Parameters[semantic.FieldPayload])
		if flow == nil {
			return lanruntime.Result{}, fmt.Errorf("flow name or mode is required")
		}
		return adapter.ExecuteFlow(ctx, lanruntime.FlowRequest{
			RequestID: request.RequestID, Target: target, Flow: flow,
			Payload: controlPayloadFromRequest(request), Duration: request.Parameters[semantic.FieldDuration], Delay: request.Parameters[semantic.FieldDelay],
		})
	case "scene.execute":
		return adapter.ExecuteScene(ctx, lanruntime.SceneRequest{RequestID: request.RequestID, Target: target})
	default:
		return lanruntime.Result{}, &lanruntime.Error{Kind: lanruntime.ErrorUnsupported, Stage: "route", Message: "intent is not mapped to LAN control"}
	}
}

func lanWritableProperties(request contract.Request) (map[string]any, error) {
	properties := nodePropertiesFromRequest(request)
	if len(properties) == 0 {
		return nil, fmt.Errorf("at least one writable property is required")
	}
	for property := range properties {
		if semantic.PropertySensitive(property) {
			return nil, fmt.Errorf("sensitive property is not allowed")
		}
	}
	return properties, nil
}

func lanStateBatchRequests(request contract.Request, defaultHouseID string) []lanruntime.PropertyRequest {
	houseID := firstNonEmptyString(requestHouseID(request), defaultHouseID)
	items := stateBatchItemsFromRequest(request, houseID)
	queries := []lanruntime.PropertyRequest{}
	for _, item := range items {
		id := firstNonEmptyString(item.nodeID, item.deviceID)
		nodeType := item.nodeType
		if nodeType == "" && item.deviceID != "" {
			nodeType = "device"
		}
		target := lanruntime.Target{HouseID: houseID, Type: nodeType, ID: id}
		if len(item.propertySet) == 0 {
			queries = append(queries, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: item.propertyName})
			continue
		}
		for _, property := range item.propertySet {
			queries = append(queries, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: property})
		}
	}
	return queries
}

func invokeLANLightSet(ctx context.Context, adapter *lanruntime.Adapter, request contract.Request, target lanruntime.Target, spec lightPropertySpec) (lanruntime.Result, error) {
	value, expected, ok := spec.resolveValue(request)
	if !ok {
		return lanruntime.Result{}, fmt.Errorf("required light value is missing or invalid")
	}
	return adapter.Set(ctx, lanruntime.PropertyRequest{RequestID: request.RequestID, Target: target, Property: spec.propertyID, Value: firstExpectedValue(value, expected)})
}

func invokeLANLightAdjust(ctx context.Context, adapter *lanruntime.Adapter, request contract.Request, target lanruntime.Target, spec lightAdjustSpec) (lanruntime.Result, error) {
	delta, ok := spec.resolveDelta(request)
	if !ok {
		return lanruntime.Result{}, fmt.Errorf("required light adjustment is missing or invalid")
	}
	return adapter.Adjust(ctx, lanruntime.AdjustRequest{RequestID: request.RequestID, Target: target, Property: spec.propertyID, Delta: float64(delta), Min: float64(spec.min), Max: float64(spec.max)})
}

func firstExpectedValue(writeValue, expectedValue any) any {
	if expectedValue != nil {
		return expectedValue
	}
	return writeValue
}

func lanTargetFromRequest(request contract.Request, defaultHouseID string) lanruntime.Target {
	target := entityGetTargetFromRequest(request)
	houseID := firstNonEmptyString(requestHouseID(request), defaultHouseID)
	return lanruntime.Target{HouseID: houseID, Type: target.entityType, ID: target.id, Name: target.name, Room: target.roomName}
}

func lanRouteFailure(request contract.Request, mode string, err error) lanRouteResult {
	kind := lanruntime.KindOf(err)
	if mode == controlModeLocalPreferred && kind != lanruntime.ErrorUncertain {
		return lanRouteResult{Fallback: true}
	}
	code := "lan_backend_unavailable"
	status := "blocked"
	if kind == lanruntime.ErrorUncertain {
		code, status = "uncertain_local_write", "partial"
	}
	return lanRouteResult{Handled: true, Response: contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     localizedLANMessage(request.Locale, code),
		Result: map[string]any{
			"backend":                 "lan",
			semantic.FieldSafeToRetry: false,
			semantic.FieldNextAction:  "check_gateway_state_before_retrying",
		},
		Warnings: []string{code},
		TraceID:  "lan-runtime-route",
		Metrics:  noAPIMetrics(),
		Error:    &contract.Error{Code: code, Message: redactLANError(err)},
	}}
}

func lanOperationResponse(request contract.Request, result lanruntime.Result) contract.Response {
	status := "success"
	warnings := []string{}
	if result.Outcome == lanruntime.OutcomeUncertain {
		status = "partial"
		warnings = append(warnings, "uncertain_local_write")
	} else if result.Outcome == lanruntime.OutcomeUnverified {
		status = "partial"
		warnings = append(warnings, "lan_write_not_verified")
	} else if result.Outcome == lanruntime.OutcomeNotApplied {
		status = "partial"
		warnings = append(warnings, "lan_write_not_applied")
	}
	response := contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     localizedLANMessage(request.Locale, lanResultMessageCode(result)),
		Result: map[string]any{
			"backend":                   "lan",
			semantic.FieldSource:        "gateway_lan_mcp",
			"tool":                      result.Tool,
			semantic.FieldEntity:        map[string]any{"id": result.Target.ID, "name": result.Target.Name, "type": result.Target.Type, "roomName": result.Target.Room},
			semantic.FieldProperty:      semantic.PropertyName(result.Property),
			semantic.FieldExpectedValue: result.ExpectedValue,
			semantic.FieldVerifiedValue: result.Value,
			semantic.FieldVerified:      result.Verified,
			"evidence":                  result.Evidence,
			"outcome":                   result.Outcome,
			"data":                      result.Data,
		},
		Warnings: warnings,
		TraceID:  "lan-runtime-control",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
	if result.Outcome == lanruntime.OutcomeUncertain {
		response.Error = &contract.Error{Code: "uncertain_local_write", Message: result.CallError}
	}
	return response
}

func lanResultMessageCode(result lanruntime.Result) string {
	if result.Outcome == lanruntime.OutcomeApplied && result.Evidence == "gateway_ack" {
		return "acknowledged"
	}
	return string(result.Outcome)
}

func lanPreviewResponse(request contract.Request, contextInfo runtimeContext) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     localizedLANMessage(request.Locale, "preview"),
		Result: map[string]any{
			"backend": "lan", "endpointConfigured": contextInfo.LANEndpoint != "",
			semantic.FieldDryRun: true, semantic.FieldPersistentWrites: false,
		},
		Warnings: []string{"lan_preview_no_gateway_write"},
		TraceID:  "lan-runtime-preview",
		Metrics:  noAPIMetrics(),
	}
}

func localizedLANMessage(locale, code string) string {
	zh := map[string]string{
		"applied":                 "已通过家庭网关在局域网内完成操作，并核对了设备状态。",
		"acknowledged":            "家庭网关已确认接收并执行这项局域网操作。",
		"read_success":            "已通过家庭网关读取本地设备状态。",
		"unverified":              "局域网操作已发送，但网关暂时无法提供可核对的状态。",
		"not_applied":             "局域网操作未在设备状态中生效。",
		"uncertain":               "局域网连接在操作后中断，当前结果无法确认；为避免重复控制，未改走云端。",
		"uncertain_local_write":   "局域网操作结果无法确认；请先查看设备状态，不要立即重复执行。",
		"lan_backend_unavailable": "当前无法使用家庭网关的局域网能力。",
		"preview":                 "已生成局域网操作预览，没有控制真实设备。",
	}
	en := map[string]string{
		"applied":                 "Completed the operation through the home gateway and verified the device state.",
		"acknowledged":            "The home gateway acknowledged and executed this LAN operation.",
		"read_success":            "Read the local device state through the home gateway.",
		"unverified":              "The LAN operation was sent, but the gateway could not provide a state to verify.",
		"not_applied":             "The LAN operation was not reflected in the device state.",
		"uncertain":               "The LAN connection ended after the operation. The result is uncertain, so no cloud retry was attempted.",
		"uncertain_local_write":   "The LAN operation result is uncertain. Check device state before trying again.",
		"lan_backend_unavailable": "The home gateway LAN capability is not currently available.",
		"preview":                 "Prepared a LAN operation preview without controlling a real device.",
	}
	if locale == i18n.English {
		return firstNonEmptyString(en[code], en["lan_backend_unavailable"])
	}
	return firstNonEmptyString(zh[code], zh["lan_backend_unavailable"])
}

func redactLANError(err error) string {
	message := strings.TrimSpace(fmt.Sprint(err))
	message = strings.ReplaceAll(message, "Authorization", "credential")
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}

func lanRoutableIntent(intent string) bool {
	switch intent {
	case "state.query", "light.power.set", "light.brightness.set", "light.brightness.adjust",
		"light.color_temperature.set", "light.color_temperature.adjust", "light.color.set",
		"device.property.set", "node.property.set", "node.property.toggle", "node.properties.set",
		"node.property.batch_set", "node.action.execute", "lighting.flow.execute", "state.batch.query", "scene.execute":
		return true
	default:
		return false
	}
}

func lanWriteIntent(intent string) bool {
	return lanRoutableIntent(intent) && intent != "state.query" && intent != "state.batch.query"
}
