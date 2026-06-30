package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeLightingDesignPlanBuildsLocalDesignEvidence(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true},{"id":"device-2","name":"灯带","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","properties":[{"propId":"power"},{"propId":"brightness"},{"propId":"colorTemperature"}]},{"id":"device-2","name":"灯带","properties":[{"propId":"power"},{"propId":"brightness"},{"propId":"color"}]}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-secret", "client-design-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-plan","locale":"zh-CN","utterance":"给客厅做一个观影灯光方案","intent":"lighting.design.plan","targets":[{"entityType":"room","id":"room-1"}],"parameters":{"mood":"观影"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-design-secret") || strings.Contains(stderr.String(), "token-design-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 8 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "lighting-design-plan-local" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["persistentWrites"] != false || result["applyBehavior"] != "caller_authored_actions_required" {
		t.Fatalf("result = %#v", result)
	}
	if _, ok := result["selectedRecipe"]; ok {
		t.Fatalf("Runtime must not select subjective lighting recipes: %#v", result)
	}
	evidence := result["deviceEvidence"].([]any)
	if len(evidence) != 2 {
		t.Fatalf("deviceEvidence = %#v", evidence)
	}
}

func TestInvokeLightingDesignPlanClarifiesUnknownTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-design-secret", "client-design-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-missing","locale":"zh-CN","utterance":"给书房做阅读方案","intent":"lighting.design.plan","targets":[{"entityType":"room","name":"书房"}]}`
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
	if response["status"] != "clarification_required" || response["traceId"] != "lighting-design-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "entity_not_found" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeLightingDesignApplyDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","properties":[{"propId":"p"},{"propId":"l"},{"propId":"ct"},{"propId":"c"}]}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-apply-secret", "client-design-apply-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-plan","locale":"zh-CN","utterance":"把客厅应用观影灯光设计","intent":"lighting.design.apply","targets":[{"entityType":"room","id":"room-1"}],"parameters":{"brightness":20,"colorTemperature":3000,"hex":"#3366ff"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/properties/") {
			t.Fatalf("lighting.design.apply dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	if preview["intent"] != "lighting.design.apply" {
		t.Fatalf("preview = %#v", preview)
	}
	payloadPreview := preview["payloadPreview"].(map[string]any)
	actions := payloadPreview["actions"].([]any)
	if len(actions) < 2 {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
}

func TestInvokeLightingDesignApplyPrefersExplicitValuesOverRecipe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","properties":[{"propId":"p"},{"propId":"l"},{"propId":"ct"}]}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-apply-secret", "client-design-apply-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-explicit","locale":"zh-CN","utterance":"恢复原来的关灯亮度100色温2700","intent":"lighting.design.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"mood":"阅读","power":false,"brightness":100,"colorTemperature":2700}}`
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
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	actions := preview["payloadPreview"].(map[string]any)["actions"].([]any)
	got := map[string]any{}
	for _, raw := range actions {
		action := raw.(map[string]any)
		got[action["propertyName"].(string)] = action["value"]
	}
	brightness, brightnessOK := requestInt(got["l"])
	colorTemperature, colorTemperatureOK := requestInt(got["ct"])
	if got["p"] != false || !brightnessOK || brightness != 100 || !colorTemperatureOK || colorTemperature != 2700 {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestInvokeLightingDesignApplyUsesExplicitDesignActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			t.Fatalf("explicit design actions should not require capability schema read")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-explicit-secret", "client-design-explicit-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-actions","locale":"zh-CN","utterance":"恢复原来的关灯亮度100色温3900","intent":"lighting.design.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"design":{"actions":[{"propertyName":"p","value":false},{"propertyName":"l","value":100},{"propertyName":"ct","value":3900}]}}}`
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
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	actions := preview["payloadPreview"].(map[string]any)["actions"].([]any)
	if len(actions) != 3 {
		t.Fatalf("actions = %#v", actions)
	}
	got := map[string]any{}
	for _, raw := range actions {
		action := raw.(map[string]any)
		got[action["propertyName"].(string)] = action["value"]
		if action["deviceId"] != "device-1" {
			t.Fatalf("action = %#v", action)
		}
	}
	brightness, brightnessOK := requestInt(got["l"])
	colorTemperature, colorTemperatureOK := requestInt(got["ct"])
	if got["p"] != false || !brightnessOK || brightness != 100 || !colorTemperatureOK || colorTemperature != 3900 {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestInvokeLightingDesignApplyOnlyUsesExplicitPowerWhenOnlyPowerProvided(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/schema/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"device-1","name":"主灯","properties":[{"propId":"p"},{"propId":"l"},{"propId":"ct"}]}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-apply-secret", "client-design-apply-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-power-only","locale":"zh-CN","utterance":"只把主灯关掉","intent":"lighting.design.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"power":false}}`
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
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	actions := preview["payloadPreview"].(map[string]any)["actions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("actions = %#v", actions)
	}
	action := actions[0].(map[string]any)
	if action["propertyName"] != "p" || action["value"] != false {
		t.Fatalf("action = %#v", action)
	}
}

func TestInvokeLightingDesignApplyExecutesDirectly(t *testing.T) {
	writeBodies := map[string]map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p",
			"/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/l",
			"/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/ct",
			"/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/c":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			writeBodies[request.URL.Path] = body
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/p":
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/l":
			_, _ = writer.Write([]byte(`{"success":true,"data":20}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/ct":
			_, _ = writer.Write([]byte(`{"success":true,"data":3000}`))
		case "/apis/iot/v1/controll/device/device-1/r/properties/c":
			_, _ = writer.Write([]byte(`{"success":true,"data":3368703}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-design-apply-secret", "client-design-apply-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-execute","locale":"zh-CN","utterance":"应用主灯照明设计","intent":"lighting.design.apply","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"design":{"actions":[{"deviceId":"device-1","propertyName":"p","value":true},{"deviceId":"device-1","propertyName":"l","value":20},{"deviceId":"device-1","propertyName":"ct","value":3000},{"deviceId":"device-1","propertyName":"c","value":3368703}]}}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBodies["/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p"]["value"] != true {
		t.Fatalf("writeBodies = %#v", writeBodies)
	}
	if writeBodies["/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/l"]["value"] != float64(20) {
		t.Fatalf("writeBodies = %#v", writeBodies)
	}
	if writeBodies["/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/ct"]["value"] != float64(3000) {
		t.Fatalf("writeBodies = %#v", writeBodies)
	}
	if writeBodies["/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/c"]["value"] != float64(3368703) {
		t.Fatalf("writeBodies = %#v", writeBodies)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "lighting-design-apply-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["createdArtifacts"].([]any) == nil || result["verified"] != true || result["actionCount"] != float64(4) {
		t.Fatalf("result = %#v", result)
	}
}
