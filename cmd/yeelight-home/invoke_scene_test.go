package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeSceneExecuteRunsScene(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"晚安"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-secret", "client-scene-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-1","locale":"zh-CN","utterance":"执行晚安","intent":"scene.execute","targets":[{"entityType":"scene","id":"scene-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-scene-secret") || strings.Contains(stderr.String(), "token-scene-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 7 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "scene-execute-command" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["source"] != "open_control_scene_endpoint" {
		t.Fatalf("result = %#v", result)
	}
	entity, ok := result["entity"].(map[string]any)
	if !ok || entity["id"] != "scene-1" || entity["type"] != "scene" {
		t.Fatalf("entity = %#v", result["entity"])
	}
}

func TestInvokeSceneExecuteUsesTopologyCacheAfterWarmup(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"晚安"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-secret", "client-scene-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-cache","locale":"zh-CN","utterance":"执行晚安","intent":"scene.execute","targets":[{"entityType":"scene","name":"晚安"}]}`
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
			if metrics["cacheHits"] != float64(1) || metrics["apiCalls"] != float64(1) {
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

func TestInvokeSceneExecuteRequiresSceneTarget(t *testing.T) {
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
	app := newInvokeTestApp(t, "Bearer token-scene-secret", "client-scene-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-room","locale":"zh-CN","utterance":"执行客厅","intent":"scene.execute","targets":[{"entityType":"room","id":"room-1"}]}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "scene-execute-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "target_not_scene" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func TestInvokeSceneExecuteRequiresTargetBeforeListing(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-scene-secret", "client-scene-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-scene-missing","locale":"zh-CN","utterance":"执行一下","intent":"scene.execute"}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "scene-execute-clarification" {
		t.Fatalf("response = %#v", response)
	}
	metrics, ok := response["metrics"].(map[string]any)
	if !ok || metrics["apiCalls"] != float64(0) {
		t.Fatalf("metrics = %#v", response["metrics"])
	}
}
