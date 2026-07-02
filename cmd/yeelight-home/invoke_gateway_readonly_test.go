package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeGatewayThreadGetUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/gateway/gateway-1/r/thread-info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"networkName":"yeelight-thread","panId":"abcd","localToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-secret", "client-gateway-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-thread","locale":"zh-CN","utterance":"查看网关 Thread 信息","intent":"gateway.thread.get","targets":[{"entityType":"gateway","id":"gateway-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v2/thing/manage/house/house-1/gateway/gateway-1/r/thread-info" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-gateway-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "gateway-thread-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false || result["gatewayId"] != "gateway-1" || result["deviceId"] != nil {
		t.Fatalf("result = %#v", result)
	}
	data := result["data"].(map[string]any)
	if data["threadInfo"] == nil {
		t.Fatalf("data = %#v", data)
	}
}

func TestInvokeGatewayListRedactsLocalKeys(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/gateway/r/info/1/100" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gateway-1","name":"E1 网关","mac":"AA:BB:CC:DD","online":true,"localKey":"not-allowed","bindKey":"not-allowed","psk":"not-allowed","ltk":"not-allowed","mibk":"not-allowed","midk":"not-allowed","hrbk":"not-allowed","meibk":"not-allowed","configs":[{"propId":"ltk","value":"not-allowed"},{"propId":"wifiPassword","value":"not-allowed"}]}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-secret", "client-gateway-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-list","locale":"zh-CN","utterance":"列出家庭网关","intent":"gateway.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v2/thing/manage/house/house-1/gateway/r/info/1/100" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	output := stdout.String() + stderr.String()
	for _, forbidden := range []string{"token-gateway-secret", "not-allowed", "AA:BB:CC:DD", "localKey", "bindKey", "psk", "ltk", "mibk", "midk", "hrbk", "meibk", "wifiPassword", "configs"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	if !strings.Contains(stdout.String(), "configCount") {
		t.Fatalf("gateway list should keep only configCount summary: %s", stdout.String())
	}
}

func TestInvokeGatewayDetailResolvesNaturalGatewayNameFromDeviceProjection(t *testing.T) {
	var detailCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/gateway/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gateway-1","name":"DALI 网关","mac":"AA:BB:CC:DD"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/gateway/gateway-1/r/info":
			detailCalled = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"DALI 网关","mac":"AA:BB:CC:DD","localToken":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-name-secret", "client-gateway-name-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-detail-by-name","locale":"zh-CN","utterance":"查看 DALI 网关详情","intent":"gateway.detail.get","parameters":{"houseId":"house-1","gatewayName":"DALI网关"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !detailCalled {
		t.Fatalf("gateway detail endpoint was not called")
	}
	output := stdout.String() + stderr.String()
	for _, forbidden := range []string{"token-gateway-name-secret", "not-allowed", "AA:BB:CC:DD"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "gateway-detail-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeGatewayDetailGetRequiresGatewayContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-secret", "client-gateway-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-detail-missing","locale":"zh-CN","utterance":"查看网关详情","intent":"gateway.detail.get","parameters":{"houseId":"house-1"}}`
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
	if response["status"] != "partial" || response["traceId"] != "gateway.detail.get-partial" {
		t.Fatalf("response = %#v", response)
	}
	warnings := response["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "gateway_context_missing" {
		t.Fatalf("warnings = %#v", warnings)
	}
}
