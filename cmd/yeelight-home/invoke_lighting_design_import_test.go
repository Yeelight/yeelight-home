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

func TestInvokeLightingDesignImportInvalidNaturalPayloadReturnsDesignModelGuide(t *testing.T) {
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
	if response[semantic.FieldStatus] != "clarification_required" {
		t.Fatalf("response=%#v", response)
	}
	clarification := response[semantic.FieldClarification].(map[string]any)
	if clarification[semantic.FieldReason] != "invalid_lighting_design_import_payload" || clarification[semantic.FieldPayloadShape] == nil || clarification[semantic.FieldExamples] == nil {
		t.Fatalf("clarification=%#v", clarification)
	}
	acceptedFields := clarification[semantic.FieldAcceptedFields].([]any)
	for _, field := range []string{
		semantic.ParameterPath(semantic.FieldRooms),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldDeviceSlots),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldGroups),
		semantic.ParameterPath(semantic.FieldScenes),
		semantic.ParameterPath(semantic.FieldAutomations),
	} {
		if !containsAnyString(acceptedFields, field) {
			t.Fatalf("acceptedFields should expose design model field %s: %#v", field, acceptedFields)
		}
	}
	shape := clarification[semantic.FieldPayloadShape].(map[string]any)
	if shape[semantic.FieldRooms] == nil || shape[semantic.FieldScenes] == nil || shape[semantic.FieldAutomations] == nil {
		t.Fatalf("lighting design guide should expose standard design model: %#v", clarification)
	}
	if _, ok := shape[semantic.FieldGateway]; ok {
		t.Fatalf("internal import payload must not be advertised: %#v", shape)
	}
	if !strings.Contains(requestString(clarification[semantic.FieldNextStep]), "standard lighting design model") {
		t.Fatalf("clarification nextStep=%#v", clarification[semantic.FieldNextStep])
	}
}

func TestInvokeDeviceSlotCreateDryRunPreviewsStandardLightingDesignWithoutWriting(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := skillRequestJSON("req-slot-plan", "device.slot.create", "200191", strings.Replace(houseMetaFixtureJSON(), `"name":"客厅"`, `"name":"书房"`, 1))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	preview := response[semantic.FieldResult].(map[string]any)[semantic.FieldPreview].(map[string]any)
	if preview[semantic.FieldIntent] != "device.slot.create" {
		t.Fatalf("preview=%#v", preview)
	}
	semanticPreview := preview[semantic.FieldPayloadPreview].(map[string]any)[semantic.FieldSemanticPreview].(map[string]any)
	counts := semanticPreview[semantic.FieldCounts].(map[string]any)
	if counts[semantic.FieldRooms] != float64(1) || counts[semantic.FieldDevices] != float64(2) || counts[semantic.FieldGroups] != float64(1) {
		t.Fatalf("counts=%#v", counts)
	}
	productResolution := semanticPreview[semantic.FieldProductResolution].(map[string]any)
	if productResolution[semantic.FieldMatchedDeviceSlots] != float64(2) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func TestInvokeDeviceSlotCreateRejectsExistingRoomNameToAvoidDuplicateRoom(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := skillRequestJSON("req-slot-existing-room", "device.slot.create", "200191", houseMetaFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "clarification_required" {
		t.Fatalf("response=%#v", response)
	}
	clarification := response[semantic.FieldClarification].(map[string]any)
	if clarification[semantic.FieldReason] != "device_slot_create_existing_room_would_duplicate_room" {
		t.Fatalf("clarification=%#v", clarification)
	}
	candidates := clarification[semantic.FieldCandidates].([]any)
	if len(candidates) != 1 {
		t.Fatalf("candidates=%#v", candidates)
	}
}

func TestInvokeLightingDesignImportDryRunNamedNewHomeDoesNotUseProfileDefault(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-home-name","locale":"zh-CN","utterance":"帮我设计添加一个易来新家家庭，客厅两个黑色格栅灯","intent":"lighting.design.import","homeRef":{"name":"易来新家家庭"},"parameters":` + houseMetaFixtureJSON() + `}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	payloadPreview := response[semantic.FieldResult].(map[string]any)[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldScope] != "account" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
	semanticPreview := payloadPreview[semantic.FieldSemanticPreview].(map[string]any)
	if semanticPreview[semantic.FieldTargetMode] != "create_new_home" {
		t.Fatalf("semanticPreview=%#v", semanticPreview)
	}
}

func TestInvokeLightingDesignImportDryRunUseCurrentHomeUsesProfileDefault(t *testing.T) {
	server := newLightingDesignInvokeServer(t, nil)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-use-current","locale":"zh-CN","utterance":"把这套照明设计导入当前家庭","intent":"lighting.design.import","homeRef":{"useCurrent":true},"parameters":` + houseMetaFixtureJSON() + `}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response=%#v", response)
	}
	payloadPreview := response[semantic.FieldResult].(map[string]any)[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldHouseID] != "200171" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
	semanticPreview := payloadPreview[semantic.FieldSemanticPreview].(map[string]any)
	if semanticPreview[semantic.FieldTargetMode] != "import_into_existing_home" {
		t.Fatalf("semanticPreview=%#v", semanticPreview)
	}
}

