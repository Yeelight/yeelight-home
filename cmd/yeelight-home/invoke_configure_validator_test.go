package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestInvokeAreaCreateRejectsUnknownRoomReference(t *testing.T) {
	response := invokeConfigureWithSeededEntities(t, `{"contractVersion":"1.0","requestId":"req-area-invalid-room","locale":"zh-CN","utterance":"创建一楼区域","intent":"area.create","parameters":{"houseId":"200171","name":"一楼","roomIds":["999999"]}}`)
	assertConfigureClarificationReason(t, response, "invalid_area_resource_reference")
}

func TestInvokeGroupCreateRejectsUnknownReferences(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		reason string
	}{
		{
			name:   "unknown room",
			input:  `{"contractVersion":"1.0","requestId":"req-group-invalid-room","locale":"zh-CN","utterance":"创建客厅灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomId":"999999","groupCapability":"light","deviceIds":["50018430"]}}`,
			reason: "invalid_group_room_reference",
		},
		{
			name:   "unknown device",
			input:  `{"contractVersion":"1.0","requestId":"req-group-invalid-device","locale":"zh-CN","utterance":"创建客厅灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomId":"401398","groupCapability":"light","deviceIds":["999999"]}}`,
			reason: "invalid_group_device_reference",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := invokeConfigureWithSeededEntities(t, test.input)
			assertConfigureClarificationReason(t, response, test.reason)
		})
	}
}

