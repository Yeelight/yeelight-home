package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeSensorListUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/r/sensors" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"sensor-1","name":"门磁","localToken":"not-allowed"}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sensor-secret", "client-sensor-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-sensor-list","locale":"zh-CN","utterance":"列出家里的传感器","intent":"sensor.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/iot/v1/device/r/sensors" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if strings.Contains(stdout.String(), "not-allowed") || strings.Contains(stdout.String(), "token-sensor-secret") {
		t.Fatalf("output leaked secret: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "sensor-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeDeviceEnergySummaryUsesTargetDevice(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/energy/devices/device-1/r/summary" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"daySum":1.2,"monthSum":8.5}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-energy-secret", "client-energy-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-energy","locale":"zh-CN","utterance":"查看主灯耗电","intent":"device.energy.summary","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v1/energy/devices/device-1/r/summary" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "device-energy-summary-readonly" {
		t.Fatalf("response = %#v", response)
	}
}