func TestInvokeLightingDesignImportWithoutExplicitHouseCreatesNewHomeAndSelectsIt(t *testing.T) {
	var importBody map[string]any
	var importHouseHeader string
	server := newLightingDesignNewHomeInvokeServer(t, &importBody, &importHouseHeader)
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-new-home","locale":"zh-CN","utterance":"帮我设计添加一个易来新家家庭，客厅两个黑色格栅灯","intent":"lighting.design.import","parameters":` + houseMetaFixtureJSON() + `}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if importBody == nil {
		t.Fatalf("meta import body not captured")
	}
	if importHouseHeader != "" {
		t.Fatalf("new-home import must not send selected profile houseId header, got %q", importHouseHeader)
	}
	if _, ok := importBody["houseId"]; ok {
		t.Fatalf("new-home import must not contain profile/default houseId: %#v", importBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "lighting-design-import-execute" {
		t.Fatalf("response=%#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldHouseID] != "200777" || result[semantic.FieldSelectedHouseID] != "200777" || result[semantic.FieldTargetMode] != "create_new_home" {
		t.Fatalf("result=%#v", result)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok {
		t.Fatalf("metadata load ok=%v err=%v", ok, err)
	}
	if metadata.HouseID != "200777" || metadata.Region != "dev" {
		t.Fatalf("metadata=%#v", metadata)
	}
}

func TestInvokeDeviceSlotCreateRequiresExistingHouseContext(t *testing.T) {
	t.Setenv("YEELIGHT_API_BASE_URL", "http://127.0.0.1:1/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "")

	input := `{"contractVersion":"1.0","requestId":"req-slot-missing-house","locale":"zh-CN","utterance":"给孩子屋添加一个吸顶灯槽位","intent":"device.slot.create","parameters":` + houseMetaFixtureJSON() + `}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "clarification_required" {
		t.Fatalf("response=%#v", response)
	}
	clarification := response[semantic.FieldClarification].(map[string]any)
	if clarification[semantic.FieldReason] != "missing_house_id_for_device_slot_create" {
		t.Fatalf("clarification=%#v", clarification)
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
		t.Fatalf("import body missing internal import sections: %#v", importBody)
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
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "lighting-design-import-execute" {
		t.Fatalf("response=%#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldRequestKey] != "request-1" || result[semantic.FieldVerifiedBy] != "entity.list" {
		t.Fatalf("result=%#v", result)
	}
}

func TestInvokeLightingDesignImportFailureReturnsPartialReadback(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/meta/import":
			_, _ = writer.Write([]byte(`{"success":true,"data":"request-partial"}`))
		case request.URL.Path == "/apis/iot/v1/meta/status":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"status":"-1","houseId":"200888","reason":"backend imported partially"}}`))
		case request.URL.Path == "/apis/iot/v1/house/r/fuzzy":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200888","houseName":"粒粒的美丽家庭"}]}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":6001,"name":"客厅格栅灯组","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":7001,"name":"客厅回家模式"}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "")

	input := skillRequestJSON("req-design-partial", "lighting.design.import", "", houseMetaFixtureJSON())
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "partial" || response[semantic.FieldTraceID] != "lighting-design-import-partial" {
		t.Fatalf("response=%#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldHouseID] != "200888" || result[semantic.FieldVerified] != false {
		t.Fatalf("result=%#v", result)
	}
	partialState := result[semantic.FieldPartialState].(map[string]any)
	observed := partialState[semantic.FieldObservedCounts].(map[string]any)
	if observed["room"] != float64(1) || observed["device"] != float64(1) || observed["group"] != float64(1) {
		t.Fatalf("partialState=%#v", partialState)
	}
	if response[semantic.FieldError] == nil {
		t.Fatalf("partial response should include sanitized error: %#v", response)
	}
	if len(gotCalls) < 4 {
		t.Fatalf("expected import, status, home search, and entity readback calls: %#v", gotCalls)
	}
}

func newLightingDesignNewHomeInvokeServer(t *testing.T, importBody *map[string]any, importHouseHeader *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/meta/import":
			if importBody == nil {
				t.Fatalf("meta import body target is nil")
			}
			if importHouseHeader != nil {
				*importHouseHeader = request.Header.Get("houseId")
			}
			if err := json.NewDecoder(request.Body).Decode(importBody); err != nil {
				t.Fatalf("decode meta import body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"request-new-home"}`))
		case request.URL.Path == "/apis/iot/v1/meta/status":
			if request.URL.Query().Get("requestKey") != "request-new-home" {
				t.Fatalf("requestKey=%s", request.URL.RawQuery)
			}
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
		"rooms":[
			{
				"key":"rm1",
				"name":"客厅",
				"deviceSlots":[
					{"key":"dv1","name":"黑色格栅灯1","product":{"capabilityPid":198666,"skuCode":"1-000002044","productName":"P20 明装磁吸格栅灯"}},
					{"key":"dv2","name":"黑色格栅灯2","product":{"capabilityPid":198666,"skuCode":"1-000002044","productName":"P20 明装磁吸格栅灯"}}
				],
				"groups":[
					{"key":"gp1","name":"客厅格栅灯组","groupCategory":"lighting","groupCapability":"light","slotKeys":["dv1","dv2"]}
				]
			}
		],
		"scenes":[
			{"key":"sc1","name":"客厅回家模式","actions":[{"targetType":"group","targetKey":"gp1","targetName":"客厅格栅灯组","rank":0,"delay":0,"set":{"power":true,"brightness":60,"colorTemperature":3000}}]}
		],
		"automations":[
			{"key":"at1","name":"客厅每天9点","activeWindow":{"start":"00:00:00","end":"23:59:59"},"repeat":"daily","trigger":{"conditionKind":"alarm","time":"09:00:00"},"actions":[{"targetType":"group","targetKey":"gp1","targetName":"客厅格栅灯组","rank":0,"delay":0,"set":{"power":true}}]}
		]
	}`
}
