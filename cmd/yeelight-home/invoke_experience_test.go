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

func TestInvokeSceneTestDelegatesToSceneExecute(t *testing.T) {
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
	app := newInvokeTestApp(t, "Bearer token-scene-test-secret", "client-scene-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-test","locale":"zh-CN","utterance":"测试晚安情景","intent":"scene.test","targets":[{"entityType":"scene","id":"scene-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-scene-test-secret") || strings.Contains(stderr.String(), "token-scene-test-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 7 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "scene-test-command" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["testOnly"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeSceneTestNoGatewayReturnsBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"测试全关"}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-no-gateway-secret", "client-scene-no-gateway-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-test-no-gateway","locale":"zh-CN","utterance":"测试全关情景","intent":"scene.test","targets":[{"entityType":"scene","id":"scene-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-scene-no-gateway-secret") || strings.Contains(stderr.String(), "token-scene-no-gateway-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "blocked" || response["traceId"] != "scene-test-blocked" {
		t.Fatalf("response = %#v", response)
	}
	errPayload := response["error"].(map[string]any)
	if errPayload["code"] != "scene_no_valid_gateway" {
		t.Fatalf("error = %#v", errPayload)
	}
}

func TestInvokeLightingExperienceApplyUsesReviewedLightWrapper(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","online":true}]}}`))
		case "/apis/iot/v1/controll/device/2/device-1/w/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":20}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-experience-secret", "client-exp-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-exp-apply","locale":"zh-CN","utterance":"给主灯应用观影体验","intent":"lighting.experience.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"brightness":20}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-experience-secret") || strings.Contains(stderr.String(), "token-experience-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 8 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "lighting-experience-apply-command" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	experience := result["experience"].(map[string]any)
	if experience["temporaryControl"] != true || experience["persistentWrites"] != false || experience["delegatedIntent"] != "light.brightness.set" {
		t.Fatalf("experience = %#v", experience)
	}
	if _, ok := experience["recipe"]; ok {
		t.Fatalf("Runtime must not attach subjective recipe: %#v", experience)
	}
}

func TestInvokeLightingExperienceApplyRequiresExplicitAction(t *testing.T) {
	t.Setenv("YEELIGHT_API_BASE_URL", "http://127.0.0.1:1/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-experience-secret", "client-exp-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-exp-mood-only","locale":"zh-CN","utterance":"给主灯应用观影体验","intent":"lighting.experience.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"mood":"观影"}}`
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
	if response["status"] != "blocked" || response["traceId"] != "lighting-experience-blocked" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["blockReason"] != "explicit_experience_action_required" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeLightingExperienceApplySupportsRoomNodeTarget(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/1/room-1/w/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-experience-secret", "client-exp-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-exp-room","locale":"zh-CN","utterance":"给客厅应用观影体验，亮度20","intent":"lighting.experience.apply","targets":[{"entityType":"room","id":"room-1"}],"parameters":{"brightness":20}}`
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
	if response["status"] != "success" || response["traceId"] != "lighting-experience-apply-command" {
		t.Fatalf("response = %#v", response)
	}
	if len(gotCalls) != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	result := response["result"].(map[string]any)
	if result[semantic.FieldNodeType] != "room" || result[semantic.FieldNodeID] != "room-1" {
		t.Fatalf("result = %#v", result)
	}
}
