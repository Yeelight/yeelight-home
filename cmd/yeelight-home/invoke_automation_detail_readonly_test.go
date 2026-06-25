package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeAutomationDetailGetUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/automation/auto-1/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"auto-1","name":"回家开灯","params":{"type":"and"},"actions":[{"resId":"device-1"}],"accessToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-detail-secret", "client-auto-detail-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-auto-detail","locale":"zh-CN","utterance":"查看回家开灯自动化详情","intent":"automation.detail.get","parameters":{"houseId":"house-1","automationId":"auto-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 1 || gotCalls[0] != "GET /apis/iot/v2/thing/manage/house/house-1/automation/auto-1/r/info" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, forbidden := range []string{"token-auto-detail-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "automation-detail-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	if result["cloudWrites"] != false || data["detail"] == nil {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeAutomationDetailGetRequiresAutomationContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-detail-secret", "client-auto-detail-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-auto-detail-missing","locale":"zh-CN","utterance":"查看自动化详情","intent":"automation.detail.get","parameters":{"houseId":"house-1"}}`
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
	if response["status"] != "partial" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	unknowns := result["unknownEvidence"].([]any)
	if len(unknowns) != 1 || unknowns[0] != "automation_context_missing" {
		t.Fatalf("unknownEvidence = %#v", unknowns)
	}
}
