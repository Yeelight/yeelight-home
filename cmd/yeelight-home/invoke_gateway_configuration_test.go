package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeGatewayConfigureExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	gatewayDetailReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/r/info":
			gatewayDetailReads++
			if gatewayDetailReads < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"旧网关"}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"新网关"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-config-secret", "client-gateway-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-configure","locale":"zh-CN","utterance":"更新网关名称","intent":"gateway.configure","parameters":{"houseId":"200171","gatewayId":"gateway-1","name":"新网关","roomIds":["room-1"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["name"] != "新网关" || writeBody["gatewayId"] != nil || writeBody["houseId"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "gateway-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeGatewayConfigureResolvesNaturalGatewayAndRoomNames(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"客厅网关"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gateway-1","name":"客厅网关"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-gateway-config-secret", "client-gateway-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-gateway-natural","locale":"zh-CN","utterance":"把客廷网关改名为玄关网关并关联客廷","intent":"gateway.configure","parameters":{"houseId":"200171","gatewayName":"客廷网关","name":"玄关网关","roomNames":["客廷"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/modify") {
			t.Fatalf("gateway.configure dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	payloadPreview := preview["payloadPreview"].(map[string]any)
	if payloadPreview["name"] != "玄关网关" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
}
