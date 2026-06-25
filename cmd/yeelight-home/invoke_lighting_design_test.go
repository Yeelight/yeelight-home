package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func TestInvokeLightingDesignPlanBuildsLocalPlan(t *testing.T) {
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
	if result["persistentWrites"] != false || result["applyBehavior"] != "pending_plan_required" {
		t.Fatalf("result = %#v", result)
	}
	recipe := result["selectedRecipe"].(map[string]any)
	if recipe["name"] != "观影模式" {
		t.Fatalf("recipe = %#v", recipe)
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

func TestRuntimeLightingCatalogIsSelfContained(t *testing.T) {
	catalog := loadRuntimeLightingCatalog()
	if catalog.Status != "runtime_builtin" || len(catalog.LightingExperience.SceneRecipes) == 0 || len(catalog.LightingExperience.MoodRecipes) == 0 {
		t.Fatalf("catalog = %#v", catalog)
	}
	recipe := selectLightingRecipe(contract.Request{
		Utterance: "给客厅做一个观影灯光方案",
		Parameters: map[string]any{
			"mood": "观影",
		},
	}, catalog)
	if recipe["name"] != "观影模式" {
		t.Fatalf("recipe = %#v", recipe)
	}
}

func TestInvokeLightingDesignApplyCreatesPendingPlanWithoutWriting(t *testing.T) {
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

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-plan","locale":"zh-CN","utterance":"把客厅应用观影灯光设计","intent":"lighting.design.apply","targets":[{"entityType":"room","id":"room-1"}],"parameters":{"mood":"观影","hex":"#3366ff"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/properties/") {
			t.Fatalf("lighting.design.apply should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	planID := confirmation["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "lighting.design.apply" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	actions := record.Payload["actions"].([]any)
	if len(actions) < 2 {
		t.Fatalf("record payload = %#v", record.Payload)
	}
	preview := confirmation["payloadPreview"].(map[string]any)
	if _, ok := preview["actions"]; !ok {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokePlanCommitAppliesLightingDesignFromStoredPlan(t *testing.T) {
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
	record, err := plan.NewRecord("default", "dev", "house-1", "lighting.design.apply", "req-plan", "应用照明设计", map[string]any{
		"houseId": "house-1",
		"actions": []any{
			map[string]any{"deviceId": "device-1", "deviceName": "主灯", "propertyName": "p", "value": true},
			map[string]any{"deviceId": "device-1", "deviceName": "主灯", "propertyName": "l", "value": 20},
			map[string]any{"deviceId": "device-1", "deviceName": "主灯", "propertyName": "ct", "value": 3000},
			map[string]any{"deviceId": "device-1", "deviceName": "主灯", "propertyName": "c", "value": 3368703},
		},
	}, []string{"提交后逐项读取设备状态验证结果"}, time.Now(), pendingPlanTTL)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	input := `{"contractVersion":"1.0","requestId":"req-design-apply-commit","locale":"zh-CN","utterance":"确认应用照明设计","intent":"plan.commit","parameters":{"planId":"` + record.ID + `","actions":[{"deviceId":"ignored","propertyName":"l","value":1}]}}`
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
	if response["status"] != "success" || response["traceId"] != "lighting-design-apply-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["createdArtifacts"].([]any) == nil || result["verified"] != true || result["actionCount"] != float64(4) {
		t.Fatalf("result = %#v", result)
	}
}
