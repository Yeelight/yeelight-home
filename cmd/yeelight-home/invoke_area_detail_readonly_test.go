package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeAreaDetailGetUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/area/area-1/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"area-1","name":"一楼","desc":"公共活动区","rooms":[{"id":"room-1","name":"客厅","desc":"会客","deviceNum":2,"gatewayIds":["gw-1"],"rank":1}],"accessToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-area-detail-secret", "client-area-detail-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-area-detail","locale":"zh-CN","utterance":"查看一楼区域详情","intent":"area.detail.get","parameters":{"houseId":"house-1","areaId":"area-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 1 || gotCalls[0] != "GET /apis/iot/v2/thing/manage/house/house-1/area/area-1/r/info" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, forbidden := range []string{"token-area-detail-secret", "not-allowed", "\"desc\"", "\"deviceNum\""} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "area-detail-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	if result["cloudWrites"] != false || data["detail"] == nil {
		t.Fatalf("result = %#v", result)
	}
	detail := data["detail"].(map[string]any)
	if detail["description"] != "公共活动区" {
		t.Fatalf("detail = %#v", detail)
	}
	rooms := detail["rooms"].([]any)
	room := rooms[0].(map[string]any)
	if room["description"] != "会客" || room["deviceCount"] == nil || room["gatewayIds"] == nil {
		t.Fatalf("room = %#v", room)
	}
}

func TestInvokeAreaDetailGetRequiresAreaContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-area-detail-secret", "client-area-detail-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-area-detail-missing","locale":"zh-CN","utterance":"查看区域详情","intent":"area.detail.get","parameters":{"houseId":"house-1"}}`
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
	if len(unknowns) != 1 || unknowns[0] != "area_context_missing" {
		t.Fatalf("unknownEvidence = %#v", unknowns)
	}
}
