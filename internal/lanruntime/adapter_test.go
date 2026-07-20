package lanruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

type gatewayFixture struct {
	mu              sync.Mutex
	value           any
	controlDelay    time.Duration
	applyBeforeWait bool
	stateFails      bool
	incompatible    bool
	executeActions  bool
	lastRequestID   string
	lastOperation   string
	lastCapability  string
	omitAck         bool
	applyAfterReads int
	pendingValue    any
	stateReads      int
	restrictActions bool
	actionCalls     int
}

func TestAdapterSupportsGatewayExecuteActionsSchema(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20), executeActions: true}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	result, err := adapter.Set(context.Background(), PropertyRequest{
		RequestID: "req-real-schema-1", Target: Target{HouseID: "house-1", Type: "device", ID: "device-1"}, Property: "l", Value: float64(65),
	})
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if result.Outcome != OutcomeApplied || !result.Verified || result.Tool != "execute_actions" {
		t.Fatalf("result = %#v", result)
	}
	fixture.mu.Lock()
	requestID := fixture.lastRequestID
	fixture.mu.Unlock()
	if requestID != "req-real-schema-1" {
		t.Fatalf("request_id = %q", requestID)
	}
}

func TestAdapterExecutesCapabilityGatedActionAndFlow(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20), executeActions: true}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	target := Target{HouseID: "house-1", Type: "device", ID: "device-1"}

	action, err := adapter.ExecuteAction(context.Background(), ActionRequest{
		RequestID: "req-action-1", Target: target, ActionName: "pulse", Payload: map[string]any{"speed": "slow"},
	})
	if err != nil || action.Outcome != OutcomeApplied || !action.Verified || action.Evidence != "gateway_ack" {
		t.Fatalf("action=%#v err=%v", action, err)
	}
	flow, err := adapter.ExecuteFlow(context.Background(), FlowRequest{
		RequestID: "req-flow-1", Target: target, Flow: map[string]any{"mode": "rainbow", "duration": 30},
	})
	if err != nil || flow.Outcome != OutcomeApplied || !flow.Verified {
		t.Fatalf("flow=%#v err=%v", flow, err)
	}
	fixture.mu.Lock()
	requestID, operation, capability := fixture.lastRequestID, fixture.lastOperation, fixture.lastCapability
	fixture.mu.Unlock()
	if requestID != "req-flow-1" || operation != "execute" || capability != "rainbow" {
		t.Fatalf("requestID=%q operation=%q capability=%q", requestID, operation, capability)
	}
	if _, err := adapter.ExecuteAction(context.Background(), ActionRequest{RequestID: "req-unsupported", Target: target, ActionName: "invented"}); KindOf(err) != ErrorUnsupported {
		t.Fatalf("unsupported action err=%#v", err)
	}
}

func TestAdapterRejectsFlowOutsideExecuteActionsCapabilityEnum(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20), executeActions: true, restrictActions: true}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	_, err := adapter.ExecuteFlow(context.Background(), FlowRequest{
		RequestID: "req-flow-rejected", Target: Target{HouseID: "house-1", Type: "device", ID: "device-1"}, Flow: map[string]any{"mode": "rainbow"},
	})
	if KindOf(err) != ErrorUnsupported || fixture.actionCalls != 0 {
		t.Fatalf("err=%#v actionCalls=%d", err, fixture.actionCalls)
	}
}