func TestInvokeGroupCreateResolvesRoomAndDeviceNames(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-group-by-names","locale":"zh-CN","utterance":"把客廷的主灯和筒灯建成灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomName":"客廷","groupCapability":"light","deviceNames":["主灯","筒灯"]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	deviceIDs := preview[semantic.FieldDeviceIDs].([]any)
	if preview[semantic.FieldRoomID] != float64(401398) || len(deviceIDs) != 2 || deviceIDs[0] != float64(50018330) || deviceIDs[1] != float64(50018430) {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokeGroupCreateRejectsAmbiguousDeviceNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"dev-1","name":"主灯","roomId":"401398"},{"id":"dev-2","name":"主灯","roomId":"401398"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-group-ambiguous-device","locale":"zh-CN","utterance":"把客厅的主灯建成灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomName":"客厅","groupCapability":"light","deviceNames":["主灯"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	assertConfigureClarificationReason(t, response, "ambiguous_group_device_reference")
	clarification := response["clarification"].(map[string]any)
	candidates, ok := clarification[semantic.FieldCandidates].([]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("clarification should include candidates: %#v", clarification)
	}
}

func TestInvokeSceneCreateRejectsUnknownDeviceDetail(t *testing.T) {
	response := invokeConfigureWithSeededEntities(t, `{"contractVersion":"1.0","requestId":"req-scene-invalid-device","locale":"zh-CN","utterance":"创建回家灯光","intent":"scene.create","parameters":{"houseId":"200171","name":"回家灯光","actions":[{"targetType":"device","targetId":"999999","targetName":"不存在的灯","set":{"power":true}}]}}`)
	assertConfigureClarificationReason(t, response, "invalid_scene_resource_reference")
}

func TestInvokeSceneCreateValidatesCustomGroupTypeAgainstGroups(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-scene-custom-group","locale":"zh-CN","utterance":"创建分组情景","intent":"scene.create","parameters":{"houseId":"200171","name":"分组情景","actions":[{"targetType":"group","targetId":"600001","targetName":"已有灯组","rank":0,"set":{"power":true}}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	actions := preview["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["targetType"] != "meshGroup" || action["targetId"] != "600001" {
		t.Fatalf("group scene action should target meshGroup by default: %#v", action)
	}
}

func TestInvokeSceneCreateResolvesActionTargetName(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-scene-room-by-name","locale":"zh-CN","utterance":"创建打开客廷的情景","intent":"scene.create","parameters":{"houseId":"200171","name":"打开客厅","actions":[{"targetType":"room","targetName":"客廷","rank":0,"set":{"power":true}}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	actions := preview["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["targetType"] != "room" || action["targetId"] != "401398" || action["targetName"] != "客厅" {
		t.Fatalf("action = %#v", action)
	}
}

func TestInvokeSceneCreateDoesNotTreatCustomGroupTypeAsArea(t *testing.T) {
	response := invokeConfigureWithSeededEntities(t, `{"contractVersion":"1.0","requestId":"req-scene-area-as-custom","locale":"zh-CN","utterance":"创建区域情景","intent":"scene.create","parameters":{"houseId":"200171","name":"区域情景","actions":[{"targetType":"group","targetId":"300001","targetName":"南区","rank":0,"set":{"power":true}}]}}`)
	assertConfigureClarificationReason(t, response, "invalid_scene_resource_reference")
}

func TestInvokeSceneCreateInvalidPayloadReturnsNestedPayloadGuide(t *testing.T) {
	response := invokeConfigureWithSeededEntities(t, `{"contractVersion":"1.0","requestId":"req-scene-create-guide","locale":"zh-CN","utterance":"创建孩子屋开灯情景","intent":"scene.create","parameters":{"houseId":"200171","name":"孩子屋开灯","set":{"colorTemperature":3000}}}`)
	assertConfigureClarificationReason(t, response, "invalid_scene_create_payload")
	clarification := response["clarification"].(map[string]any)
	shape := clarification[semantic.FieldPayloadShape].(map[string]any)
	actions := shape[semantic.FieldActions].([]any)
	action := actions[0].(map[string]any)
	if action[semantic.FieldTargetType] == nil || action[semantic.FieldSet] == nil || clarification[semantic.FieldExamples] == nil || !strings.Contains(requestString(clarification[semantic.FieldNextStep]), "scene.create") {
		t.Fatalf("scene payload guide incomplete: %#v", clarification)
	}
}

func TestInvokeAutomationCreateRejectsInvalidStructure(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		reason string
	}{
		{
			name:   "params must be and with conditions",
			input:  `{"contractVersion":"1.0","requestId":"req-auto-invalid-params","locale":"zh-CN","utterance":"每天晚上十点关灯","intent":"automation.create","parameters":{"houseId":"200171","name":"每天关灯","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","conditions":[],"actions":[{"targetType":"device","targetId":"50018330","targetName":"主灯","rank":0,"set":{"power":false}}]}}`,
			reason: "invalid_automation_params",
		},
		{
			name:   "status cannot be deleted or unknown",
			input:  `{"contractVersion":"1.0","requestId":"req-auto-invalid-status","locale":"zh-CN","utterance":"每天晚上十点关灯","intent":"automation.create","parameters":{"houseId":"200171","name":"每天关灯","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","status":2,"conditions":[{"conditionKind":"timer","time":"22:00:00"}],"actions":[{"targetType":"device","targetId":"50018330","targetName":"主灯","rank":0,"set":{"power":false}}]}}`,
			reason: "invalid_automation_status",
		},
		{
			name:   "unknown action device",
			input:  `{"contractVersion":"1.0","requestId":"req-auto-invalid-action","locale":"zh-CN","utterance":"每天晚上十点关灯","intent":"automation.create","parameters":{"houseId":"200171","name":"每天关灯","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","conditions":[{"conditionKind":"timer","time":"22:00:00"}],"actions":[{"targetType":"device","targetId":"999999","targetName":"不存在的灯","rank":0,"set":{"power":false}}]}}`,
			reason: "invalid_automation_action_reference",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := invokeConfigureWithSeededEntities(t, test.input)
			assertConfigureClarificationReason(t, response, test.reason)
		})
	}
}

func TestInvokeAutomationCreateInvalidPayloadReturnsNestedPayloadGuide(t *testing.T) {
	response := invokeConfigureWithSeededEntities(t, `{"contractVersion":"1.0","requestId":"req-auto-create-guide","locale":"zh-CN","utterance":"创建主卧9点开灯自动化","intent":"automation.create","parameters":{"houseId":"200171","name":"主卧9点开灯","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","trigger":{"conditionKind":"alarm","time":"09:00:00"}}}`)
	assertConfigureClarificationReason(t, response, "invalid_automation_create_payload")
	clarification := response["clarification"].(map[string]any)
	shape := clarification[semantic.FieldPayloadShape].(map[string]any)
	trigger, ok := shape[semantic.FieldTrigger].(map[string]any)
	if !ok || trigger[semantic.FieldTime] == nil {
		t.Fatalf("trigger guide incomplete: %#v", shape[semantic.FieldTrigger])
	}
	actions := shape[semantic.FieldActions].([]any)
	action := actions[0].(map[string]any)
	if action[semantic.FieldTargetType] == nil || action[semantic.FieldSet] == nil || clarification[semantic.FieldExamples] == nil {
		t.Fatalf("automation payload guide incomplete: %#v", clarification)
	}
}

