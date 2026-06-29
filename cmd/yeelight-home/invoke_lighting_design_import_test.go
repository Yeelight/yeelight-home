package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceSlotCreateDryRunPreviewsWithoutWriting(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/room/r/info/"),
			strings.Contains(request.URL.Path, "/device/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("device.slot.create dry-run should not write")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-slot-plan","locale":"zh-CN","utterance":"先给客厅预留两个黑色格栅灯槽位","intent":"device.slot.create","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯","quantity":2,"category":"格栅灯","color":"黑色"}]}]}}`
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
	if len(calls) != 6 {
		t.Fatalf("calls=%#v", calls)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
	payloadPreview := preview["payloadPreview"].(map[string]any)
	semanticPreview := payloadPreview["semanticPreview"].(map[string]any)
	counts := semanticPreview["counts"].(map[string]any)
	if counts["devices"] != float64(2) {
		t.Fatalf("counts=%#v", counts)
	}
}

func TestInvokeLightingDesignImportInvalidPayloadReturnsPayloadGuide(t *testing.T) {
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
	shape := clarification["payloadShape"].(map[string]any)
	if shape["sceneActionContract"] == nil || shape["automationContract"] == nil {
		t.Fatalf("lighting design guide should expose nested scene/automation contracts: %#v", clarification)
	}
	normalized := shape["normalizedAlternative"].(map[string]any)
	normalizedDevices := normalized["devices"].([]any)
	deviceShape := normalizedDevices[0].(map[string]any)
	if deviceShape["attrs"] == nil || deviceShape["gatewayDeviceId"] == nil || deviceShape["roomId"] == nil {
		t.Fatalf("lighting design normalized device shape incomplete: %#v", deviceShape)
	}
	automationContract := shape["automationContract"].(map[string]any)
	automationParams := automationContract["params"].(map[string]any)
	if automationParams["conditions"] == nil || automationContract["actions"] == nil {
		t.Fatalf("lighting design automation contract incomplete: %#v", automationContract)
	}
	rooms := shape["rooms"].([]any)
	items := rooms[0].(map[string]any)["items"].([]any)
	itemShape := items[0].(map[string]any)
	if itemShape["materialCode"] == nil || itemShape["namePattern"] == nil || itemShape["groupKey"] == nil {
		t.Fatalf("lighting design item shape incomplete: %#v", itemShape)
	}
	if !strings.Contains(requestString(clarification["nextStep"]), "complete topology") {
		t.Fatalf("clarification nextStep=%#v", clarification["nextStep"])
	}
}

func TestInvokeDeviceSlotCreateExecutesDirectly(t *testing.T) {
	var syncBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/design/syncMetadata":
			if err := json.NewDecoder(request.Body).Decode(&syncBody); err != nil {
				t.Fatalf("decode sync body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceLocalIdToCloudSlotIds":{"1002":5001,"1003":5002}}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001},{"id":5002,"name":"黑色格栅灯2","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-slot-execute","locale":"zh-CN","utterance":"给客厅预留两个黑色格栅灯槽位","intent":"device.slot.create","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯","quantity":2,"category":"格栅灯","color":"黑色"}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	devices := syncBody["devices"].([]any)
	if len(devices) != 2 {
		t.Fatalf("sync devices=%#v", devices)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "lighting-design-import-execute" {
		t.Fatalf("response=%#v", response)
	}
}

func TestInvokeLightingDesignImportDryRunPreservesSelectedProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.URL.Path, "/area/r/info/"),
			strings.Contains(request.URL.Path, "/room/r/info/"),
			strings.Contains(request.URL.Path, "/device/r/info/"),
			strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("lighting.design.import dry-run should not write")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-design-product-plan","locale":"zh-CN","utterance":"主卧预留四个36度射灯槽位","intent":"lighting.design.import","parameters":{"houseId":"200191","rooms":[{"name":"主卧","items":[{"name":"36°射灯","quantity":4,"materialCode":"1-000004714","notes":"Skill按主卧重点照明选定S系列75开孔36度15w候选"}]}]}}`
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
	semanticPreview := preview["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	productResolution := semanticPreview["productResolution"].(map[string]any)
	if productResolution["matchedDeviceSlots"] != float64(4) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
	if app.preparedOperation != nil {
		t.Fatalf("dry-run must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func TestInvokeLightingDesignImportDryRunResolvesNamedHomeOverProfileDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200171","houseName":"默认家庭"},{"houseId":"200191","houseName":"粒粒的美丽家庭"}]}}`))
		case strings.Contains(request.URL.Path, "/thing/manage/house/200191/area/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/room/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/device/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/group/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("lighting.design.import dry-run should not write")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-design-home-name","locale":"zh-CN","utterance":"帮我设计一下粒粒的美丽家庭这个家庭，客厅一个吸顶灯","intent":"lighting.design.import","homeRef":{"name":"粒粒的美丽家庭"},"parameters":{"rooms":[{"name":"客厅","items":[{"name":"吸顶灯","quantity":1}]}]}}`
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
	payloadPreview := preview["payloadPreview"].(map[string]any)
	if payloadPreview["houseId"] != "200191" {
		t.Fatalf("payloadPreview=%#v", payloadPreview)
	}
}

