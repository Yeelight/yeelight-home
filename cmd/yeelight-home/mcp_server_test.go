package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPStdioInitializeAndListTools(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"production-client","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n")
	responses, stderr, code := runMCPTest(t, input)
	if code != exitOK || stderr != "" {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if len(responses) != 2 {
		t.Fatalf("responses = %#v", responses)
	}
	initialize := resultObject(t, responses[0])
	if initialize["protocolVersion"] != mcpLatestProtocolVersion {
		t.Fatalf("initialize result = %#v", initialize)
	}
	tools, ok := resultObject(t, responses[1])["tools"].([]any)
	if !ok || len(tools) < 7 {
		t.Fatalf("tools = %#v", resultObject(t, responses[1])["tools"])
	}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		if tool["inputSchema"] == nil || tool["outputSchema"] == nil || tool["annotations"] == nil {
			t.Fatalf("incomplete tool = %#v", tool)
		}
	}
}

func TestMCPStdioToolCallReusesRuntime(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"yeelight_home_explain","arguments":{"locale":"en-US","intent":"scene.update"}}}`,
	}, "\n")
	responses, stderr, code := runMCPTest(t, input)
	if code != exitOK || stderr != "" || len(responses) != 2 {
		t.Fatalf("code=%d stderr=%s responses=%#v", code, stderr, responses)
	}
	result := resultObject(t, responses[1])
	if result["isError"] != false {
		t.Fatalf("tool result = %#v", result)
	}
	structured := result["structuredContent"].(map[string]any)
	if structured["status"] != "success" || structured["traceId"] != "intent-explain-local" {
		t.Fatalf("structuredContent = %#v", structured)
	}
}

func TestMCPStdioToolValidationReturnsToolError(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"yeelight_home_control_light","arguments":{"action":"brightness","value":50}}}`,
	}, "\n")
	responses, _, code := runMCPTest(t, input)
	if code != exitOK || len(responses) != 2 {
		t.Fatalf("code=%d responses=%#v", code, responses)
	}
	result := resultObject(t, responses[1])
	if result["isError"] != true {
		t.Fatalf("tool result = %#v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), "targetId or targetName") {
		t.Fatalf("content = %#v", content)
	}
}

func TestMCPStdioRequiresCompleteInitializeHandshake(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"client","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":5,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"client","version":"1"}}}`,
	}, "\n")
	responses, _, code := runMCPTest(t, input)
	if code != exitOK || len(responses) != 5 {
		t.Fatalf("code=%d responses=%#v", code, responses)
	}
	wantCodes := []int{-32002, 0, -32002, 0, -32600}
	for index, wantCode := range wantCodes {
		errorObject, _ := responses[index]["error"].(map[string]any)
		if wantCode == 0 {
			if errorObject != nil {
				t.Fatalf("response %d = %#v", index, responses[index])
			}
			continue
		}
		if errorObject == nil || int(errorObject["code"].(float64)) != wantCode {
			t.Fatalf("response %d = %#v", index, responses[index])
		}
	}
}

func TestMCPToolSchemasRequireConcreteTargets(t *testing.T) {
	definitions := localMCPToolDefinitions()
	for _, name := range []string{"yeelight_home_get_state", "yeelight_home_control_light", "yeelight_home_run_scene"} {
		var schema map[string]any
		for _, raw := range definitions {
			definition := raw.(map[string]any)
			if definition["name"] == name {
				schema = definition["inputSchema"].(map[string]any)
				break
			}
		}
		if len(schema) == 0 || len(schema["anyOf"].([]any)) != 2 {
			t.Fatalf("%s schema = %#v", name, schema)
		}
	}
}

func TestMCPStdioRejectsObjectRequestID(t *testing.T) {
	responses, _, code := runMCPTest(t, `{"jsonrpc":"2.0","id":{"bad":true},"method":"ping"}`)
	if code != exitOK || len(responses) != 1 {
		t.Fatalf("code=%d responses=%#v", code, responses)
	}
	errorObject := responses[0]["error"].(map[string]any)
	if int(errorObject["code"].(float64)) != -32600 {
		t.Fatalf("response = %#v", responses[0])
	}
}

func TestMCPStdioProtocolErrorsAndPing(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":`,
		`[]`,
	}, "\n")
	responses, _, code := runMCPTest(t, input)
	if code != exitOK || len(responses) != 4 {
		t.Fatalf("code=%d responses=%#v", code, responses)
	}
	wantCodes := []int{-32002, 0, -32700, -32600}
	for index, wantCode := range wantCodes {
		errorObject, _ := responses[index]["error"].(map[string]any)
		if wantCode == 0 {
			if errorObject != nil || responses[index]["result"] == nil {
				t.Fatalf("ping response = %#v", responses[index])
			}
			continue
		}
		if int(errorObject["code"].(float64)) != wantCode {
			t.Fatalf("response %d = %#v", index, responses[index])
		}
	}
}

func TestMCPStdioUnknownRequestAndNotification(t *testing.T) {
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"future-version"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/custom"}`,
		`{"jsonrpc":"2.0","id":2,"method":"custom/method"}`,
	}, "\n")
	responses, _, code := runMCPTest(t, input)
	if code != exitOK || len(responses) != 2 {
		t.Fatalf("code=%d responses=%#v", code, responses)
	}
	if resultObject(t, responses[0])["protocolVersion"] != mcpLatestProtocolVersion {
		t.Fatalf("initialize = %#v", responses[0])
	}
	errorObject := responses[1]["error"].(map[string]any)
	if int(errorObject["code"].(float64)) != -32601 {
		t.Fatalf("unknown method = %#v", responses[1])
	}
}

func TestMCPCommandRequiresStdioAndValidLanguage(t *testing.T) {
	app := newTestApp(t)
	for _, args := range [][]string{{"mcp", "serve"}, {"mcp", "serve", "--stdio", "--lang", "fr"}} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := app.run(args, strings.NewReader(""), &stdout, &stderr); code != exitInvalidInput {
			t.Fatalf("args=%#v code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("MCP error wrote non-RPC stdout: %s", stdout.String())
		}
	}
}

func runMCPTest(t *testing.T, input string) ([]map[string]any, string, int) {
	t.Helper()
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"mcp", "serve", "--stdio", "--lang", "zh-CN"}, strings.NewReader(input), &stdout, &stderr)
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, stderr.String(), code
	}
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("stdout contains non-JSON-RPC line %q: %v", line, err)
		}
		if response["jsonrpc"] != "2.0" {
			t.Fatalf("invalid JSON-RPC response: %#v", response)
		}
		responses = append(responses, response)
	}
	return responses, stderr.String(), code
}

func resultObject(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", response)
	}
	return result
}