func TestInvokeAutomationCreateAcceptsSceneActionWithoutSet(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-auto-scene-action","locale":"zh-CN","utterance":"每天九点执行已有情景","intent":"automation.create","parameters":{"houseId":"200171","name":"每天执行情景","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","trigger":{"conditionKind":"alarm","time":"09:00:00"},"actions":[{"targetType":"scene","targetId":"700001","targetName":"已有情景","rank":0}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	actions := preview["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["targetType"] != "scene" || action["targetId"] != "700001" {
		t.Fatalf("action = %#v", action)
	}
}

func TestInvokeAutomationCreateDefaultsActionRank(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-auto-default-rank","locale":"zh-CN","utterance":"每天九点关闭已有灯组","intent":"automation.create","parameters":{"houseId":"200171","name":"每天关闭灯组","repeat":"daily","trigger":{"conditionKind":"alarm","time":"09:00:00"},"actions":[{"targetType":"group","targetName":"已有灯组","set":{"power":false}}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	actions := preview["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["rank"] != float64(0) {
		t.Fatalf("action = %#v", action)
	}
}

func TestInvokeAutomationCreatePreviewUsesPublicConditionFields(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-auto-public-condition-preview","locale":"zh-CN","utterance":"主灯打开后亮度大于10就柔和一点","intent":"automation.create","parameters":{"houseId":"200171","name":"主灯打开后柔和","repeat":"daily","conditions":[{"conditionKind":"fact_change","targetType":"device","targetId":"50018330","capabilityPid":198666,"property":"power","value":true},{"conditionKind":"fact","targetType":"device","targetId":"50018330","capabilityPid":198666,"property":"brightness","operation":"gt","value":10}],"actions":[{"targetType":"device","targetId":"50018330","targetName":"主灯","set":{"power":true,"brightness":35,"colorTemperature":3000}}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	conditionGroup := preview[semantic.FieldConditions].(map[string]any)
	if conditionGroup[semantic.FieldConditionType] != "and" {
		t.Fatalf("condition group = %#v", conditionGroup)
	}
	conditions := conditionGroup[semantic.FieldConditions].([]any)
	first := conditions[0].(map[string]any)
	if first[semantic.FieldConditionKind] != "fact_change" || first[semantic.FieldTargetType] != "device" || first[semantic.FieldTargetID] != float64(50018330) || first[semantic.FieldCapabilityProductID] != float64(198666) || first[semantic.FieldProperty] != semantic.FieldPower {
		t.Fatalf("first condition = %#v", first)
	}
	second := conditions[1].(map[string]any)
	if second[semantic.FieldConditionKind] != "fact" || second[semantic.FieldProperty] != semantic.FieldBrightness || second[semantic.FieldOperation] != "gt" || second[semantic.FieldValue] != float64(10) {
		t.Fatalf("second condition = %#v", second)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	for _, forbidden := range []string{`"resId"`, `"typeId"`, `"prop"`, `"pid"`, `"clock"`, `"extArgs"`} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("automation preview leaked internal field %s: %s", forbidden, string(encoded))
		}
	}
}

func TestInvokeAutomationCreateResolvesSceneActionTargetName(t *testing.T) {
	response := invokeConfigureWithSeededEntitiesDryRun(t, `{"contractVersion":"1.0","requestId":"req-auto-scene-action-by-name","locale":"zh-CN","utterance":"每天九点执行已有情景","intent":"automation.create","parameters":{"houseId":"200171","name":"每天执行情景","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","trigger":{"conditionKind":"alarm","time":"09:00:00"},"actions":[{"targetType":"scene","targetName":"已有情景","rank":0}]}}`)
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	actions := preview["actions"].([]any)
	action := actions[0].(map[string]any)
	if action["targetType"] != "scene" || action["targetId"] != "700001" || action["targetName"] != "已有情景" {
		t.Fatalf("action = %#v", action)
	}
}

