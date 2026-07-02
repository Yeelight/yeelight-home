package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeLightingDesignImportPayloadAcceptsSemanticDesignModel(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", semanticLightingDesignFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if payload["houseId"] != float64(200191) || payload["name"] != "粒粒的美丽家庭" || payload["version"] != 2 {
		t.Fatalf("payload base fields = %#v", payload)
	}
	gateway := payload["gateway"].(map[string]any)
	if gateway["tempId"] != "1" {
		t.Fatalf("gateway tempId=%#v", gateway["tempId"])
	}
	rooms := gateway["roomList"].([]any)
	if len(rooms) != 1 {
		t.Fatalf("rooms=%#v", rooms)
	}
	room := rooms[0].(map[string]any)
	devices := room["deviceList"].([]any)
	if len(devices) != 2 {
		t.Fatalf("devices=%#v", devices)
	}
	firstDevice := devices[0].(map[string]any)
	pid, _ := lightingDesignIntFromAny(firstDevice["pid"])
	if firstDevice["roomTempId"] != nil || pid != 198666 {
		t.Fatalf("first device=%#v", firstDevice)
	}
	if firstDevice["materialCode"] != "1-000002044" {
		t.Fatalf("first device materialCode=%#v device=%#v", firstDevice["materialCode"], firstDevice)
	}
	extra := firstDevice["extraMeta"].(map[string]string)
	if extra["materialCode"] != "1-000002044" {
		t.Fatalf("extraMeta=%#v", extra)
	}
	groups := room["groupList"].([]any)
	group := groups[0].(map[string]any)
	componentID, _ := lightingDesignIntFromAny(group["componentId"])
	if componentID != 4 {
		t.Fatalf("groups=%#v", groups)
	}
	if group["groupCategory"] != nil || group["groupCapability"] != nil || group["slotKeys"] != nil {
		t.Fatalf("design-only group fields leaked: %#v", group)
	}
	scene := payload["sceneList"].([]any)[0].(map[string]any)
	detail := scene["details"].([]any)[0].(map[string]any)
	typeID, _ := lightingDesignIntFromAny(detail["typeId"])
	action, _ := lightingDesignIntFromAny(detail["action"])
	if typeID != 4 || detail["tempId"] != "gp1" || action != 0 {
		t.Fatalf("scene detail=%#v", detail)
	}
	if detail["params"] != `{"delay":0,"set":{"ct":3000,"l":60,"p":true}}` {
		t.Fatalf("scene detail params=%#v", detail["params"])
	}
	automation := payload["automationList"].([]any)[0].(map[string]any)
	if automation["version"] != 2 || automation["params"] != `{"conditions":[{"conditions":[{"clock":"09:00:00","type":"alarm"}],"type":"or"}],"type":"and"}` {
		t.Fatalf("automation=%#v", automation)
	}
}

func TestNormalizeLightingDesignImportPayloadPreservesAutomationV2ConditionGroups(t *testing.T) {
	payload := semanticLightingDesignFixture()
	automation := payload["automations"].([]any)[0].(map[string]any)
	automation["trigger"] = map[string]any{
		"conditionType": "and",
		"conditions": []any{
			map[string]any{
				"conditionType": "or",
				"conditions": []any{
					map[string]any{"conditionKind": "alarm", "time": "09:00:00"},
				},
			},
		},
	}
	normalized, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	automationMeta := normalized["automationList"].([]any)[0].(map[string]any)
	if automationMeta["params"] != `{"conditions":[{"conditions":[{"clock":"09:00:00","type":"alarm"}],"type":"or"}],"type":"and"}` {
		t.Fatalf("automation params=%#v", automationMeta["params"])
	}
}

