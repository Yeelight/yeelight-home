package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestInvokeLocalOnlyControlsGatewayWithoutCloudToken(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
	input := `{"contractVersion":"1.0","requestId":"req-lan-local","locale":"zh-CN","utterance":"把客厅主灯调到65%","intent":"light.brightness.set","parameters":{"houseId":"house-1","deviceId":"device-1","brightness":65}}`

	response, stderr, code := invokeJSON(t, app, input)
	if code != exitOK || stderr != "" {
		t.Fatalf("code=%d stderr=%s response=%#v", code, stderr, response)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["backend"] != "lan" || result["verified"] != true || result["tool"] != "control_node" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeLocalPreferredFallsBackBeforeLANCall(t *testing.T) {
	var cloudCalls int
	cloud := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cloudCalls++
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer cloud.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", cloud.URL)
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, "http://127.0.0.1:1/mcp")
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer cloud-test"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	input := `{"contractVersion":"1.0","requestId":"req-lan-fallback","locale":"en-US","utterance":"turn on the living room","intent":"light.power.set","parameters":{"houseId":"house-1","nodeType":"room","nodeId":"room-1","power":true}}`

	response, stderr, code := invokeJSON(t, app, input)
	if code != exitOK || stderr != "" || cloudCalls != 1 {
		t.Fatalf("code=%d stderr=%s cloudCalls=%d response=%#v", code, stderr, cloudCalls, response)
	}
	result := response["result"].(map[string]any)
	if result["backend"] != "cloud" || result["fallbackFrom"] != "lan" || !containsJSONList(response["warnings"], "lan_fallback_to_cloud") {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeLocalOnlyUnavailableDoesNotTryCloud(t *testing.T) {
	var cloudCalls int
	cloud := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cloudCalls++
		http.Error(writer, "must not be called", http.StatusInternalServerError)
	}))
	defer cloud.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", cloud.URL)
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, "http://127.0.0.1:1/mcp")
	input := `{"contractVersion":"1.0","requestId":"req-lan-blocked","locale":"zh-CN","utterance":"开灯","intent":"light.power.set","parameters":{"houseId":"house-1","deviceId":"device-1","power":true}}`

	response, _, code := invokeJSON(t, app, input)
	if code != exitOK || cloudCalls != 0 || response["status"] != "blocked" {
		t.Fatalf("code=%d cloudCalls=%d response=%#v", code, cloudCalls, response)
	}
}

func TestInvokeLocalOnlyUnsupportedActionNeverTriesCloud(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	var cloudCalls int
	cloud := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cloudCalls++
		http.Error(writer, "must not be called", http.StatusInternalServerError)
	}))
	defer cloud.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", cloud.URL)
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
	input := `{"contractVersion":"1.0","requestId":"req-lan-action","locale":"en-US","utterance":"run device action","intent":"node.action.execute","parameters":{"houseId":"house-1","nodeType":"device","nodeId":"device-1","actionName":"pulse"}}`

	response, _, code := invokeJSON(t, app, input)
	if code != exitOK || cloudCalls != 0 || response["status"] != "blocked" {
		t.Fatalf("code=%d cloudCalls=%d response=%#v", code, cloudCalls, response)
	}
}

