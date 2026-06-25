package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeLightPowerSetWritesAndVerifiesDeviceProperty(t *testing.T) {
	var gotCalls []string
	var writeBody map[string]any
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
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":false}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-light-power","locale":"zh-CN","utterance":"关闭主灯","intent":"light.power.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"on":false}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-light-secret") || strings.Contains(stderr.String(), "token-light-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != false {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if len(gotCalls) != 8 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-power-set-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["verified"] != true || result["verifiedValue"] != false || result["propertyName"] != "p" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightPowerSetReportsVerificationMismatch(t *testing.T) {
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-light-power-mismatch","locale":"zh-CN","utterance":"关闭主灯","intent":"light.power.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"on":false}}`
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
	if response["status"] != "partial" || response["traceId"] != "light-power-set-verification-mismatch" {
		t.Fatalf("response = %#v", response)
	}
	errorResult, ok := response["error"].(map[string]any)
	if !ok || errorResult["code"] != "write_verification_mismatch" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func TestInvokeLightPowerSetRequiresDeviceAndPowerValue(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-light-missing-value","locale":"zh-CN","utterance":"控制主灯","intent":"light.power.set","targets":[{"entityType":"device","id":"device-1"}]}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "light-control-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "missing_power_value" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func TestInvokeLightBrightnessSetWritesAndVerifiesDeviceProperty(t *testing.T) {
	var writeBody map[string]any
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/l":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-brightness-set","locale":"zh-CN","utterance":"把主灯亮度设为 42","intent":"light.brightness.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"brightness":42}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != float64(42) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-brightness-set-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["verified"] != true || result["verifiedValue"] != float64(42) || result["propertyName"] != "l" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightColorTemperatureSetWritesAndVerifiesDeviceProperty(t *testing.T) {
	var writeBody map[string]any
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/ct":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/ct":
			_, _ = writer.Write([]byte(`{"success":true,"data":4000}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-ct-set","locale":"zh-CN","utterance":"把主灯色温设为 4000K","intent":"light.color_temperature.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"colorTemperature":4000}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != float64(4000) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-color-temperature-set-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["verified"] != true || result["verifiedValue"] != float64(4000) || result["propertyName"] != "ct" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightColorSetWritesAndVerifiesDeviceProperty(t *testing.T) {
	var writeBody map[string]any
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/c":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/c":
			_, _ = writer.Write([]byte(`{"success":true,"data":16711680}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-color-set","locale":"zh-CN","utterance":"把主灯设为红色","intent":"light.color.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"color":16711680}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != float64(16711680) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-color-set-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["verified"] != true || result["verifiedValue"] != float64(16711680) || result["propertyName"] != "c" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightColorSetAcceptsHexValue(t *testing.T) {
	var writeBody map[string]any
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/c":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/c":
			_, _ = writer.Write([]byte(`{"success":true,"data":65280}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-color-set-hex","locale":"zh-CN","utterance":"把主灯设为绿色","intent":"light.color.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"hex":"#00FF00"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["command"] != "set" || writeBody["value"] != float64(65280) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-color-set-command" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeLightColorSetRequiresColorValue(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-color-missing","locale":"zh-CN","utterance":"把主灯换个颜色","intent":"light.color.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"hex":"red"}}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "light-control-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "missing_color_value" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}