func TestNormalizeLightingDesignImportPayloadMapsEventAndFactAutomationConditions(t *testing.T) {
	payload := semanticLightingDesignFixture()
	automation := payload["automations"].([]any)[0].(map[string]any)
	automation["trigger"] = map[string]any{
		"conditionType": "and",
		"conditions": []any{
			map[string]any{
				"conditionKind": "event",
				"targetType":    "device",
				"targetKey":     "dv1",
				"capabilityPid": 198666,
				"eventId":       42,
				"eventArgs":     map[string]any{"arg1": 423},
			},
			map[string]any{
				"conditionKind": "fact",
				"targetType":    "device",
				"targetKey":     "dv2",
				"capabilityPid": 198666,
				"property":      "brightness",
				"operation":     "gt",
				"value":         10,
			},
		},
	}

	params := normalizedAutomationParams(t, payload)
	groups := params["conditions"].([]any)
	if len(groups) != 2 {
		t.Fatalf("condition groups=%#v", groups)
	}
	eventGroup := groups[0].(map[string]any)
	if eventGroup["type"] != "or" {
		t.Fatalf("event group=%#v", eventGroup)
	}
	event := eventGroup["conditions"].([]any)[0].(map[string]any)
	if event["type"] != "event" || event["tempId"] != "dv1" || event["typeId"] != float64(2) || event["pid"] != float64(198666) || event["id"] != float64(42) {
		t.Fatalf("event condition=%#v", event)
	}
	if args := event["extArgs"].(map[string]any); args["arg1"] != float64(423) {
		t.Fatalf("event args=%#v", args)
	}
	factGroup := groups[1].(map[string]any)
	if factGroup["type"] != "and" {
		t.Fatalf("fact group=%#v", factGroup)
	}
	fact := factGroup["conditions"].([]any)[0].(map[string]any)
	if fact["type"] != "fact" || fact["tempId"] != "dv2" || fact["typeId"] != float64(2) || fact["pid"] != float64(198666) || fact["prop"] != "l" || fact["operation"] != "gt" || fact["value"] != float64(10) {
		t.Fatalf("fact condition=%#v", fact)
	}
}

func TestNormalizeLightingDesignImportPayloadMapsFactChangeAutomationCondition(t *testing.T) {
	payload := semanticLightingDesignFixture()
	automation := payload["automations"].([]any)[0].(map[string]any)
	automation["trigger"] = map[string]any{
		"conditionKind": "fact_change",
		"targetType":    "device",
		"targetKey":     "dv1",
		"capabilityPid": 198666,
		"property":      "power",
		"value":         true,
	}

	params := normalizedAutomationParams(t, payload)
	groups := params["conditions"].([]any)
	if len(groups) != 1 {
		t.Fatalf("condition groups=%#v", groups)
	}
	group := groups[0].(map[string]any)
	if group["type"] != "or" {
		t.Fatalf("fact_change group=%#v", group)
	}
	condition := group["conditions"].([]any)[0].(map[string]any)
	if condition["type"] != "fact_change" || condition["tempId"] != "dv1" || condition["typeId"] != float64(2) || condition["pid"] != float64(198666) || condition["prop"] != "p" || condition["value"] != true {
		t.Fatalf("fact_change condition=%#v", condition)
	}
}

func TestNormalizeLightingDesignImportPayloadPreservesFactGroupConditionType(t *testing.T) {
	payload := semanticLightingDesignFixture()
	automation := payload["automations"].([]any)[0].(map[string]any)
	automation["trigger"] = map[string]any{
		"conditionType": "and",
		"conditions": []any{
			map[string]any{
				"conditionType": "or",
				"conditions": []any{
					map[string]any{
						"conditionKind": "fact",
						"targetType":    "device",
						"targetKey":     "dv1",
						"capabilityPid": 198666,
						"property":      "motionDetected",
						"value":         true,
					},
				},
			},
		},
	}

	params := normalizedAutomationParams(t, payload)
	group := params["conditions"].([]any)[0].(map[string]any)
	if group["type"] != "or" {
		t.Fatalf("fact group type should be preserved, params=%#v", params)
	}
	condition := group["conditions"].([]any)[0].(map[string]any)
	if condition["prop"] != "mv" {
		t.Fatalf("fact condition=%#v", condition)
	}
}

func TestLightingDesignVerificationRequiresAutomations(t *testing.T) {
	if lightingDesignVerificationPasses(EntityListResult{
		Counts: map[string]int{"room": 1, "device": 1, "group": 1, "scene": 1},
	}, map[string]int{
		"rooms":       1,
		"devices":     1,
		"groups":      1,
		"scenes":      1,
		"automations": 1,
	}) {
		t.Fatal("expected verification to fail when automation count is missing")
	}
	if !lightingDesignVerificationPasses(EntityListResult{
		Counts: map[string]int{"room": 1, "device": 1, "group": 1, "scene": 1, "automation": 1},
	}, map[string]int{
		"rooms":       1,
		"devices":     1,
		"groups":      1,
		"scenes":      1,
		"automations": 1,
	}) {
		t.Fatal("expected verification to pass when all requested entity types exist")
	}
}

