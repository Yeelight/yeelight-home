package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeHomeLockAllDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-lock-secret", "client-home-lock-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-home-lock-plan","locale":"zh-CN","utterance":"锁定这个家里所有设备的重置能力","intent":"home.lock_all","parameters":{"houseId":"200171"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/lockall") {
			t.Fatalf("home.lock_all dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if preview["affectedScope"] != "whole_house" || preview["deviceCount"] != float64(2) {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokeHomeUnlockAllExecutesDirectly(t *testing.T) {
	unlockCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/200171/unlockall":
			unlockCalls++
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-unlock-secret", "client-home-lock-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-home-unlock-execute","locale":"zh-CN","utterance":"解锁这个家里所有设备的重置能力","intent":"home.unlock_all","parameters":{"houseId":"200171"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if unlockCalls != 1 {
		t.Fatalf("unlockCalls = %d", unlockCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-lock-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.unlock_all" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}
