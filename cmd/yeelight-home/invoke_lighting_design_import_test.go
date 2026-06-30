package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeLightingDesignImportInvalidNaturalPayloadReturnsHouseMetaGuide(t *testing.T) {
	t.Setenv("YEELIGHT_API_BASE_URL", "http://127.0.0.1:1/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-design-invalid","locale":"zh-CN","utterance":"帮我导入一个照明设计","intent":"lighting.design.import","parameters":{"houseId":"200191","rooms":[{"items":[{"name":"吸顶灯"}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response=%#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_lighting_design_import_payload" || clarification["payloadShape"] == nil || clarification["examples"] == nil {
		t.Fatalf("clarification=%#v", clarification)
	}
	acceptedFields := clarification["acceptedFields"].([]any)
	for _, field := range []string{
		"parameters.gateway.roomList",
		"parameters.gateway.roomList[].deviceList",
		"parameters.gateway.roomList[].groupList",
		"parameters.sceneList",
		"parameters.automationList",
	} {
		if !containsAnyString(acceptedFields, field) {
			t.Fatalf("acceptedFields should expose HouseMeta field %s: %#v", field, acceptedFields)
		}
	}
	shape := clarification["payloadShape"].(map[string]any)
	if shape["gateway"] == nil || shape["sceneList"] == nil || shape["automationList"] == nil || shape["shortKeyCompatibility"] == nil {
		t.Fatalf("lighting design guide should expose HouseMeta nested contract: %#v", clarification)
	}
	if _, ok := shape["rooms"]; ok {
		t.Fatalf("legacy natural rooms[] must not be advertised: %#v", shape)
	}
	if !strings.Contains(requestString(clarification["nextStep"]), "HouseMeta") {
		t.Fatalf("clarification nextStep=%#v", clarification["nextStep"])
	}
}

func TestInvokeDeviceSlotCreateDryRunPreviewsHouseMetaWithoutWriting(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := skillRequestJSON("req-slot-plan", "device.slot.create", "200191", houseMetaFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	preview := response["result"].(map[string]any)["preview"].(map[string]any)
	if preview["intent"] != "device.slot.create" {
		t.Fatalf("preview=%#v", preview)
	}
	semanticPreview := preview["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	counts := semanticPreview["counts"].(map[string]any)
	if counts["rooms"] != float64(1) || counts["devices"] != float64(2) || counts["groups"] != float64(1) {
		t.Fatalf("counts=%#v", counts)
	}
	productResolution := semanticPreview["productResolution"].(map[string]any)
	if productResolution["matchedDeviceSlots"] != float64(2) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func TestInvokeLightingDesignImportDryRunResolvesNamedHomeOverProfileDefault(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-home-name","locale":"zh-CN","utterance":"帮我设计一下粒粒的美丽家庭这个家庭，客厅两个黑色格栅灯","intent":"lighting.design.import","homeRef":{"name":"粒粒的美丽家庭"},"parameters":` + houseMetaFixtureJSON() + `}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	payloadPreview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview["houseId"] != "200191" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
}

func TestInvokeLightingDesignImportExecutesViaMetaImport(t *testing.T) {
	var importBody map[string]any
	server := newLightingDesignInvokeServer(t, &importBody)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := skillRequestJSON("req-design-execute", "lighting.design.import", "200191", houseMetaFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if importBody == nil {
		t.Fatalf("meta import body not captured")
	}
	if _, ok := importBody["rooms"]; ok {
		t.Fatalf("legacy natural rooms[] must not reach cloud: %#v", importBody)
	}
	if importBody["gateway"] == nil || importBody["sceneList"] == nil || importBody["automationList"] == nil {
		t.Fatalf("import body missing HouseMeta sections: %#v", importBody)
	}
	gateway := importBody["gateway"].(map[string]any)
	room := gateway["roomList"].([]any)[0].(map[string]any)
	group := room["groupList"].([]any)[0].(map[string]any)
	if group["componentId"] != float64(4) {
		t.Fatalf("group=%#v", group)
	}
	sceneDetail := importBody["sceneList"].([]any)[0].(map[string]any)["details"].([]any)[0].(map[string]any)
	if sceneDetail["typeId"] != float64(4) || sceneDetail["tempId"] != "gp1" || sceneDetail["params"] != `{"delay":0,"set":{"ct":3000,"l":60,"p":true}}` {
		t.Fatalf("scene detail=%#v", sceneDetail)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "lighting-design-import-execute" {
		t.Fatalf("response=%#v", response)
	}
	result := response["result"].(map[string]any)
	if result["requestKey"] != "request-1" || result["verifiedBy"] != "entity.list" {
		t.Fatalf("result=%#v", result)
	}
}

func newLightingDesignInvokeServer(t *testing.T, importBody *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200171","houseName":"默认家庭"},{"houseId":"200191","houseName":"粒粒的美丽家庭"}]}}`))
		case request.URL.Path == "/apis/iot/v1/meta/import":
			if importBody == nil {
				t.Fatalf("dry-run should not write meta import")
			}
			if err := json.NewDecoder(request.Body).Decode(importBody); err != nil {
				t.Fatalf("decode meta import body: %v", err)
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
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("legacy design sync path must not be called")
		default:
			http.NotFound(writer, request)
		}
	}))
}

func skillRequestJSON(requestID string, intent string, houseID string, parameters string) string {
	return `{"contractVersion":"1.0","requestId":"` + requestID + `","locale":"zh-CN","utterance":"照明设计导入","intent":"` + intent + `","parameters":{"houseId":"` + houseID + `",` + strings.TrimPrefix(parameters, "{") + `}`
}

func houseMetaFixtureJSON() string {
	return `{
		"name":"粒粒的美丽家庭",
		"gateway":{
			"tempId":"gw1",
			"name":"默认网关",
			"roomList":[
				{
					"tempId":"rm1",
					"name":"客厅",
					"deviceList":[
						{"tempId":"dv1","name":"黑色格栅灯1","pid":198666,"materialCode":"1-000002044","productName":"P20 明装磁吸格栅灯"},
						{"tempId":"dv2","name":"黑色格栅灯2","pid":198666,"materialCode":"1-000002044","productName":"P20 明装磁吸格栅灯"}
					],
					"groupList":[
						{"tempId":"gp1","name":"客厅格栅灯组","componentId":4,"deviceTempIdList":["dv1","dv2"]}
					]
				}
			]
		},
		"sceneList":[
			{"tempId":"sc1","name":"客厅回家模式","details":[{"typeId":4,"tempId":"gp1","resName":"客厅格栅灯组","rank":0,"params":{"delay":0,"set":{"p":true,"l":60,"ct":3000}}}]}
		],
		"automationList":[
			{"tempId":"at1","name":"客厅每天9点","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","params":{"type":"and","conditions":[{"type":"alarm","clock":"09:00:00"}]},"actions":[{"typeId":4,"tempId":"gp1","resName":"客厅格栅灯组","rank":0,"params":{"delay":0,"set":{"p":true}}}]}
		]
	}`
}
