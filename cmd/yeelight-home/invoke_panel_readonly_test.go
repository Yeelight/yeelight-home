package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokePanelListUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/panel/r/list/house-1" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"panel-1","name":"面板","localToken":"not-allowed"}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-secret", "client-panel-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-panel-list","locale":"zh-CN","utterance":"列出家里的面板","intent":"panel.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v1/panel/r/list/house-1" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if strings.Contains(stdout.String(), "not-allowed") || strings.Contains(stdout.String(), "token-panel-secret") {
		t.Fatalf("output leaked secret: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "panel-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeScreenControlListUsesSingleScreenPathWhenTargetProvided(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/ai/house-1/screen-1/control/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"resId":"device-1","resType":2}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-screen-secret", "client-screen-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-screen-control","locale":"zh-CN","utterance":"查看厨房屏控制哪些设备","intent":"screen.control.list","targets":[{"entityType":"device","id":"screen-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/iot/v1/ai/house-1/screen-1/control/r/info" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "screen-control-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false || result["deviceId"] != "screen-1" {
		t.Fatalf("result = %#v", result)
	}
}
