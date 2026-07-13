package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestModuleCommandDeviceListUsesRuntimeIntent(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/device/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","accessToken":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer module-device-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"device", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "module-device-secret") || strings.Contains(stderr.String(), "module-device-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/device/r/all" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["requestId"] == "" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "device.list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestModuleCommandSceneExecuteMapsFlagsToTarget(t *testing.T) {
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
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer module-scene-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"scene", "execute", "--scene-id", "scene-1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "scene-execute-command" {
		t.Fatalf("response = %#v", response)
	}
	if len(gotCalls) != 7 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestModuleCommandLightBrightnessAdjustSupportsRoomScope(t *testing.T) {
	var gotCalls []string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/controll/device/1/room-1/w/properties/l/adjust" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer module-light-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"light", "brightness-adjust", "--room-id", "room-1", "--delta", "-10", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "module-light-secret") || strings.Contains(stderr.String(), "module-light-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 1 || gotBody["value"] != float64(-10) {
		t.Fatalf("gotCalls = %#v body = %#v", gotCalls, gotBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "light-brightness-adjust-command" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["nodeType"] != "room" || result["nodeId"] != "room-1" || result["property"] != "brightness" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBuildModuleRequestLightDeviceFlagCreatesDeviceTarget(t *testing.T) {
	flags, err := parseFlags([]string{"--device-id", "992001", "--brightness", "50"})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	request, err := buildModuleRequest("light", "brightness", moduleCommands["light"]["brightness"], flags)
	if err != nil {
		t.Fatalf("buildModuleRequest error: %v", err)
	}
	target := entityGetTargetFromRequest(request)
	if target.entityType != "device" || target.id != "992001" {
		t.Fatalf("target = %#v", target)
	}
	if len(request.Targets) != 1 || request.Targets[0]["entityType"] != "device" || request.Targets[0]["id"] != "992001" {
		t.Fatalf("request targets = %#v", request.Targets)
	}
}

func TestModuleCommandReturnsAuthRequiredWithoutToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := newTestApp(t).run([]string{"light", "on", "--device-id", "device-1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "auth_required" {
		t.Fatalf("response = %#v", response)
	}
}
