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

func TestInvokePanelButtonConfigureDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-config-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-config-plan","locale":"zh-CN","utterance":"把客厅面板第一个按键绑定回家情景","intent":"panel.button.configure","targets":[{"entityType":"device","id":"panel-1"}],"parameters":{"houseId":"200171","buttons":[{"id":"btn-1","keyValue":1,"targetId":"scene-1","targetType":"scene","visible":1,"type":2,"accessToken":"must-drop"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/panel/w/button/update/") {
			t.Fatalf("panel.button.configure dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "must-drop") || strings.Contains(stdout.String(), "token-panel-config-secret") {
		t.Fatalf("output leaked sensitive payload: %s", stdout.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "panel.button.configure" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(stdout.String(), `"resId"`) || strings.Contains(stdout.String(), `"resType"`) {
		t.Fatalf("preview leaked internal panel fields: %s", stdout.String())
	}
}

func TestInvokePanelButtonConfigureResolvesNaturalDeviceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-config-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-config-natural","locale":"zh-CN","utterance":"把客廷面板第一个按键叫回家","intent":"panel.button.configure","parameters":{"houseId":"200171","deviceName":"客廷面板","buttons":[{"id":"btn-1","alias":"回家"}]}}`
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
	payloadPreview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "panel-1" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
}

func TestInvokeKnobConfigureRequiresKnownDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-config-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-knob-config-missing","locale":"zh-CN","utterance":"配置不存在的旋钮","intent":"knob.configure","parameters":{"houseId":"200171","deviceId":"missing","actions":[{"id":"detail-1","index":1,"mode":"scene"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_panel_device_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
	if clarification[semantic.FieldPayloadShape] == nil || clarification[semantic.FieldExamples] == nil || !strings.Contains(requestString(clarification[semantic.FieldNextStep]), "knob.get") {
		t.Fatalf("clarification missing knob payload guide = %#v", clarification)
	}
}

func TestInvokePanelButtonEventResetResolvesNaturalDeviceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-reset-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-event-reset-natural","locale":"zh-CN","utterance":"清空客廷面板单击动作","intent":"panel.button_event.reset","parameters":{"houseId":"200171","deviceName":"客廷面板","buttonEventId":"101"}}`
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
	payloadPreview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "panel-1" || payloadPreview[semantic.FieldButtonEventID] != "101" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
}

func TestInvokePanelButtonEventUpdateAcceptsNestedButtonEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-event-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-event-nested","locale":"zh-CN","utterance":"把客廷面板单击绑定回家情景","intent":"panel.button_event.update","parameters":{"houseId":"200171","deviceName":"客廷面板","buttonEvent":{"buttonEventId":"101","alias":"单击","actions":[{"targetType":"scene","targetId":"scene-1","targetName":"回家模式"}]}}}`
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
	payloadPreview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "panel-1" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
	event := payloadPreview[semantic.FieldButtonEvent].(map[string]any)
	if event[semantic.FieldButtonEventID] != "101" || event[semantic.FieldAlias] != "单击" {
		t.Fatalf("event = %#v", event)
	}
}

func TestInvokePanelButtonEventUpdateInvalidPayloadReturnsPayloadGuide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-event-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-event-bad","locale":"zh-CN","utterance":"把面板单击改成回家模式","intent":"panel.button_event.update","parameters":{"houseId":"200171","deviceId":"panel-1","buttonEventId":"101","actions":[]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_panel_button_event_payload" || clarification[semantic.FieldPayloadShape] == nil || clarification[semantic.FieldExamples] == nil {
		t.Fatalf("clarification = %#v", clarification)
	}
	if !strings.Contains(requestString(clarification[semantic.FieldNextStep]), "complete actions list") {
		t.Fatalf("clarification nextStep = %#v", clarification[semantic.FieldNextStep])
	}
}

func TestInvokeKnobConfigureResolvesNaturalDeviceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-knob-config-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-knob-natural","locale":"zh-CN","utterance":"把客廷旋钮第一路绑定回家模式","intent":"knob.configure","parameters":{"houseId":"200171","deviceName":"客廷旋钮","actions":[{"id":"detail-1","index":1,"configType":"scene","targetType":"scene","targetId":"scene-1"}]}}`
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
	payloadPreview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "knob-1" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
	actions := payloadPreview[semantic.FieldActions].([]any)
	action := actions[0].(map[string]any)
	if action[semantic.FieldIndex] != float64(1) || action[semantic.FieldConfigType] != "scene" {
		t.Fatalf("action preview = %#v", action)
	}
}

func TestInvokeKnobConfigureInvalidPayloadReturnsPayloadGuide(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-knob-config-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-knob-bad","locale":"zh-CN","utterance":"把旋钮第一路绑定回家模式","intent":"knob.configure","parameters":{"houseId":"200171","deviceId":"knob-1","actions":[]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_knob_configure_payload" || clarification[semantic.FieldPayloadShape] == nil || clarification[semantic.FieldExamples] == nil {
		t.Fatalf("clarification = %#v", clarification)
	}
	if !strings.Contains(requestString(clarification[semantic.FieldNextStep]), "Preserve the current detail row") {
		t.Fatalf("clarification nextStep = %#v", clarification[semantic.FieldNextStep])
	}
}

func TestInvokeKnobResetPreviewIncludesIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-knob-reset-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-knob-reset-plan","locale":"zh-CN","utterance":"清空旋钮第一个子键绑定","intent":"knob.reset","targets":[{"entityType":"device","id":"panel-1"}],"parameters":{"houseId":"200171","deviceId":"panel-1","index":1}}`
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
	preview := response["result"].(map[string]any)["preview"].(map[string]any)["payloadPreview"].(map[string]any)
	index, ok := requestInt(preview["index"])
	if !ok || index != 1 {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokePanelButtonEventBatchUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writePanelConfigEntityList(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-event-secret", "client-panel-config-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-panel-event-plan","locale":"zh-CN","utterance":"批量更新客厅面板按键动作","intent":"panel.button_event.batch_update","targets":[{"entityType":"device","id":"panel-1"}],"parameters":{"houseId":"200171","buttonEvents":[{"buttonEventId":"101","alias":"单击","actions":[{"targetId":"scene-1","targetType":"scene","accessToken":"must-drop"}]},{"buttonEventId":"102","actions":[{"targetId":"scene-2","targetType":"scene"}]}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/panel/w/button/event/update") {
			t.Fatalf("panel button event update dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "must-drop") || strings.Contains(stdout.String(), "token-panel-event-secret") {
		t.Fatalf("output leaked sensitive payload: %s", stdout.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "panel.button_event.batch_update" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
	if strings.Contains(stdout.String(), `"resId"`) || strings.Contains(stdout.String(), `"typeId"`) || strings.Contains(stdout.String(), `"resType"`) {
		t.Fatalf("preview leaked internal panel event fields: %s", stdout.String())
	}
}

func TestInvokeKnobConfigureExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	knobReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/multi-knob/knob-1/detail":
			knobReadCalls++
			if knobReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"id":"detail-1","index":1,"mode":"old"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"id":"detail-1","index":1,"mode":"scene","alias":"观影旋钮","resId":"scene-1","resName":"观影"}]}}`))
		case "/apis/iot/v1/multi-knob/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writePanelConfigEntityList(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-config-secret", "client-panel-config-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-knob-config-execute","locale":"zh-CN","utterance":"配置旋钮第一个子键为情景","intent":"knob.configure","parameters":{"houseId":"200171","deviceId":"knob-1","actions":[{"id":"detail-1","index":1,"mode":"scene","alias":"观影旋钮","targetType":"scene","targetId":"scene-1","targetName":"观影"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["id"] != "knob-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	details := writeBody[semantic.FieldDetails].([]any)
	detail := details[0].(map[string]any)
	if detail[semantic.FieldAlias] != "观影旋钮" || detail["resName"] != "观影" {
		t.Fatalf("writeBody details = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "panel-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePanelButtonEventUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	buttonReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"panel-1","name":"客厅面板"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1":
			buttonReadCalls++
			if buttonReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"buttonEventId":101,"alias":"old","details":[{"resId":"old","typeId":2}]}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"buttonEventId":101,"alias":"单击","details":[{"resId":"scene-1","typeId":6}]}]}`))
		case "/apis/iot/v1/panel/w/button/event/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writePanelConfigEntityList(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-event-secret", "client-panel-config-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-panel-event-execute","locale":"zh-CN","utterance":"更新面板按键动作","intent":"panel.button_event.update","parameters":{"houseId":"200171","deviceId":"panel-1","buttonEventId":"101","alias":"单击","actions":[{"targetType":"scene","targetId":"scene-1","targetName":"回家模式"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["buttonEventId"] != "101" || writeBody["alias"] != "单击" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	details := writeBody["details"].([]any)
	detail := details[0].(map[string]any)
	if detail["typeId"] != float64(6) || detail["resId"] != "scene-1" || detail["resName"] != "回家模式" {
		t.Fatalf("writeBody details = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "panel-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePanelButtonConfigureExecutesDirectly(t *testing.T) {
	var writeBody []map[string]any
	buttonReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/panel-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"panel-1","name":"客厅面板"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1":
			buttonReadCalls++
			if buttonReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"2":[{"id":"btn-1","deviceId":"panel-1","name":"K1","alias":"K1","keyValue":1,"index":1,"resId":"0","resType":0,"visible":0,"icon":0,"sort":0,"type":2,"extend":""}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"2":[{"id":"btn-1","deviceId":"panel-1","name":"K1","alias":"入口灯","keyValue":1,"index":1,"resId":"0","resType":0,"visible":0,"icon":0,"sort":0,"type":2,"extend":""}]}}`))
		case "/apis/iot/v1/panel/w/button/update/panel-1":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode write body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writePanelConfigEntityList(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-button-secret", "client-panel-config-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-panel-button-execute","locale":"zh-CN","utterance":"修改面板按钮别名","intent":"panel.button.configure","parameters":{"houseId":"200171","deviceId":"panel-1","buttons":[{"id":"btn-1","alias":"入口灯"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 1 || writeBody[0]["alias"] != "入口灯" || writeBody[0]["type"] != float64(2) || writeBody[0]["deviceId"] != "panel-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "panel-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeKnobResetExecutesDirectly(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/multi-knob/knob-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"index":1,"mode":"scene"}]}}`))
		case "/apis/iot/v1/multi-knob/knob-1/1/reset":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writePanelConfigEntityList(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-event-secret", "client-panel-config-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-knob-reset-execute","locale":"zh-CN","utterance":"重置旋钮第一个子键位","intent":"knob.reset","parameters":{"houseId":"200171","deviceId":"knob-1","index":1}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(strings.Join(gotCalls, "\n"), "POST /apis/iot/v1/multi-knob/knob-1/1/reset") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "panel-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func writePanelConfigEntityList(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"panel-1","name":"客厅面板"},{"id":"knob-1","name":"客厅旋钮"}]}}`))
	default:
		http.NotFound(writer, request)
	}
}
