package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/p":
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

	input := `{"contractVersion":"1.0","requestId":"req-light-power","locale":"zh-CN","utterance":"关闭主灯","intent":"light.power.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"power":false}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-light-secret") || strings.Contains(stderr.String(), "token-light-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if _, ok := writeBody["command"]; ok || writeBody["value"] != false {
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
	if !ok || result["verified"] != true || result["verifiedValue"] != false || result["property"] != "power" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightPowerSetUsesTopologyCacheAfterWarmup(t *testing.T) {
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/p":
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

	input := `{"contractVersion":"1.0","requestId":"req-light-power-cache","locale":"zh-CN","utterance":"打开主灯","intent":"light.power.set","targets":[{"entityType":"device","name":"主灯"}],"parameters":{"power":true}}`
	for i := 0; i < 2; i++ {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("run %d exit code = %d, stderr = %s", i, code, stderr.String())
		}
		if i == 1 {
			var response map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("invalid json response: %v", err)
			}
			metrics := response["metrics"].(map[string]any)
			if metrics[semantic.FieldCacheHits] != float64(1) || metrics[semantic.FieldAPICalls] != float64(2) {
				t.Fatalf("metrics=%#v response=%#v", metrics, response)
			}
		}
	}
	listCalls := 0
	for _, call := range gotCalls {
		if strings.Contains(call, "/thing/manage/house/house-1/") || strings.Contains(call, "/automations/r/list") {
			listCalls++
		}
	}
	if listCalls != 6 {
		t.Fatalf("second run should not repeat topology list calls, gotCalls=%#v", gotCalls)
	}
}

func TestInvokeLightColorTemperatureDryRunDoesNotWrite(t *testing.T) {
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/ct",
			"/apis/iot/v1/controll/device/device-1/r/properties/ct":
			t.Fatalf("dry-run must not call write or verification API: %s", request.URL.Path)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-light-ct-dry-run","locale":"zh-CN","utterance":"把主灯调暖一点","intent":"light.color_temperature.set","targets":[{"entityType":"device","name":"主灯"}],"parameters":{"colorTemperature":3000}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "direct-write-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["dryRun"] != true || result["persistentWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
	warnings := response["warnings"].([]any)
	if containsAnyString(warnings, "dry_run_no_cloud_write_not_available_for_direct_execution") {
		t.Fatalf("warnings = %#v", warnings)
	}
	planned := result["planned"].(map[string]any)
	if planned["property"] != "colorTemperature" || planned["value"] != float64(3000) {
		t.Fatalf("planned = %#v", planned)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/properties/ct") || strings.Contains(call, "/r/properties/ct") {
			t.Fatalf("dry-run should not call control/state endpoints, gotCalls=%#v", gotCalls)
		}
	}
}

func TestInvokeLightPowerSetResolvesDeviceWithinNamedRoom(t *testing.T) {
	var writePath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-kid","name":"孩子屋"},{"id":"room-living","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-kid-ceiling","name":"吸顶灯","roomId":"room-kid","online":true},{"id":"device-living-ceiling","name":"吸顶灯","roomId":"room-living","online":true}]}}`))
		case "/apis/iot/v1/controll/device/2/device-kid-ceiling/w/properties/p":
			writePath = request.URL.Path
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-kid-ceiling/r/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-room-secret", "client-light-room-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-light-room-name","locale":"zh-CN","utterance":"打开孩子屋的吸顶灯","intent":"light.power.set","parameters":{"roomName":"孩子屋","deviceName":"吸顶灯","power":true}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(writePath, "device-kid-ceiling") {
		t.Fatalf("writePath = %s", writePath)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	entity := result["entity"].(map[string]any)
	if entity["id"] != "device-kid-ceiling" || entity["roomId"] != "room-kid" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestInvokeLightPowerSetUsesRoomTargetAsQualifier(t *testing.T) {
	var writePath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-kid","name":"孩子屋"},{"id":"room-living","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-kid-ceiling","name":"吸顶灯","roomId":"room-kid","online":true},{"id":"device-living-ceiling","name":"吸顶灯","roomId":"room-living","online":true}]}}`))
		case "/apis/iot/v1/controll/device/2/device-kid-ceiling/w/properties/p":
			writePath = request.URL.Path
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-kid-ceiling/r/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-target-secret", "client-light-target-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-light-two-targets","locale":"zh-CN","utterance":"打开孩子屋的吸顶灯","intent":"light.power.set","targets":[{"entityType":"room","name":"孩子屋"},{"entityType":"device","name":"吸顶灯"}],"parameters":{"power":true}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(writePath, "device-kid-ceiling") {
		t.Fatalf("writePath = %s", writePath)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/p":
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

	input := `{"contractVersion":"1.0","requestId":"req-light-power-mismatch","locale":"zh-CN","utterance":"关闭主灯","intent":"light.power.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"power":false}}`
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/l":
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
	if _, ok := writeBody["command"]; ok || writeBody["value"] != float64(42) {
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
	if !ok || result["verified"] != true || result["verifiedValue"] != float64(42) || result["property"] != "brightness" {
		t.Fatalf("result = %#v", response["result"])
	}
}

func TestInvokeLightBrightnessSetPollsUntilVerificationMatches(t *testing.T) {
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			stateReadCount++
			if stateReadCount == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":41}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":42}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-brightness-set-delayed","locale":"zh-CN","utterance":"把主灯亮度设为 42","intent":"light.brightness.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"brightness":42}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stateReadCount != 2 {
		t.Fatalf("stateReadCount = %d", stateReadCount)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-brightness-set-command" {
		t.Fatalf("response = %#v", response)
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/ct":
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
	if _, ok := writeBody["command"]; ok || writeBody["value"] != float64(4000) {
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
	if !ok || result["verified"] != true || result["verifiedValue"] != float64(4000) || result["property"] != "colorTemperature" {
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/c":
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
	if _, ok := writeBody["command"]; ok || writeBody["value"] != float64(16711680) {
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
	if !ok || result["verified"] != true || result["expectedValue"] != float64(16711680) || result["verifiedValue"] != float64(16711680) || result["property"] != "color" {
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
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/c":
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
	if _, ok := writeBody["command"]; ok || writeBody["value"] != float64(65280) {
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

func TestInvokeLightColorSetAcceptsRGBObjectValue(t *testing.T) {
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"氛围灯"}]}}`))
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/c":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/c":
			_, _ = writer.Write([]byte(`{"success":true,"data":16744628}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-light-secret", "client-light-rgb-object-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-color-set-rgb-object","locale":"zh-CN","utterance":"把氛围灯设为粉色","intent":"light.color.set","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"color":{"red":255,"green":128,"blue":180}}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, ok := writeBody["command"]; ok || writeBody["value"] != float64(16744628) {
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
