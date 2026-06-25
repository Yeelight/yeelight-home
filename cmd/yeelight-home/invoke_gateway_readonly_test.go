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
	if result["cloudWrites"] != false || result["deviceId"] != "gateway-1" {
		t.Fatalf("result = %#v", result)
	}
	data := result["data"].(map[string]any)
	if data["threadInfo"] == nil {
		t.Fatalf("data = %#v", data)
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
