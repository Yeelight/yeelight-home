package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestMetadataReadonlyReadPathBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":600,"message":"参数格式错误"}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceWeatherGet(context.Background(), MetadataReadonlyRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Parameters: map[string]any{
			"queryType": "default",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceWeatherGet error = %v", err)
	}
	if !result.Partial || result.Capability != "device.weather.get" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if result.Data != nil {
		t.Fatalf("partial business result should not expose raw data: %#v", result.Data)
	}
}

func TestAutomationSupportedListProjectsPublicConditions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/automations/r/supported/v2" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":8784640,"actions":[{"id":1,"type":"event","desc":[{"languageId":"1","value":"Button event"},{"languageId":"2","value":"按键事件"}],"argsDesc":[{"type":"eventId","dataType":"int","unit":"","valueRange":"1,2"}],"supportVersion":"v1,v2"}]}]}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunAutomationSupportedList(context.Background(), MetadataReadonlyRequest{
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	}, true)
	if err != nil {
		t.Fatalf("RunAutomationSupportedList error = %v", err)
	}
	data := result.Data.(map[string]any)
	rows := data[semantic.FieldSupportedV2].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows = %#v", rows)
	}
	row := rows[0].(map[string]any)
	if row[semantic.FieldCapabilityPID] != float64(8784640) {
		t.Fatalf("row = %#v", row)
	}
	conditions := row[semantic.FieldConditions].([]any)
	condition := conditions[0].(map[string]any)
	if condition[semantic.FieldConditionKind] != "event" || condition[semantic.FieldName] != "按键事件" {
		t.Fatalf("condition = %#v", condition)
	}
	input := condition[semantic.FieldInputs].([]any)[0].(map[string]any)
	if input[semantic.FieldKey] != "eventId" || input[semantic.FieldInputType] != "int" || input[semantic.FieldValueRange] != "1,2" {
		t.Fatalf("input = %#v", input)
	}
	versions := condition[semantic.FieldSupportedVersions].([]any)
	if len(versions) != 2 || versions[0] != "v1" || versions[1] != "v2" {
		t.Fatalf("versions = %#v", versions)
	}
	encoded, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{`"pid"`, `"actions"`, `"desc"`, `"argsDesc"`, `"supportVersion"`, `"dataType"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("supported projection leaked %s: %s", forbidden, text)
		}
	}
}

func TestAutomationListPageProjectsPublicRuleShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/automations/house-1/r/list/1/20" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","houseId":"house-1","name":"开灯后柔和","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","version":2,"status":1,"ruleId":"rule-1","set":{"type":"and","conditions":[{"type":"fact_change","typeId":2,"resId":"dev-1","pid":198666,"prop":"p","value":true},{"type":"fact","typeId":2,"resId":"dev-1","pid":198666,"prop":"l","operation":"gt","value":10}]},"actions":[{"typeId":2,"resId":"dev-1","resName":"主灯","rank":0,"params":"{\"set\":{\"p\":true,\"l\":35,\"ct\":3000}}"}]}],"total":"1"}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunAutomationListPage(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunAutomationListPage error = %v", err)
	}
	data := result.Data.(map[string]any)
	page := data[semantic.FieldAutomations].(map[string]any)
	entries := page[semantic.FieldEntries].([]any)
	automation := entries[0].(map[string]any)
	conditions := automation[semantic.FieldConditions].([]any)
	firstCondition := conditions[0].(map[string]any)
	if firstCondition[semantic.FieldConditionKind] != "fact_change" || firstCondition[semantic.FieldTargetType] != "device" || firstCondition[semantic.FieldTargetID] != "dev-1" || firstCondition[semantic.FieldProperty] != semantic.FieldPower {
		t.Fatalf("first condition = %#v", firstCondition)
	}
	actions := automation[semantic.FieldActions].([]any)
	set := actions[0].(map[string]any)[semantic.FieldSet].(map[string]any)
	if set[semantic.FieldPower] != true || set[semantic.FieldBrightness] != float64(35) || set[semantic.FieldColorTemperature] != float64(3000) {
		t.Fatalf("actions = %#v", actions)
	}
	assertNoAutomationInternalFields(t, result.Data)
}

func TestAutomationRuleListProjectsPublicRuleShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/rule/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"rule-1","houseId":"house-1","name":"automation rule","createTime":1782958039,"updateTime":1782958040,"set":{"if":{"type":"and","conditions":[{"type":"time","after":"00:00:00","before":"23:59:59","repeatType":2,"weekdays":"0x7f"},{"type":"alarm","clock":"09:00:00","operation":"eq","repeatType":2,"weekdays":"0x7f"}]},"then":{"method":"gateway_set.prop","nodes":[{"id":"dev-1","nt":2,"duration":1200,"set":{"p":true,"l":60,"ct":3000}}]}},"status":1,"targetId":"auto-1","targetType":"automation","valid":1,"version":2}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunAutomationRuleList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunAutomationRuleList error = %v", err)
	}
	data := result.Data.(map[string]any)
	rules := data[semantic.FieldRules].([]any)
	rule := rules[0].(map[string]any)
	if rule[semantic.FieldRepeat] != "daily" {
		t.Fatalf("rule repeat = %#v", rule)
	}
	trigger := rule[semantic.FieldTrigger].(map[string]any)
	if trigger[semantic.FieldConditionKind] != "alarm" || trigger[semantic.FieldTime] != "09:00:00" {
		t.Fatalf("trigger = %#v", trigger)
	}
	actions := rule[semantic.FieldActions].([]any)
	action := actions[0].(map[string]any)
	set := action[semantic.FieldSet].(map[string]any)
	if action[semantic.FieldTargetType] != "device" || action[semantic.FieldTargetID] != "dev-1" || set[semantic.FieldBrightness] != float64(60) {
		t.Fatalf("actions = %#v", actions)
	}
	assertNoAutomationInternalFields(t, result.Data)
}

func assertNoAutomationInternalFields(t *testing.T, value any) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{`"if"`, `"then"`, `"nodes"`, `"nt"`, `"clock"`, `"typeId"`, `"resId"`, `"pid"`, `"prop"`, `"repeatType"`, `"weekdays"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("automation projection leaked %s: %s", forbidden, text)
		}
	}
}

func TestSceneDetailGetReturnsEditablePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/scene/scene-1/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"scene-1","name":"孩子屋开灯","desc":"暖光","details":[{"typeId":2,"resId":50018330,"resName":"孩子屋吸顶灯","action":0,"rank":0,"params":"{\"set\":{\"p\":true,\"ct\":3000,\"l\":60}}","accessToken":"not-allowed"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunSceneDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "200171",
		Parameters:  map[string]any{"sceneId": "scene-1"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunSceneDetailGet error = %v", err)
	}
	data := result.Data.(map[string]any)
	payload := data["editablePayload"].(map[string]any)
	if payload["sceneId"] != "scene-1" || payload["name"] != "孩子屋开灯" {
		t.Fatalf("payload = %#v", payload)
	}
	actions := payload["actions"].([]any)
	set := actions[0].(map[string]any)["set"].(map[string]any)
	if set["colorTemperature"] != float64(3000) || set["brightness"] != float64(60) {
		t.Fatalf("actions = %#v", actions)
	}
	detailActions := data["detail"].(map[string]any)["actions"].([]any)
	detailAction := detailActions[0].(map[string]any)
	if detailAction["targetType"] != "device" || detailAction["targetId"] != float64(50018330) || detailAction["targetName"] != "孩子屋吸顶灯" {
		t.Fatalf("detail actions lost public target fields: %#v", detailActions)
	}
	detailSet := detailAction["set"].(map[string]any)
	if detailSet["power"] != true || detailSet["colorTemperature"] != float64(3000) || detailSet["brightness"] != float64(60) {
		t.Fatalf("detail actions lost public set: %#v", detailActions)
	}
	for _, forbidden := range []string{"details", "params", "typeId", "resId", "p", "l", "ct"} {
		if _, ok := detailAction[forbidden]; ok {
			t.Fatalf("detail action leaked internal field %q: %#v", forbidden, detailAction)
		}
		if _, ok := detailSet[forbidden]; ok {
			t.Fatalf("detail set leaked internal field %q: %#v", forbidden, detailSet)
		}
	}
	if text, ok := data["detail"].(map[string]any)["accessToken"].(string); ok && text != "" {
		t.Fatalf("detail leaked sensitive value: %#v", data["detail"])
	}
	updateShape := data["updateShape"].(map[string]any)
	actionShape := updateShape["actions"].([]any)
	flow := updateShape["flow"].([]string)
	if actionShape[0].(map[string]any)["set"].(map[string]any)["colorTemperature"] == nil || !updateShape["completeList"].(bool) || len(flow) == 0 || flow[0] != "call scene.detail.get" {
		t.Fatalf("data = %#v", data)
	}
}

func TestAutomationDetailGetReturnsEditablePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/200171/automation/auto-1/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"auto-1","name":"主卧每天9点开灯","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","version":3,"params":"{\"type\":\"and\",\"conditions\":[{\"type\":\"alarm\",\"clock\":\"09:00:00\"}]}","actions":[{"typeId":2,"resId":50018330,"resName":"主卧吸顶灯","rank":0,"params":"{\"set\":{\"p\":true,\"ct\":3000,\"l\":60}}"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunAutomationDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "200171",
		Parameters:  map[string]any{"automationId": "auto-1"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunAutomationDetailGet error = %v", err)
	}
	data := result.Data.(map[string]any)
	payload := data["editablePayload"].(map[string]any)
	if payload["automationId"] != "auto-1" || payload["repeat"] != "daily" || payload["version"] != float64(3) {
		t.Fatalf("payload = %#v", payload)
	}
	activeWindow := payload["activeWindow"].(map[string]any)
	if activeWindow["start"] != "00:00:00" || activeWindow["end"] != "23:59:59" {
		t.Fatalf("activeWindow = %#v", activeWindow)
	}
	trigger := payload["trigger"].(map[string]any)
	if trigger["time"] != "09:00:00" {
		t.Fatalf("trigger = %#v", trigger)
	}
	actions := payload["actions"].([]any)
	if actions[0].(map[string]any)["set"].(map[string]any)["colorTemperature"] != float64(3000) {
		t.Fatalf("actions = %#v", actions)
	}
	updateShape := data["updateShape"].(map[string]any)
	triggerShape := updateShape["trigger"].(map[string]any)
	actionShape := updateShape["actions"].([]any)
	flow := updateShape["flow"].([]string)
	if triggerShape["time"] == nil || actionShape[0].(map[string]any)["set"] == nil || !updateShape["completeRule"].(bool) || len(flow) == 0 || flow[0] != "call automation.detail.get" {
		t.Fatalf("data = %#v", data)
	}
}
