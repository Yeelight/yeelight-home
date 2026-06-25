package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeStateQueryReadsDeviceProperty(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/power":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-state-secret", "client-state-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-state-1","locale":"zh-CN","utterance":"主灯开着吗","intent":"state.query","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"propertyName":"power"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-state-secret") || strings.Contains(stderr.String(), "token-state-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 7 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "state-query-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["propertyName"] != "power" || result["value"] != true || result["queryScope"] != "single_property" {
		t.Fatalf("result = %#v", result)
	}
	entity, ok := result["entity"].(map[string]any)
	if !ok || entity["id"] != "device-1" || entity["type"] != "device" {
		t.Fatalf("entity = %#v", result["entity"])
	}
}

func TestInvokeStateQueryReadsDeviceProperties(t *testing.T) {
	var gotStateCalls []string
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","properties":[{"propId":"power"},{"propId":"brightness"}],"subDevices":[{"properties":[{"propId":"power"}]}]}]}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/power":
			gotStateCalls = append(gotStateCalls, request.URL.Path)
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/brightness":
			gotStateCalls = append(gotStateCalls, request.URL.Path)
			_, _ = writer.Write([]byte(`{"success":true,"data":72}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-state-secret", "client-state-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-state-all","locale":"zh-CN","utterance":"看看主灯状态","intent":"state.query","targets":[{"entityType":"device","id":"device-1"}]}`
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
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["queryScope"] != "all_properties" {
		t.Fatalf("result = %#v", result)
	}
	properties, ok := result["properties"].(map[string]any)
	if !ok || properties["power"] != true || properties["brightness"] != float64(72) {
		t.Fatalf("properties = %#v", result["properties"])
	}
	if len(gotStateCalls) != 2 {
		t.Fatalf("state calls = %#v", gotStateCalls)
	}
	metrics, ok := response["metrics"].(map[string]any)
	if !ok || metrics["apiCalls"] != float64(8) {
		t.Fatalf("metrics = %#v", response["metrics"])
	}
}

func TestInvokeStateQueryRequiresDeviceTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-state-secret", "client-state-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-state-room","locale":"zh-CN","utterance":"客厅现在什么状态","intent":"state.query","targets":[{"entityType":"room","id":"room-1"}]}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "state-query-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "target_not_device" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}