func TestLightingDesignVerificationRequiresRequestedCounts(t *testing.T) {
	if lightingDesignVerificationPasses(EntityListResult{
		Counts: map[string]int{"room": 4, "device": 18, "group": 1, "scene": 2, "automation": 1},
	}, map[string]int{
		"rooms":       4,
		"devices":     18,
		"groups":      5,
		"scenes":      2,
		"automations": 1,
	}) {
		t.Fatal("expected verification to fail when only part of requested groups were imported")
	}
	if !lightingDesignVerificationPasses(EntityListResult{
		Counts: map[string]int{"room": 4, "device": 18, "group": 5, "scene": 2, "automation": 1},
	}, map[string]int{
		"rooms":       4,
		"devices":     18,
		"groups":      5,
		"scenes":      2,
		"automations": 1,
	}) {
		t.Fatal("expected verification to pass when requested counts are present")
	}
}

func normalizedAutomationParams(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	normalized, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	automationMeta := normalized["automationList"].([]any)[0].(map[string]any)
	var params map[string]any
	if err := json.Unmarshal([]byte(automationMeta["params"].(string)), &params); err != nil {
		t.Fatalf("automation params JSON: %v", err)
	}
	return params
}

func TestNormalizeLightingDesignImportPayloadAllowsNewHomeWithoutHouseID(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("", semanticLightingDesignFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if _, ok := payload["houseId"]; ok {
		t.Fatalf("new-home import must not inject a profile/default houseId: %#v", payload)
	}
	if payload["name"] != "粒粒的美丽家庭" || payload["version"] != 2 {
		t.Fatalf("payload base fields = %#v", payload)
	}
}

func TestNormalizeLightingDesignImportPayloadRejectsNaturalTopology(t *testing.T) {
	_, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{"name": "客厅", "items": []any{map[string]any{"name": "吸顶灯"}}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "rooms[].deviceSlots[]") {
		t.Fatalf("expected design model guidance error, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadRejectsLegacyOverwriteFlags(t *testing.T) {
	payload := semanticLightingDesignFixture()
	payload["clearAll"] = true
	_, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err == nil || !strings.Contains(err.Error(), "clearAll/overwrite is not supported") {
		t.Fatalf("expected clearAll rejection, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadIgnoresInternalProductFields(t *testing.T) {
	payload := semanticLightingDesignFixture()
	slot := payload["rooms"].([]any)[0].(map[string]any)["deviceSlots"].([]any)[0].(map[string]any)
	product := slot["product"].(map[string]any)
	product["materialCode"] = "bad-code"
	product["pid"] = 1
	product["pcId"] = 2
	normalized, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	device := normalized["gateway"].(map[string]any)["roomList"].([]any)[0].(map[string]any)["deviceList"].([]any)[0].(map[string]any)
	pid, _ := lightingDesignIntFromAny(device["pid"])
	extra := device["extraMeta"].(map[string]string)
	if extra["materialCode"] != "1-000002044" || pid != 198666 || device["pcId"] != nil {
		t.Fatalf("device product fields = %#v", device)
	}
}

func TestNormalizeLightingDesignImportPayloadRequiresPublicProductIdentity(t *testing.T) {
	payload := semanticLightingDesignFixture()
	slot := payload["rooms"].([]any)[0].(map[string]any)["deviceSlots"].([]any)[0].(map[string]any)
	slot["product"] = map[string]any{"materialCode": "1-000002044", "pid": 198666, "pcId": 4}
	_, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err == nil || !strings.Contains(err.Error(), "skuCode is required") {
		t.Fatalf("expected public product identity error, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadRejectsInternalHouseMeta(t *testing.T) {
	normalized, err := NormalizeLightingDesignImportPayload("200191", semanticLightingDesignFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	_, err = NormalizeLightingDesignImportPayload("200191", normalized)
	if err == nil || !strings.Contains(err.Error(), "lighting.design.import requires the CLI lighting design model") {
		t.Fatalf("expected internal HouseMeta rejection, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadValidatesReferences(t *testing.T) {
	payload := semanticLightingDesignFixture()
	automation := payload["automations"].([]any)[0].(map[string]any)
	action := automation["actions"].([]any)[0].(map[string]any)
	action["targetKey"] = "gp-missing"
	_, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err == nil || !strings.Contains(err.Error(), "does not match an imported resource") {
		t.Fatalf("expected reference validation error, got %v", err)
	}
}

func TestLightingDesignImportClientUsesMetaImportAndVerifies(t *testing.T) {
	var importBody map[string]any
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/meta/import":
			if err := json.NewDecoder(request.Body).Decode(&importBody); err != nil {
				t.Fatalf("decode import body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"request-1"}`))
		case request.URL.Path == "/apis/iot/v1/meta/status":
			if request.URL.Query().Get("requestKey") != "request-1" {
				t.Fatalf("requestKey=%s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"status":"1","houseId":"200191"}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001},{"id":5002,"name":"黑色格栅灯2","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":6001,"name":"客厅格栅灯组","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":7001,"name":"客厅回家模式"}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":8001,"name":"客厅每天9点"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	normalized, err := NormalizeLightingDesignImportPayload("200191", semanticLightingDesignFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	result, err := NewLightingDesignImportClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), LightingDesignImportRequest{
		HouseID:        "200191",
		Intent:         LightingDesignImportCapability,
		Payload:        normalized,
		VerifyAttempts: 1,
		Credentials: LightingDesignImportCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v calls=%#v", err, calls)
	}
	if result.RequestKey != "request-1" || result.Mode != "house_meta_import" || result.Counts["devices"] != 2 || result.Counts["groups"] != 1 || result.Counts["automations"] != 1 {
		t.Fatalf("result=%#v", result)
	}
	if importBody["gateway"] == nil || importBody["sceneList"] == nil || importBody["automationList"] == nil {
		t.Fatalf("import body=%#v", importBody)
	}
	if result.APICalls != 8 {
		t.Fatalf("apiCalls=%d calls=%#v", result.APICalls, calls)
	}
}

func TestLightingDesignImportClientCreatesNewHomeWhenHouseIDEmpty(t *testing.T) {
	var importBody map[string]any
	var importHouseHeader string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/meta/import":
			importHouseHeader = request.Header.Get("houseId")
			if err := json.NewDecoder(request.Body).Decode(&importBody); err != nil {
				t.Fatalf("decode import body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"request-new-home"}`))
		case request.URL.Path == "/apis/iot/v1/meta/status":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"status":"1","houseId":"200777"}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001},{"id":5002,"name":"黑色格栅灯2","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":6001,"name":"客厅格栅灯组","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":7001,"name":"客厅回家模式"}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":8001,"name":"客厅每天9点"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	normalized, err := NormalizeLightingDesignImportPayload("", semanticLightingDesignFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	result, err := NewLightingDesignImportClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), LightingDesignImportRequest{
		Intent:         LightingDesignImportCapability,
		Payload:        normalized,
		VerifyAttempts: 1,
		Credentials: LightingDesignImportCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.HouseID != "200777" || result.RequestKey != "request-new-home" || result.VerifiedBy != "entity.list" {
		t.Fatalf("result=%#v", result)
	}
	if importHouseHeader != "" {
		t.Fatalf("new-home meta import must not send default houseId header, got %q", importHouseHeader)
	}
	if _, ok := importBody["houseId"]; ok {
		t.Fatalf("new-home meta import body must not contain houseId: %#v", importBody)
	}
}

func semanticLightingDesignFixture() map[string]any {
	return map[string]any{
		"key":         "hm1",
		"name":        "粒粒的美丽家庭",
		"gatewayName": "默认网关",
		"rooms": []any{
			map[string]any{
				"key":  "rm1",
				"name": "客厅",
				"deviceSlots": []any{
					map[string]any{"key": "dv1", "name": "黑色格栅灯1", "product": map[string]any{"skuCode": "1-000002044", "capabilityPid": 198666, "productComponentId": 4, "productName": "P20 明装磁吸格栅灯"}},
					map[string]any{"key": "dv2", "name": "黑色格栅灯2", "product": map[string]any{"skuCode": "1-000002044", "capabilityPid": 198666, "productComponentId": 4, "productName": "P20 明装磁吸格栅灯"}},
				},
				"groups": []any{
					map[string]any{"key": "gp1", "name": "客厅格栅灯组", "groupCategory": "lighting", "groupCapability": "light", "slotKeys": []any{"dv1", "dv2"}},
				},
			},
		},
		"scenes": []any{
			map[string]any{
				"key":  "sc1",
				"name": "客厅回家模式",
				"actions": []any{
					map[string]any{"targetType": "group", "targetKey": "gp1", "targetName": "客厅格栅灯组", "rank": 0, "delay": 0, "set": map[string]any{"power": true, "brightness": 60, "colorTemperature": 3000}},
				},
			},
		},
		"automations": []any{
			map[string]any{
				"key":          "at1",
				"name":         "客厅每天9点",
				"activeWindow": map[string]any{"start": "00:00:00", "end": "23:59:59"},
				"repeat":       "daily",
				"trigger":      map[string]any{"conditionKind": "alarm", "time": "09:00:00"},
				"actions": []any{
					map[string]any{"targetType": "group", "targetKey": "gp1", "targetName": "客厅格栅灯组", "rank": 0, "delay": 0, "set": map[string]any{"power": true}},
				},
			},
		},
	}
}
