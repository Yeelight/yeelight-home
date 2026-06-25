package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeBehaviorExecuteDelegatesSafeColorControl(t *testing.T) {
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/c":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/c":
			_, _ = writer.Write([]byte(`{"success":true,"data":255}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-behavior-secret", "client-behavior-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-behavior-color","locale":"zh-CN","utterance":"把主灯设置成蓝色","intent":"behavior.execute","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"controlRequest":{"command":{"command":"set","params":[{"propName":"c","value":255}]}}}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-behavior-secret") || strings.Contains(stderr.String(), "token-behavior-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != float64(255) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-color-set-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["verified"] != true || result["propertyName"] != "c" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeBehaviorExecuteRejectsUnsupportedPropertyWithoutAPI(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-behavior-secret", "client-behavior-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-behavior-unsupported","locale":"zh-CN","utterance":"控制主灯未知属性","intent":"behavior.execute","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"propName":"tp","value":50}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "not_supported" || response["traceId"] != "behavior-execute-unsupported" {
		t.Fatalf("response = %#v", response)
	}
	errorResult, ok := response["error"].(map[string]any)
	if !ok || errorResult["code"] != "unsupported_behavior_execute" {
		t.Fatalf("error = %#v", response["error"])
	}
	metrics, ok := response["metrics"].(map[string]any)
	if !ok || metrics["apiCalls"] != float64(0) {
		t.Fatalf("metrics = %#v", response["metrics"])
	}
}