func TestInvokeLocalOnlyExecutesAdvertisedActionAndFlow(t *testing.T) {
	for _, test := range []struct {
		name       string
		requestID  string
		input      string
		capability string
	}{
		{name: "action", requestID: "req-lan-action-supported", capability: "pulse", input: `{"contractVersion":"1.0","requestId":"req-lan-action-supported","locale":"en-US","utterance":"run pulse","intent":"node.action.execute","parameters":{"houseId":"house-1","nodeType":"device","nodeId":"device-1","actionName":"pulse"}}`},
		{name: "flow", requestID: "req-lan-flow-supported", capability: "rainbow", input: `{"contractVersion":"1.0","requestId":"req-lan-flow-supported","locale":"en-US","utterance":"run rainbow","intent":"lighting.flow.execute","parameters":{"houseId":"house-1","nodeType":"device","nodeId":"device-1","flow":{"mode":"rainbow","duration":30}}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, gateway := newRuntimeActionGateway(t, false)
			defer gateway.Close()
			app := newTestApp(t)
			configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
			response, stderr, code := invokeJSON(t, app, test.input)
			if code != exitOK || stderr != "" || response["status"] != "success" {
				t.Fatalf("code=%d stderr=%s response=%#v", code, stderr, response)
			}
			result := response["result"].(map[string]any)
			if result["backend"] != "lan" || result["tool"] != "execute_actions" || result["verified"] != true || result["evidence"] != "gateway_ack" {
				t.Fatalf("result=%#v", result)
			}
			if strings.Contains(strings.ToLower(response["userMessage"].(string)), "verified the device state") {
				t.Fatalf("ACK-only message overclaimed state verification: %#v", response)
			}
			fixture.mu.Lock()
			requestID, capability := fixture.lastRequestID, fixture.lastCapability
			fixture.mu.Unlock()
			if requestID != test.requestID || capability != test.capability {
				t.Fatalf("requestID=%q capability=%q", requestID, capability)
			}
		})
	}
}

func TestInvokeUncertainLANActionNeverFallsBackToCloud(t *testing.T) {
	_, gateway := newRuntimeActionGateway(t, true)
	defer gateway.Close()
	var cloudCalls int
	cloud := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cloudCalls++
		http.Error(writer, "must not be called", http.StatusInternalServerError)
	}))
	defer cloud.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", cloud.URL)
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, gateway.URL+"/mcp")
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer cloud-test"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	input := `{"contractVersion":"1.0","requestId":"req-lan-action-uncertain","locale":"en-US","utterance":"run pulse","intent":"node.action.execute","parameters":{"houseId":"house-1","nodeType":"device","nodeId":"device-1","actionName":"pulse"}}`
	response, _, code := invokeJSON(t, app, input)
	if code != exitOK || cloudCalls != 0 || response["status"] != "partial" {
		t.Fatalf("code=%d cloudCalls=%d response=%#v", code, cloudCalls, response)
	}
	if response["error"].(map[string]any)["code"] != "uncertain_local_write" {
		t.Fatalf("response=%#v", response)
	}
}

func TestInvokeLocalOnlyBatchStateUsesGateway(t *testing.T) {
	gateway := newRuntimeGateway(t, false)
	defer gateway.Close()
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, gateway.URL+"/mcp")
	input := `{"contractVersion":"1.0","requestId":"req-lan-state-batch","locale":"zh-CN","utterance":"看看两个状态","intent":"state.batch.query","parameters":{"houseId":"house-1","items":[{"nodeType":"device","nodeId":"device-1","properties":["brightness","power"]}]}}`

	response, stderr, code := invokeJSON(t, app, input)
	if code != exitOK || stderr != "" || response["status"] != "success" {
		t.Fatalf("code=%d stderr=%s response=%#v", code, stderr, response)
	}
	result := response["result"].(map[string]any)
	if result["backend"] != "lan" || result["source"] != "gateway_lan_mcp" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeUncertainLANWriteNeverFallsBackToCloud(t *testing.T) {
	gateway := newRuntimeGateway(t, true)
	defer gateway.Close()
	var cloudCalls int
	cloud := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cloudCalls++
		http.Error(writer, "must not be called", http.StatusInternalServerError)
	}))
	defer cloud.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", cloud.URL)
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalPreferred, gateway.URL+"/mcp")
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer cloud-test"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	input := `{"contractVersion":"1.0","requestId":"req-lan-uncertain","locale":"en-US","utterance":"set brightness","intent":"light.brightness.set","parameters":{"houseId":"house-1","deviceId":"device-1","brightness":70}}`

	response, _, code := invokeJSON(t, app, input)
	if code != exitOK || cloudCalls != 0 || response["status"] != "partial" {
		t.Fatalf("code=%d cloudCalls=%d response=%#v", code, cloudCalls, response)
	}
	errorObject := response["error"].(map[string]any)
	if errorObject["code"] != "uncertain_local_write" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeLocalOnlyPreviewDoesNotContactGateway(t *testing.T) {
	app := newTestApp(t)
	configureLANProfile(t, app, controlModeLocalOnly, "http://127.0.0.1:1/mcp")
	input := `{"contractVersion":"1.0","requestId":"req-lan-preview","locale":"zh-CN","utterance":"预览开灯","intent":"light.power.set","parameters":{"houseId":"house-1","deviceId":"device-1","power":true},"options":{"dryRun":true}}`

	response, _, code := invokeJSON(t, app, input)
	if code != exitOK || response["status"] != "success" || response["traceId"] != "lan-runtime-preview" {
		t.Fatalf("code=%d response=%#v", code, response)
	}
}

func configureLANProfile(t *testing.T, app *app, mode, endpoint string) {
	t.Helper()
	if err := app.metadataStore.Save(credential.ProfileMetadata{
		Profile: "default", Region: "dev", HouseID: "house-1", Language: "zh-CN",
		ControlMode: mode, LANEndpoint: endpoint,
	}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
}

func invokeJSON(t *testing.T, app *app, input string) (map[string]any, string, int) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error: %v, stdout=%s", err, stdout.String())
	}
	return response, stderr.String(), code
}

func containsJSONList(value any, expected string) bool {
	items, _ := value.([]any)
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

type runtimeGateway struct {
	mu             sync.Mutex
	value          any
	uncertain      bool
	actionFlow     bool
	lastRequestID  string
	lastCapability string
}

func newRuntimeGateway(t *testing.T, uncertain bool) *httptest.Server {
	t.Helper()
	fixture := &runtimeGateway{value: float64(20), uncertain: uncertain}
	return httptest.NewServer(http.HandlerFunc(fixture.handle))
}

func newRuntimeActionGateway(t *testing.T, uncertain bool) (*runtimeGateway, *httptest.Server) {
	t.Helper()
	fixture := &runtimeGateway{value: float64(20), actionFlow: true, uncertain: uncertain}
	return fixture, httptest.NewServer(http.HandlerFunc(fixture.handle))
}

func (gateway *runtimeGateway) handle(writer http.ResponseWriter, request *http.Request) {
	var rpc struct {
		ID     any            `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	_ = json.NewDecoder(request.Body).Decode(&rpc)
	writer.Header().Set("Content-Type", "application/json")
	switch rpc.Method {
	case "initialize":
		writeRuntimeRPC(writer, rpc.ID, map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{"tools": map[string]any{}}})
	case "tools/list":
		writeRuntimeRPC(writer, rpc.ID, map[string]any{"tools": gateway.tools()})
	case "tools/call":
		gateway.callTool(writer, request, rpc.ID, rpc.Params)
	}
}

func (gateway *runtimeGateway) tools() []any {
	tools := []any{
		map[string]any{"name": "list_nodes", "description": "List gateway nodes", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"houseId": map[string]any{"type": "string"}}}},
		map[string]any{"name": "get_node_state", "description": "Get node state", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"nodeId": map[string]any{"type": "string"}, "propertyName": map[string]any{"type": "string"}}, "required": []any{"nodeId"}}},
		map[string]any{"name": "control_node", "description": "Control node property", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"nodeId": map[string]any{"type": "string"}, "propertyName": map[string]any{"type": "string"}, "value": map[string]any{}}, "required": []any{"nodeId", "propertyName", "value"}}},
		map[string]any{"name": "execute_scene", "description": "Execute scene", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"sceneId": map[string]any{"type": "string"}}, "required": []any{"sceneId"}}},
	}
	if gateway.actionFlow {
		tools = append(tools, runtimeExecuteActionsTool())
	}
	return tools
}

