package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeSceneUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSceneUpdateSeedList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-update-secret", "client-scene-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-scene-update-plan","locale":"zh-CN","utterance":"把回家灯光情景改为打开主灯","intent":"scene.update","parameters":{"houseId":"200171","sceneId":"scene-1","name":"回家灯光更新","details":[{"typeId":2,"resId":"50018330","params":{"set":{"p":true}}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/scene/scene-1/w/modify") {
			t.Fatalf("scene.update dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "token-scene-update-secret") || strings.Contains(stderr.String(), "token-scene-update-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if preview["sceneId"] != "scene-1" || preview["name"] != "回家灯光更新" {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokeSceneUpdateRequiresKnownScene(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSceneUpdateSeedList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-update-secret", "client-scene-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-scene-update-missing","locale":"zh-CN","utterance":"更新不存在的情景","intent":"scene.update","parameters":{"houseId":"200171","sceneId":"scene-missing","name":"不存在","details":[{"typeId":2,"resId":"50018330","rank":0,"params":{"set":{"p":true}}}]}}`
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
	if clarification["reason"] != "invalid_scene_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
	if clarification["payloadShape"] == nil || clarification["examples"] == nil || !strings.Contains(requestString(clarification["nextStep"]), "scene.detail.get") {
		t.Fatalf("clarification missing payload guide = %#v", clarification)
	}
}

func TestInvokeSceneUpdateInvalidPayloadReturnsPayloadGuide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSceneUpdateSeedList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-update-secret", "client-scene-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-scene-update-bad-shape","locale":"zh-CN","utterance":"把孩子屋开灯改暖一点","intent":"scene.update","parameters":{"houseId":"200171","sceneId":"scene-1","name":"孩子屋开灯","params":{"set":{"ct":3000}}}}`
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
	if clarification["reason"] != "invalid_scene_update_payload" || clarification["payloadShape"] == nil || clarification["examples"] == nil {
		t.Fatalf("clarification = %#v", clarification)
	}
	if !strings.Contains(requestString(clarification["nextStep"]), "complete updated list") {
		t.Fatalf("clarification nextStep = %#v", clarification["nextStep"])
	}
}

func TestInvokeSceneUpdateExecutesDirectly(t *testing.T) {
	detailCalls := 0
	var gotCalls []string
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode scene update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			detailCalls++
			name := "回家灯光"
			if detailCalls > 1 {
				name = "回家灯光更新"
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","name":"` + name + `","details":[{"typeId":2,"resId":50018330,"rank":0,"action":0,"params":"{\"set\":{\"p\":true}}"}]}}`))
		default:
			writeSceneUpdateSeedList(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-update-secret", "client-scene-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-scene-update-execute","locale":"zh-CN","utterance":"把回家灯光情景改为打开主灯","intent":"scene.update","parameters":{"houseId":"200171","sceneId":"scene-1","name":"回家灯光更新","details":[{"typeId":2,"resId":"50018330","params":{"set":{"p":true}}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	updateCalls := 0
	for _, call := range gotCalls {
		if call == "POST /apis/iot/v2/thing/manage/house/200171/scene/scene-1/w/modify" {
			updateCalls++
		}
		if strings.Contains(call, "ignored") {
			t.Fatalf("commit request payload leaked into API call: %#v", gotCalls)
		}
	}
	if updateCalls != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if writeBody["id"] != "scene-1" || writeBody["name"] != "回家灯光更新" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	details, ok := writeBody["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("writeBody details = %#v", writeBody["details"])
	}
	detail, ok := details[0].(map[string]any)
	if !ok || detail["resName"] != "主灯" || detail["action"] != float64(0) || detail["rank"] != float64(0) || detail["params"] != `{"set":{"p":true}}` {
		t.Fatalf("writeBody detail = %#v", details[0])
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "scene-update-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["sceneId"] != "scene-1" || result["name"] != "回家灯光更新" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func writeSceneUpdateSeedList(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","houseId":"200171"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"回家灯光","houseId":"200171"}]}}`))
	default:
		http.NotFound(writer, request)
	}
}