func TestInvokeSceneCreateRejectsAmbiguousActionTargetName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"},{"id":"room-2","name":"客停"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-scene-ambiguous-room-by-name","locale":"zh-CN","utterance":"创建打开客廷的情景","intent":"scene.create","parameters":{"houseId":"200171","name":"打开客厅","actions":[{"targetType":"room","targetName":"客廷","rank":0,"set":{"power":true}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	assertConfigureClarificationReason(t, response, "ambiguous_scene_resource_reference")
}

func TestConfigureCreateRejectsSourceBackedCountLimits(t *testing.T) {
	tests := []struct {
		name       string
		intent     string
		parameters map[string]any
		reason     string
	}{
		{
			name:   "area rooms",
			intent: "area.create",
			parameters: map[string]any{
				"houseId": "200171",
				"name":    "超大区域",
				"roomIds": numberedStrings(100000, areaRoomLimit+1),
			},
			reason: "area_room_limit_exceeded",
		},
		{
			name:   "group devices",
			intent: "group.create",
			parameters: map[string]any{
				"houseId":         "200171",
				"name":            "超大灯组",
				"roomId":          "401398",
				"groupCapability": "light",
				"deviceIds":       numberedStrings(500000, groupDeviceLimit+1),
			},
			reason: "group_device_limit_exceeded",
		},
		{
			name:   "scene actions",
			intent: "scene.create",
			parameters: map[string]any{
				"houseId": "200171",
				"name":    "超大情景",
				"actions": repeatedActions(sceneActionLimit + 1),
			},
			reason: "scene_action_limit_exceeded",
		},
		{
			name:   "automation conditions",
			intent: "automation.create",
			parameters: map[string]any{
				"houseId":      "200171",
				"name":         "超大自动化条件",
				"activeWindow": map[string]any{"start": "00:00:00", "end": "23:59:59"},
				"repeat":       "daily",
				"conditions":   repeatedConditions(automationIfLimit + 1),
				"actions":      repeatedActions(1),
			},
			reason: "automation_condition_limit_exceeded",
		},
		{
			name:   "automation actions",
			intent: "automation.create",
			parameters: map[string]any{
				"houseId":      "200171",
				"name":         "超大自动化动作",
				"activeWindow": map[string]any{"start": "00:00:00", "end": "23:59:59"},
				"repeat":       "daily",
				"conditions":   repeatedConditions(1),
				"actions":      repeatedActions(automationThenLimit + 1),
			},
			reason: "automation_action_limit_exceeded",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := marshalConfigureRequest(t, test.intent, test.parameters)
			response := invokeConfigureWithSeededEntities(t, input)
			assertConfigureClarificationReason(t, response, test.reason)
		})
	}
}

func TestConfigureCreateRejectsHouseScopedCountLimits(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		counts     map[string]int
		payload    map[string]any
		reason     string
	}{
		{
			name:       "room total",
			entityType: "room",
			counts:     map[string]int{"room": houseRoomLimit},
			reason:     "house_room_limit_exceeded",
		},
		{
			name:       "area total",
			entityType: "area",
			counts:     map[string]int{"area": houseAreaLimit},
			payload:    map[string]any{},
			reason:     "house_area_limit_exceeded",
		},
		{
			name:       "group total",
			entityType: "group",
			counts:     map[string]int{"group": houseGroupLimit},
			payload:    map[string]any{"roomId": "401398"},
			reason:     "house_group_limit_exceeded",
		},
		{
			name:       "scene total",
			entityType: "scene",
			counts:     map[string]int{"scene": houseSceneLimit},
			payload:    map[string]any{"actions": []map[string]any{{"targetType": "device", "targetId": "50018330", "set": map[string]any{"power": true}}}},
			reason:     "house_scene_limit_exceeded",
		},
		{
			name:       "automation total",
			entityType: "automation",
			counts:     map[string]int{"automation": houseAutomationLimit},
			payload: map[string]any{
				"conditions": []any{map[string]any{"conditionKind": "timer", "time": "22:00:00"}},
				"actions":    []map[string]any{{"targetType": "device", "targetId": "50018330", "set": map[string]any{"power": false}}},
			},
			reason: "house_automation_limit_exceeded",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reason := validateConfigureCreatePayload(test.entityType, test.payload, api.EntityListResult{Counts: test.counts})
			if reason != test.reason {
				t.Fatalf("reason = %q", reason)
			}
		})
	}
}

