package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeEntityCapabilitiesReturnsConservativeSummary(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
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
	app := newInvokeTestApp(t, "Bearer token-capabilities-secret", "client-capabilities-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-capabilities-1","locale":"zh-CN","utterance":"客厅能做什么","intent":"entity.capabilities","targets":[{"entityType":"room","id":"room-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-capabilities-secret") || strings.Contains(stderr.String(), "token-capabilities-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 6 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "entity-capabilities-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["capabilitySource"] != "entity.list_projection" || result["schemaStatus"] != "not_connected" {
		t.Fatalf("result = %#v", result)
	}
	operations, ok := result["operations"].(map[string]any)
	if !ok {
		t.Fatalf("operations = %#v", result["operations"])
	}
	writeOps, ok := operations["write"].([]any)
	if !ok || len(writeOps) != 0 {
		t.Fatalf("operations = %#v", operations)
	}
	limitations, ok := result["limitations"].([]any)
	if !ok || len(limitations) == 0 {
		t.Fatalf("limitations = %#v", result["limitations"])
	}
}

func TestInvokeEntityCapabilitiesUsesDeviceSchemaWhenTargetIsDevice(t *testing.T) {
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
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","pid":17000008,"cid":1001,"category":"light","roomId":"room-1","subDevices":[{"cid":2001,"index":1,"name":"主灯组件","properties":[{"propId":"brightness","desc":"亮度","access":6,"format":"uint8","type":1,"valueRange":{"min":1,"max":100,"step":1}}],"supportActions":[{"actionName":"set_brightness"}],"events":[{"eventId":10,"eventTypeId":1,"name":"状态变化"}]}],"supportActions":[{"actionName":"set_power"}]}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-capabilities-secret", "client-capabilities-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-capabilities-device","locale":"zh-CN","utterance":"主灯能做什么","intent":"entity.capabilities","targets":[{"entityType":"device","id":"device-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-capabilities-secret") || strings.Contains(stderr.String(), "token-capabilities-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 7 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "entity-capabilities-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["capabilitySource"] != "device_schema_endpoint" || result["schemaStatus"] != "connected" {
		t.Fatalf("result = %#v", result)
	}
	operations, ok := result["operations"].(map[string]any)
	if !ok {
		t.Fatalf("operations = %#v", result["operations"])
	}
	writeOps, ok := operations["write"].([]any)
	if !ok || len(writeOps) != 0 {
		t.Fatalf("operations = %#v", operations)
	}
	schema, ok := result["deviceSchema"].(map[string]any)
	if !ok || schema["pid"] != "17000008" || schema["category"] != "light" {
		t.Fatalf("deviceSchema = %#v", result["deviceSchema"])
	}
	components, ok := schema["components"].([]any)
	if !ok || len(components) != 1 {
		t.Fatalf("components = %#v", schema["components"])
	}
	if strings.Contains(stdout.String(), "rawSecret") {
		t.Fatalf("raw schema leaked: %s", stdout.String())
	}
}

func TestInvokeEntityCapabilitiesRequiresTargetWhenMissing(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-capabilities-secret", "client-capabilities-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-capabilities-missing","locale":"zh-CN","utterance":"它能做什么","intent":"entity.capabilities"}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "entity-capabilities-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "missing_target" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}
