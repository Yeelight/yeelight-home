package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestInvokeAutomationEnableDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeAutomationStatusSeedList(writer, request, "0")
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-status-secret", "client-auto-status-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-auto-enable-plan","locale":"zh-CN","utterance":"启用回家开灯自动化","intent":"automation.enable","parameters":{"houseId":"200171","automationId":"auto-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/automations/w/enable") || strings.Contains(call, "/automations/w/disable") {
			t.Fatalf("automation status dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "token-auto-status-secret") || strings.Contains(stderr.String(), "token-auto-status-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	previewContainer := result["preview"].(map[string]any)
	preview := previewContainer["payloadPreview"].(map[string]any)
	if preview["automationId"] != "auto-1" {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokeAutomationDisableRequiresKnownAutomation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeAutomationStatusSeedList(writer, request, "1")
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-status-secret", "client-auto-status-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-auto-disable-missing","locale":"zh-CN","utterance":"停用不存在的自动化","intent":"automation.disable","parameters":{"houseId":"200171","automationId":"auto-missing"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_automation_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeAutomationDisableExecutesDirectly(t *testing.T) {
	automationListCalls := 0
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/automations/w/disable/auto-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/automations/r/list":
			automationListCalls++
			status := "1"
			if automationListCalls > 1 {
				status = "0"
			}
			writeAutomationStatusSeedList(writer, request, status)
		default:
			writeAutomationStatusSeedList(writer, request, "1")
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-status-secret", "client-auto-status-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-auto-disable-execute","locale":"zh-CN","utterance":"停用回家开灯自动化","intent":"automation.disable","parameters":{"houseId":"200171","automationId":"auto-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	disableCalls := 0
	for _, call := range gotCalls {
		if call == "POST /apis/iot/v1/automations/w/disable/auto-1" {
			disableCalls++
		}
	}
	if disableCalls != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "automation-status-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["automationId"] != "auto-1" || result["status"] != "0" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeAutomationEnableUsesTopologyCacheForTargetResolution(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		http.NotFound(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-status-secret", "client-auto-status-1", "200171")
	if err := app.topologyCache.Save("default", "dev", "200171", api.EntityListResult{
		Region:  "dev",
		HouseID: "200171",
		Total:   1,
		Counts:  map[string]int{"automation": 1},
		Entities: []api.EntitySummary{{
			Type:    "automation",
			ID:      "auto-1",
			Name:    "回家开灯",
			HouseID: "200171",
			Status:  "0",
		}},
	}, time.Unix(1000, 0)); err != nil {
		t.Fatalf("Save cache error: %v", err)
	}
	if cached, ok := app.topologyCache.Load("default", "dev", "200171", time.Now()); !ok || topologyCacheHits(cached) != 1 {
		t.Fatalf("cache preload failed: ok=%v cached=%#v", ok, cached)
	}

	input := `{"contractVersion":"1.0","requestId":"req-auto-enable-cache","locale":"zh-CN","utterance":"启用回家开灯自动化","intent":"automation.enable","parameters":{"houseId":"200171","automationName":"回家开灯"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	metrics := response["metrics"].(map[string]any)
	if metrics[semantic.FieldCacheHits] != float64(1) {
		t.Fatalf("metrics=%#v response=%#v", metrics, response)
	}
	if len(gotCalls) != 0 {
		t.Fatalf("dry-run target resolution should use cached topology without cloud calls, gotCalls=%#v", gotCalls)
	}
}

func TestEntityGetTargetDoesNotTreatHouseIDAsAutomationID(t *testing.T) {
	request := contract.Request{
		Intent: "automation.enable",
		Parameters: map[string]any{
			"houseId":        "200171",
			"automationName": "回家开灯",
		},
	}
	target := entityGetTargetFromRequest(request)
	if target.entityType != "automation" || target.id != "" || target.name != "回家开灯" {
		t.Fatalf("target = %#v", target)
	}
}

func TestInvokeAutomationEnableRefreshesTopologyWhenCachedTargetMissing(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/automations/w/enable/auto-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			writeAutomationStatusSeedList(writer, request, "1")
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-status-secret", "client-auto-status-1", "200171")
	if err := app.topologyCache.Save("default", "dev", "200171", api.EntityListResult{
		Region:  "dev",
		HouseID: "200171",
		Total:   1,
		Counts:  map[string]int{"automation": 1},
		Entities: []api.EntitySummary{{
			Type:    "automation",
			ID:      "auto-old",
			Name:    "旧自动化",
			HouseID: "200171",
		}},
	}, time.Unix(1000, 0)); err != nil {
		t.Fatalf("Save cache error: %v", err)
	}

	input := `{"contractVersion":"1.0","requestId":"req-auto-enable-refresh","locale":"zh-CN","utterance":"启用回家开灯自动化","intent":"automation.enable","parameters":{"houseId":"200171","automationName":"回家开灯"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	topologyCalls := 0
	for _, call := range gotCalls {
		if strings.Contains(call, "/thing/manage/house/200171/") || strings.Contains(call, "/automations/r/list") {
			topologyCalls++
		}
	}
	if topologyCalls < 6 {
		t.Fatalf("expected topology refresh after cache miss, gotCalls=%#v", gotCalls)
	}
}

func writeAutomationStatusSeedList(writer http.ResponseWriter, request *http.Request, automationStatus string) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"回家开灯","houseId":"200171","status":` + automationStatus + `}]}}`))
	default:
		http.NotFound(writer, request)
	}
}