func TestAdapterActionWithoutAckIsUnverified(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20), executeActions: true, omitAck: true}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	result, err := adapter.ExecuteAction(context.Background(), ActionRequest{
		RequestID: "req-no-ack", Target: Target{HouseID: "house-1", Type: "device", ID: "device-1"}, ActionName: "pulse",
	})
	if err != nil || result.Outcome != OutcomeUnverified || result.Verified {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestAdapterUncertainActionIsNeverReportedApplied(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20), executeActions: true, controlDelay: 80 * time.Millisecond}
	adapter := newFixtureAdapter(t, fixture, 25*time.Millisecond)
	result, err := adapter.ExecuteAction(context.Background(), ActionRequest{
		RequestID: "req-action-uncertain", Target: Target{HouseID: "house-1", Type: "device", ID: "device-1"}, ActionName: "pulse",
	})
	if err != nil || result.Outcome != OutcomeUncertain || result.CallError == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestAdapterDiscoversToolsResolvesNameAndVerifiesWrite(t *testing.T) {
	fixture := &gatewayFixture{value: float64(20)}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	result, err := adapter.Set(context.Background(), PropertyRequest{
		Target:   Target{HouseID: "house-1", Type: "device", Name: "Living Light", Room: "Living Room"},
		Property: "l", Value: float64(65),
	})
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if result.Outcome != OutcomeApplied || !result.Verified || result.Target.ID != "device-1" || result.Tool != "control_node" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterReadUsesStateTool(t *testing.T) {
	fixture := &gatewayFixture{value: float64(42)}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	result, err := adapter.Query(context.Background(), PropertyRequest{Target: Target{ID: "device-1", Type: "device"}, Property: "brightness"})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if result.Outcome != OutcomeReadSuccess || result.Value != float64(42) || result.Tool != "get_node_state" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterTimeoutReadbackDistinguishesAppliedAndNotApplied(t *testing.T) {
	for _, test := range []struct {
		name            string
		applyBeforeWait bool
		want            Outcome
	}{
		{name: "applied", applyBeforeWait: true, want: OutcomeApplied},
		{name: "not-applied", applyBeforeWait: false, want: OutcomeNotApplied},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := &gatewayFixture{value: float64(10), controlDelay: 80 * time.Millisecond, applyBeforeWait: test.applyBeforeWait}
			adapter := newFixtureAdapter(t, fixture, 25*time.Millisecond)
			result, err := adapter.Set(context.Background(), PropertyRequest{Target: Target{ID: "device-1", Type: "device"}, Property: "l", Value: float64(70)})
			if err != nil {
				t.Fatalf("Set error: %v", err)
			}
			if result.Outcome != test.want || result.CallError == "" {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestAdapterTimeoutWithoutReadableStateIsUncertain(t *testing.T) {
	fixture := &gatewayFixture{value: float64(10), controlDelay: 80 * time.Millisecond, stateFails: true}
	adapter := newFixtureAdapter(t, fixture, 25*time.Millisecond)
	result, err := adapter.Set(context.Background(), PropertyRequest{Target: Target{ID: "device-1", Type: "device"}, Property: "l", Value: float64(70)})
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if result.Outcome != OutcomeUncertain || result.CallError == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestAdapterWaitsForDelayedStateApplication(t *testing.T) {
	fixture := &gatewayFixture{value: float64(10), applyAfterReads: 3}
	adapter := newFixtureAdapterWithVerification(t, fixture, 500*time.Millisecond, 4, time.Millisecond)
	result, err := adapter.Set(context.Background(), PropertyRequest{
		Target: Target{ID: "device-1", Type: "device"}, Property: "l", Value: float64(70),
	})
	if err != nil || result.Outcome != OutcomeApplied || !result.Verified || fixture.stateReads != 3 {
		t.Fatalf("result=%#v stateReads=%d err=%v", result, fixture.stateReads, err)
	}
}

func TestAdapterRejectsIncompatibleControlSchema(t *testing.T) {
	fixture := &gatewayFixture{value: float64(10), incompatible: true}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	_, err := adapter.Set(context.Background(), PropertyRequest{Target: Target{ID: "device-1"}, Property: "l", Value: 50})
	if err == nil {
		t.Fatal("expected incompatible control tool to be rejected")
	}
	typed, ok := err.(*Error)
	if !ok || typed.Kind != ErrorUnsupported {
		t.Fatalf("error = %#v", err)
	}
}

func TestAdapterExecutesSchemaCompatibleScene(t *testing.T) {
	adapter := newFixtureAdapter(t, &gatewayFixture{value: true}, 500*time.Millisecond)
	result, err := adapter.ExecuteScene(context.Background(), SceneRequest{Target: Target{HouseID: "house-1", ID: "scene-1", Type: "scene"}})
	if err != nil {
		t.Fatalf("ExecuteScene error: %v", err)
	}
	if result.Outcome != OutcomeApplied || result.Tool != "execute_scene" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBuildToolArgumentsSupportsNestedControlCommandSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object", "required": []any{"controlRequest"},
		"properties": map[string]any{"controlRequest": map[string]any{
			"type": "object", "required": []any{"nodeId", "nodeType", "command", "confirmSideEffect"},
			"properties": map[string]any{
				"nodeId": map[string]any{"type": "string"}, "nodeType": map[string]any{"type": "integer"},
				"confirmSideEffect": map[string]any{"type": "boolean"}, "dryRun": map[string]any{"type": "boolean"},
				"command": map[string]any{"type": "object", "required": []any{"command", "params"}, "properties": map[string]any{
					"command": map[string]any{"type": "string"}, "params": map[string]any{"type": "array"},
				}},
			},
		}},
	}
	arguments, err := buildToolArguments(schema, roleControl, operationValues{
		target: Target{ID: "device-1", Type: "device"}, action: "set", properties: map[string]any{"ct": 3000, "l": 65},
	})
	if err != nil {
		t.Fatalf("buildToolArguments error: %v", err)
	}
	want := map[string]any{"controlRequest": map[string]any{
		"nodeId": "device-1", "nodeType": 2, "confirmSideEffect": true, "dryRun": false,
		"command": map[string]any{"command": "set", "params": []any{
			map[string]any{"propName": "ct", "value": 3000}, map[string]any{"propName": "l", "value": 65},
		}},
	}}
	if !reflect.DeepEqual(arguments, want) {
		t.Fatalf("arguments = %#v", arguments)
	}
}

func TestAdapterToggleAndMultiplePropertiesReuseVerifiedSet(t *testing.T) {
	fixture := &gatewayFixture{value: false}
	adapter := newFixtureAdapter(t, fixture, 500*time.Millisecond)
	toggled, err := adapter.Toggle(context.Background(), PropertyRequest{Target: Target{ID: "device-1", Type: "device"}, Property: "p"})
	if err != nil || toggled.Outcome != OutcomeApplied || toggled.Value != true {
		t.Fatalf("toggled = %#v, err = %v", toggled, err)
	}
	multiple, err := adapter.SetProperties(context.Background(), PropertiesRequest{
		Target: Target{ID: "device-1", Type: "device"}, Properties: map[string]any{"ct": float64(3000), "l": float64(55)},
	})
	if err != nil || multiple.Outcome != OutcomeApplied || !multiple.Verified {
		t.Fatalf("multiple = %#v, err = %v", multiple, err)
	}
}

func newFixtureAdapter(t *testing.T, fixture *gatewayFixture, timeout time.Duration) *Adapter {
	return newFixtureAdapterWithVerification(t, fixture, timeout, 1, time.Millisecond)
}

func newFixtureAdapterWithVerification(t *testing.T, fixture *gatewayFixture, timeout time.Duration, attempts int, interval time.Duration) *Adapter {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(fixture.handle))
	t.Cleanup(server.Close)
	client, err := lanmcp.NewClient(server.URL+"/mcp", lanmcp.Options{Timeout: timeout, HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	adapter, err := Connect(context.Background(), Options{Client: client, VerificationAttempts: attempts, VerificationInterval: interval})
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	return adapter
}

func (fixture *gatewayFixture) handle(writer http.ResponseWriter, request *http.Request) {
	var rpc struct {
		ID     any            `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	_ = json.NewDecoder(request.Body).Decode(&rpc)
	writer.Header().Set("Content-Type", "application/json")
	switch rpc.Method {
	case "initialize":
		writeFixtureRPC(writer, rpc.ID, map[string]any{"protocolVersion": lanmcp.DefaultProtocolVersion, "capabilities": map[string]any{"tools": map[string]any{}}})
	case "tools/list":
		writeFixtureRPC(writer, rpc.ID, map[string]any{"tools": fixture.tools()})
	case "tools/call":
		fixture.callTool(writer, rpc.ID, rpc.Params)
	default:
		writeFixtureRPC(writer, rpc.ID, map[string]any{})
	}
}

func (fixture *gatewayFixture) tools() []any {
	controlSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{"controlRequest": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nodeId": map[string]any{"type": "string"}, "propertyName": map[string]any{"type": "string"}, "value": map[string]any{},
			},
			"required": []any{"nodeId", "propertyName", "value"},
		}},
		"required": []any{"controlRequest"},
	}
	if fixture.incompatible {
		controlSchema = map[string]any{"type": "object", "properties": map[string]any{"opaque": map[string]any{"type": "string"}}, "required": []any{"opaque"}}
	}
	controlName := "control_node"
	controlDescription := "Control a node"
	if fixture.executeActions {
		controlName = "execute_actions"
		controlDescription = "Execute device actions"
		controlSchema = map[string]any{
			"type": "object", "required": []any{"request_id", "actions"},
			"properties": map[string]any{"request_id": map[string]any{"type": "string"}, "actions": map[string]any{
				"type": "array", "items": map[string]any{
					"type": "object", "required": []any{"target_type", "type", "target_id", "operation", "capability", "value"},
					"properties": map[string]any{
						"target_type": map[string]any{"type": "string", "enum": []any{"node", "scene"}},
						"type":        map[string]any{"type": "string", "enum": []any{"device", "group", "room", "area", "house"}},
						"target_id":   map[string]any{"type": "string"},
						"operation":   map[string]any{"type": "string", "enum": []any{"set", "toggle", "adjust", "read", "execute"}},
						"capability":  map[string]any{"type": "string"},
						"value":       map[string]any{},
					},
				},
			}},
		}
		if fixture.restrictActions {
			actionsSchema := asMap(asMap(controlSchema["properties"])["actions"])
			itemSchema := asMap(actionsSchema["items"])
			asMap(itemSchema["properties"])["capability"] = map[string]any{"type": "string", "enum": []any{"power", "brightness", "color_rgb", "color_temperature"}}
		}
	}
	return []any{
		map[string]any{"name": "list_nodes", "description": "List gateway nodes", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"houseId": map[string]any{"type": "string"}}}},
		map[string]any{"name": "get_node", "description": "Get one normalized node by id", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []any{"id"}}},
		map[string]any{"name": "get_node_state", "description": "Get node state", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"nodeId": map[string]any{"type": "string"}, "propertyName": map[string]any{"type": "string"}}, "required": []any{"nodeId"}}},
		map[string]any{"name": controlName, "description": controlDescription, "inputSchema": controlSchema},
		map[string]any{"name": "execute_scene", "description": "Execute scene", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"sceneId": map[string]any{"type": "string"}}, "required": []any{"sceneId"}}},
	}
}

func (fixture *gatewayFixture) callTool(writer http.ResponseWriter, id any, params map[string]any) {
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]any)
	switch name {
	case "list_nodes":
		fixture.mu.Lock()
		value := fixture.value
		fixture.mu.Unlock()
		writeFixtureTool(writer, id, map[string]any{"nodes": []any{map[string]any{
			"nodeId": "device-1", "name": "Living Light", "roomName": "Living Room", "nodeType": "device",
			"properties":     map[string]any{"l": value},
			"supportActions": []any{map[string]any{"actionName": "pulse"}},
			"supportFlows":   []any{map[string]any{"mode": "rainbow"}},
		}}}, false)
	case "get_node_state":
		if fixture.stateFails {
			writeFixtureTool(writer, id, map[string]any{"message": "state unavailable"}, true)
			return
		}
		fixture.mu.Lock()
		fixture.stateReads++
		if fixture.applyAfterReads > 0 && fixture.stateReads >= fixture.applyAfterReads && fixture.pendingValue != nil {
			fixture.value = fixture.pendingValue
			fixture.pendingValue = nil
		}
		value := fixture.value
		fixture.mu.Unlock()
		property, _ := arguments["propertyName"].(string)
		if property == "" {
			property = "l"
		}
		writeFixtureTool(writer, id, map[string]any{"nodeId": "device-1", "properties": map[string]any{property: value}}, false)
	case "get_node":
		writeFixtureTool(writer, id, map[string]any{
			"nodeId": "device-1", "houseId": "house-1", "nodeType": "device", "name": "Living Light",
			"supportActions": []any{map[string]any{"actionName": "pulse"}},
			"supportFlows":   []any{map[string]any{"mode": "rainbow"}},
		}, false)
	case "control_node":
		request, _ := arguments["controlRequest"].(map[string]any)
		if fixture.applyBeforeWait {
			fixture.mu.Lock()
			fixture.value = request["value"]
			fixture.mu.Unlock()
		}
		if fixture.controlDelay > 0 {
			time.Sleep(fixture.controlDelay)
		}
		if fixture.applyAfterReads > 0 {
			fixture.mu.Lock()
			fixture.pendingValue = request["value"]
			fixture.mu.Unlock()
		} else if !fixture.applyBeforeWait && fixture.controlDelay == 0 {
			fixture.mu.Lock()
			fixture.value = request["value"]
			fixture.mu.Unlock()
		}
		writeFixtureTool(writer, id, map[string]any{"accepted": true}, false)
	case "execute_actions":
		fixture.mu.Lock()
		fixture.actionCalls++
		fixture.mu.Unlock()
		requestID, _ := arguments["request_id"].(string)
		if requestID == "" {
			writeFixtureTool(writer, id, map[string]any{"message": "missing request_id"}, true)
			return
		}
		actions, _ := arguments["actions"].([]any)
		if len(actions) == 0 {
			writeFixtureTool(writer, id, map[string]any{"message": "missing actions"}, true)
			return
		}
		action, _ := actions[0].(map[string]any)
		if action["target_type"] != "node" || action["type"] != "device" || action["target_id"] != "device-1" {
			writeFixtureTool(writer, id, map[string]any{"message": "invalid action mapping", "action": action}, true)
			return
		}
		operation, _ := action["operation"].(string)
		capability, _ := action["capability"].(string)
		if operation == "set" && capability != "brightness" || operation == "execute" && capability != "pulse" && capability != "rainbow" {
			writeFixtureTool(writer, id, map[string]any{"message": "invalid operation mapping", "action": action}, true)
			return
		}
		fixture.mu.Lock()
		fixture.lastRequestID = requestID
		fixture.lastOperation = operation
		fixture.lastCapability = capability
		if operation == "set" {
			fixture.value = action["value"]
		}
		omitAck := fixture.omitAck
		delay := fixture.controlDelay
		fixture.mu.Unlock()
		if delay > 0 {
			time.Sleep(delay)
		}
		if omitAck {
			writeFixtureTool(writer, id, map[string]any{"queued": true}, false)
			return
		}
		writeFixtureTool(writer, id, map[string]any{"accepted": true}, false)
	case "execute_scene":
		writeFixtureTool(writer, id, map[string]any{"executed": true}, false)
	}
}

func writeFixtureTool(writer http.ResponseWriter, id any, data any, isError bool) {
	writeFixtureRPC(writer, id, map[string]any{"structuredContent": data, "content": []any{map[string]any{"type": "text", "text": "ok"}}, "isError": isError})
}

func writeFixtureRPC(writer http.ResponseWriter, id any, result any) {
	_ = json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}
