package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeLightBrightnessAdjustReadsAdjustsAndVerifiesDeviceProperty(t *testing.T) {
	var gotCalls []string
	var adjustBody map[string]any
	stateReadCount := 0
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯"}]}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			stateReadCount++
			if stateReadCount == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":41}`))
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/l/adjust":
			if err := json.NewDecoder(request.Body).Decode(&adjustBody); err != nil {
				t.Fatalf("decode adjust body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-brightness-adjust","locale":"zh-CN","utterance":"把主灯亮度暗一点","intent":"light.brightness.adjust","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"delta":-1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-light-secret") || strings.Contains(stderr.String(), "token-light-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if adjustBody["value"] != float64(-1) {
		t.Fatalf("adjustBody = %#v", adjustBody)
	}
	if len(gotCalls) != 9 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-brightness-adjust-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["beforeValue"] != float64(42) || result["delta"] != float64(-1) || result["expectedValue"] != float64(41) || result["verified"] != true || result["verifiedValue"] != float64(41) || result["property"] != "brightness" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightColorTemperatureAdjustReadsAdjustsAndVerifiesDeviceProperty(t *testing.T) {
	var adjustBody map[string]any
	stateReadCount := 0
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
		case "/apis/iot/v1/controll/device/device-1/r/properties/ct":
			stateReadCount++
			if stateReadCount == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":2700}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":2710}`))
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/ct/adjust":
			if err := json.NewDecoder(request.Body).Decode(&adjustBody); err != nil {
				t.Fatalf("decode adjust body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-ct-adjust","locale":"zh-CN","utterance":"把主灯色温调高一点","intent":"light.color_temperature.adjust","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"delta":10}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if adjustBody["value"] != float64(10) {
		t.Fatalf("adjustBody = %#v", adjustBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-color-temperature-adjust-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["beforeValue"] != float64(2700) || result["expectedValue"] != float64(2710) || result["verified"] != true || result["property"] != "colorTemperature" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightBrightnessAdjustReportsVerificationMismatch(t *testing.T) {
	stateReadCount := 0
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
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			stateReadCount++
			if stateReadCount == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/l/adjust":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-brightness-adjust-mismatch","locale":"zh-CN","utterance":"把主灯亮度暗一点","intent":"light.brightness.adjust","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"delta":-1}}`
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
	if response["status"] != "partial" || response["traceId"] != "light-brightness-adjust-verification-mismatch" {
		t.Fatalf("response = %#v", response)
	}
	errorResult, ok := response["error"].(map[string]any)
	if !ok || errorResult["code"] != "write_verification_mismatch" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func TestInvokeLightBrightnessAdjustRequiresDelta(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-brightness-adjust-missing","locale":"zh-CN","utterance":"把主灯亮度调一下","intent":"light.brightness.adjust","targets":[{"entityType":"device","id":"device-1"}]}`
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
	if !ok || clarification["reason"] != "missing_brightness_delta" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}