func TestInvokeLightingDesignImportDryRunAcceptsFullNaturalDesignPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"200191","houseName":"粒粒的美丽家庭"}]}}`))
		case strings.Contains(request.URL.Path, "/thing/manage/house/200191/area/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/room/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/device/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/group/r/info/"),
			strings.Contains(request.URL.Path, "/thing/manage/house/200191/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case strings.Contains(request.URL.Path, "/design/syncMetadata"):
			t.Fatalf("lighting.design.import dry-run should not write")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200171")

	input := `{
		"contractVersion":"1.0",
		"requestId":"lighting-design-import-natural-full",
		"locale":"zh-CN",
		"utterance":"帮我设计一下粒粒的美丽家庭这个家庭，客厅一个吸顶灯，2个黑色格栅灯，2个白色嵌入式射灯，主卧每天9点亮起来。",
		"intent":"lighting.design.import",
		"homeRef":{"name":"粒粒的美丽家庭"},
		"parameters":{
			"rooms":[
				{"name":"客厅","items":[
					{"name":"吸顶灯","quantity":1,"category":"吸顶灯","materialCode":"1-000000031","pid":198666,"pcId":4,"productName":"Yeelight Pro M20 吸顶灯 C450"},
					{"name":"黑色格栅灯","quantity":2,"category":"格栅灯","color":"黑色","materialCode":"1-000002044","pid":198666,"pcId":4,"productName":"Yeelight Pro E20 嵌入式格栅灯-5头"},
					{"name":"白色嵌入式射灯","quantity":2,"category":"筒射灯","color":"白色","installStyle":"嵌入式","materialCode":"1-000001247","pid":198661,"pcId":4,"productName":"Yeelight Pro M20 嵌入式射灯-3寸"}
				]},
				{"name":"主卧","items":[
					{"name":"方形吸顶灯","quantity":1,"category":"吸顶灯","shape":"方形","materialCode":"1-000000031","pid":198666,"pcId":4},
					{"name":"36°射灯","quantity":4,"category":"射灯","beamAngle":"36°","materialCode":"1-000005105","pid":198666,"pcId":4}
				]}
			],
			"groups":[
				{"name":"客厅格栅灯组","roomName":"客厅","match":{"category":"格栅灯"}},
				{"name":"客厅嵌入式射灯组","roomName":"客厅","match":{"name":"白色嵌入式射灯"}},
				{"name":"主卧36°射灯组","roomName":"主卧","match":{"name":"36°射灯"}}
			],
			"scenes":[
				{"name":"客厅离家模式","roomName":"客厅","description":"关闭客厅照明","actions":[{"target":"客厅","type":"light.power","value":"off"}]},
				{"name":"客厅回家模式","roomName":"客厅","description":"打开客厅舒适照明","actions":[{"target":"客厅吸顶灯","type":"light.power","value":"on"}]}
			],
			"automations":[
				{"name":"主卧灯每天9点亮起来","roomName":"主卧","trigger":{"type":"schedule","time":"09:00","repeat":"daily"},"actions":[{"target":"主卧","type":"light.power","value":"on"}],"enabled":true}
			]
		}
	}`
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
	payloadPreview := preview["payloadPreview"].(map[string]any)
	semanticPreview := payloadPreview["semanticPreview"].(map[string]any)
	counts := semanticPreview["counts"].(map[string]any)
	if counts["rooms"] != float64(2) || counts["devices"] != float64(10) || counts["groups"] != float64(3) || counts["scenes"] != float64(2) || counts["automations"] != float64(1) {
		t.Fatalf("counts=%#v", counts)
	}
	productResolution := semanticPreview["productResolution"].(map[string]any)
	if productResolution["matchedDeviceSlots"] != float64(10) {
		t.Fatalf("productResolution=%#v", productResolution)
	}
}

func TestInvokeLightingDesignImportExecutesDirectly(t *testing.T) {
	var syncBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/design/syncMetadata":
			if err := json.NewDecoder(request.Body).Decode(&syncBody); err != nil {
				t.Fatalf("decode sync body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceLocalIdToCloudSlotIds":{"1002":5001}}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"),
			strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-secret", "client-lighting-design-import", "200191")

	input := `{"contractVersion":"1.0","requestId":"req-slot-execute","locale":"zh-CN","utterance":"创建客厅黑色格栅灯槽位","intent":"lighting.design.import","parameters":{"houseId":"200191","rooms":[{"name":"客厅","items":[{"name":"黑色格栅灯1","quantity":1}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response=%#v", response)
	}
	if syncBody["rooms"].([]any)[0].(map[string]any)["localName"] != "客厅" {
		t.Fatalf("direct import used wrong payload: %#v", syncBody)
	}
}