func runtimeExecuteActionsTool() map[string]any {
	actionProperties := map[string]any{
		"target_type": map[string]any{"type": "string", "enum": []any{"node"}},
		"type":        map[string]any{"type": "string"},
		"target_id":   map[string]any{"type": "string"},
		"operation":   map[string]any{"type": "string"},
		"capability":  map[string]any{"type": "string"},
		"value":       map[string]any{},
	}
	actionSchema := map[string]any{
		"type": "object", "required": []any{"target_type", "type", "target_id", "operation", "capability", "value"}, "properties": actionProperties,
	}
	return map[string]any{
		"name": "execute_actions", "description": "Execute device actions",
		"inputSchema": map[string]any{
			"type": "object", "required": []any{"request_id", "actions"},
			"properties": map[string]any{
				"request_id": map[string]any{"type": "string"},
				"actions":    map[string]any{"type": "array", "items": actionSchema},
			},
		},
	}
}

func (gateway *runtimeGateway) callTool(writer http.ResponseWriter, request *http.Request, id any, params map[string]any) {
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]any)
	switch name {
	case "list_nodes":
		gateway.mu.Lock()
		value := gateway.value
		gateway.mu.Unlock()
		node := map[string]any{"nodeId": "device-1", "name": "Living Light", "roomName": "Living Room", "nodeType": "device", "properties": map[string]any{"p": false, "l": value}}
		if gateway.actionFlow {
			node["supportActions"] = []any{map[string]any{"actionName": "pulse"}}
			node["supportFlows"] = []any{map[string]any{"mode": "rainbow"}}
		}
		writeRuntimeTool(writer, id, map[string]any{"nodes": []any{node}}, false)
	case "control_node":
		if gateway.uncertain {
			hijacker := writer.(http.Hijacker)
			connection, _, _ := hijacker.Hijack()
			_ = connection.Close()
			return
		}
		gateway.mu.Lock()
		gateway.value = arguments["value"]
		gateway.mu.Unlock()
		writeRuntimeTool(writer, id, map[string]any{"accepted": true}, false)
	case "get_node_state":
		if gateway.uncertain {
			writeRuntimeTool(writer, id, map[string]any{"message": "state unavailable"}, true)
			return
		}
		gateway.mu.Lock()
		value := gateway.value
		gateway.mu.Unlock()
		property, _ := arguments["propertyName"].(string)
		properties := map[string]any{"p": false, "l": value}
		if property != "" {
			properties = map[string]any{property: value}
		}
		writeRuntimeTool(writer, id, map[string]any{"nodeId": "device-1", "properties": properties}, false)
	case "execute_scene":
		writeRuntimeTool(writer, id, map[string]any{"executed": true}, false)
	case "execute_actions":
		if gateway.uncertain {
			hijacker := writer.(http.Hijacker)
			connection, _, _ := hijacker.Hijack()
			_ = connection.Close()
			return
		}
		actions, _ := arguments["actions"].([]any)
		if len(actions) != 1 {
			writeRuntimeTool(writer, id, map[string]any{"message": "one action required"}, true)
			return
		}
		action, _ := actions[0].(map[string]any)
		gateway.mu.Lock()
		gateway.lastRequestID, _ = arguments["request_id"].(string)
		gateway.lastCapability, _ = action["capability"].(string)
		gateway.mu.Unlock()
		writeRuntimeTool(writer, id, map[string]any{"accepted": true}, false)
	}
}

func writeRuntimeTool(writer http.ResponseWriter, id any, data any, isError bool) {
	writeRuntimeRPC(writer, id, map[string]any{"structuredContent": data, "content": []any{map[string]any{"type": "text", "text": "ok"}}, "isError": isError})
}

func writeRuntimeRPC(writer http.ResponseWriter, id any, result any) {
	_ = json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}
