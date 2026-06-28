package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceRemoveExecutesDirectlyAfterCallerConfirmation(t *testing.T) {
	deviceVisible := true
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			if deviceVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401398"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/50018330/w/info":
			deviceVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-r3-secret", "client-r3-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-remove-plan","locale":"zh-CN","utterance":"删除主灯","intent":"device.remove","parameters":{"houseId":"200171","deviceId":"50018330"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "destructive-delete-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "device.remove" || result["risk"] != "R3" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
	deleteCalls := 0
	for _, call := range gotCalls {
		if strings.Contains(call, "DELETE /apis/iot/v2/thing/manage/house/200171/device/50018330/w/info") {
			deleteCalls++
		}
	}
	if deleteCalls != 1 {
		t.Fatalf("delete calls = %#v", gotCalls)
	}
}

func TestInvokeHomeDeleteUsesHouseScopedFallbackForNewlyCreatedHome(t *testing.T) {
	homeVisible := true
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200181/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200181/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200181/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200181/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200181/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			if homeVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			http.NotFound(writer, request)
		case "/apis/iot/v1/house/200181/w/delete":
			homeVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-delete-secret", "client-home-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-home-delete-plan","locale":"zh-CN","utterance":"删除临时家庭","intent":"home.delete","parameters":{"houseId":"200181","name":"临时家庭"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "destructive-delete-execute" {
		t.Fatalf("response = %#v, calls=%#v", response, gotCalls)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "home.delete" || result["risk"] != "R3" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeGatewayDeleteInvalidGatewayReturnsClarification(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/gateway/not-a-gateway/r/info":
			_, _ = writer.Write([]byte(`{"success":false,"code":600,"message":"参数格式错误"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-r3-secret", "client-r3-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-delete-invalid","locale":"zh-CN","utterance":"删除不存在的网关","intent":"gateway.delete","parameters":{"houseId":"200171","gatewayId":"not-a-gateway"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") {
			t.Fatalf("gateway.delete should not write while target is invalid: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "entity_not_found" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeGatewayDeleteServerBusinessErrorStillFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-r3-secret", "client-r3-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-delete-server-error","locale":"zh-CN","utterance":"删除异常网关","intent":"gateway.delete","parameters":{"houseId":"200171","gatewayId":"gateway-500"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code == exitOK {
		t.Fatalf("expected server business error to fail, stdout = %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "gateway.detail.get returned non-success business response") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