func TestInvokeRoomCreateRejectsHouseRoomLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeHouseRoomLimitListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-limit","locale":"zh-CN","utterance":"创建一个书房","intent":"room.create","parameters":{"houseId":"200171","name":"书房"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	assertConfigureClarificationReason(t, response, "house_room_limit_exceeded")
}

func invokeConfigureWithSeededEntities(t *testing.T, input string) map[string]any {
	t.Helper()
	return invokeConfigureWithSeededEntitiesArgs(t, input, []string{"invoke", "--stdin"})
}

func invokeConfigureWithSeededEntitiesDryRun(t *testing.T, input string) map[string]any {
	t.Helper()
	return invokeConfigureWithSeededEntitiesArgs(t, input, []string{"invoke", "--stdin", "--dry-run"})
}

func invokeConfigureWithSeededEntitiesArgs(t *testing.T, input string, args []string) map[string]any {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run(args, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	return decodeInvokeResponse(t, stdout.Bytes())
}

func assertConfigureClarificationReason(t *testing.T, response map[string]any, reason string) {
	t.Helper()
	if response["status"] != "clarification_required" || response["traceId"] != "configure-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != reason {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func marshalConfigureRequest(t *testing.T, intent string, parameters map[string]any) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"contractVersion": "1.0",
		"requestId":       "req-" + strings.ReplaceAll(intent, ".", "-"),
		"locale":          "zh-CN",
		"utterance":       "测试配置",
		"intent":          intent,
		"parameters":      parameters,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return string(data)
}

func numberedStrings(start int, count int) []string {
	result := make([]string, 0, count)
	for index := 0; index < count; index++ {
		result = append(result, strconv.Itoa(start+index))
	}
	return result
}

func repeatedActions(count int) []map[string]any {
	result := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		result = append(result, map[string]any{
			"targetType": "device",
			"targetId":   "50018330",
			"targetName": "主灯",
			"rank":       index,
			"set":        map[string]any{"power": false},
		})
	}
	return result
}

func repeatedConditions(count int) []map[string]any {
	result := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		result = append(result, map[string]any{
			"conditionKind": "timer",
			"time":          "22:" + fmtTwoDigits(index) + ":00",
		})
	}
	return result
}

func fmtTwoDigits(value int) string {
	return strconv.Itoa(value/10) + strconv.Itoa(value%10)
}

func writeSeededHouseScopedListForConfigureTest(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"300001","name":"南区"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401398"},{"id":"50018430","name":"筒灯","roomId":"401398"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"600001","name":"已有灯组"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"700001","name":"已有情景"}]}}`))
	case "/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	default:
		http.NotFound(writer, request)
	}
}

func writeHouseRoomLimitListForConfigureTest(writer http.ResponseWriter, request *http.Request) {
	switch {
	case request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		request.URL.Path == "/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case strings.HasPrefix(request.URL.Path, "/apis/iot/v2/thing/manage/house/200171/room/r/info/"):
		pageNo := roomListPageNo(request.URL.Path)
		if pageNo >= 1 && pageNo <= 5 {
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[` + configureTestEntityRows("room", "房间", (pageNo-1)*100+1, 100) + `]}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	default:
		http.NotFound(writer, request)
	}
}

func roomListPageNo(path string) int {
	parts := strings.Split(path, "/")
	for index, part := range parts {
		if part == "info" && index+1 < len(parts) {
			pageNo, _ := strconv.Atoi(parts[index+1])
			return pageNo
		}
	}
	return 0
}

func configureTestEntityRows(idPrefix string, namePrefix string, first int, count int) string {
	var builder strings.Builder
	for index := 0; index < count; index++ {
		if index > 0 {
			builder.WriteString(",")
		}
		id := first + index
		_, _ = fmt.Fprintf(&builder, `{"id":"%s-%d","name":"%s %d"}`, idPrefix, id, namePrefix, id)
	}
	return builder.String()
}
