package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeAutomationUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeAutomationUpdateSeedList(writer, request, "回家开灯", "1")
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-update-secret", "client-auto-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-auto-update-plan","locale":"zh-CN","utterance":"把回家开灯自动化改成18点触发","intent":"automation.update","parameters":{"houseId":"200171","automationId":"auto-1","name":"回家开灯更新","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","params":{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]},"actions":[{"typeId":2,"resId":"50018330","rank":0,"params":{"set":{"power":true}}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/automations/auto-1/w/update") {
			t.Fatalf("automation.update dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "token-auto-update-secret") || strings.Contains(stderr.String(), "token-auto-update-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	previewContainer := result["preview"].(map[string]any)
	preview := previewContainer["payloadPreview"].(map[string]any)
	if preview["automationId"] != "auto-1" || preview["name"] != "回家开灯更新" {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestInvokeAutomationUpdateRejectsStatusField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeAutomationUpdateSeedList(writer, request, "回家开灯", "1")
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-update-secret", "client-auto-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-auto-update-status","locale":"zh-CN","utterance":"更新自动化并启用","intent":"automation.update","parameters":{"houseId":"200171","automationId":"auto-1","name":"回家开灯","status":0,"startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"params":{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]},"actions":[{"typeId":2,"resId":"50018330","rank":0,"params":{"set":{"power":true}}}]}}`
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
	if clarification["reason"] != "automation_status_update_requires_enable_disable_intent" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeAutomationUpdateRequiresKnownAutomation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeAutomationUpdateSeedList(writer, request, "回家开灯", "1")
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-update-secret", "client-auto-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-auto-update-missing","locale":"zh-CN","utterance":"更新不存在的自动化","intent":"automation.update","parameters":{"houseId":"200171","automationId":"auto-missing","name":"不存在","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"params":{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]},"actions":[{"typeId":2,"resId":"50018330","rank":0,"params":{"set":{"power":true}}}]}}`
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
	if clarification["reason"] != "invalid_automation_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeAutomationUpdateExecutesDirectly(t *testing.T) {
	automationListCalls := 0
	var gotCalls []string
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/automations/auto-1/w/update":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode automation update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		case "/apis/iot/v1/automations/r/list":
			automationListCalls++
			name := "回家开灯"
			if automationListCalls > 1 {
				name = "回家开灯更新"
			}
			writeAutomationUpdateSeedList(writer, request, name, "1")
		default:
			writeAutomationUpdateSeedList(writer, request, "回家开灯", "1")
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-update-secret", "client-auto-update-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-auto-update-execute","locale":"zh-CN","utterance":"把回家开灯自动化改成18点触发","intent":"automation.update","parameters":{"houseId":"200171","automationId":"auto-1","name":"回家开灯更新","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","params":{"type":"and","conditions":[{"type":"alarm","clock":"18:00:00"}]},"actions":[{"typeId":2,"resId":"50018330","rank":0,"params":{"set":{"power":true}}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	updateCalls := 0
	for _, call := range gotCalls {
		if call == "PUT /apis/iot/v1/automations/auto-1/w/update" {
			updateCalls++
		}
	}
	if updateCalls != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if writeBody["automationId"] != nil || writeBody["id"] != "auto-1" || writeBody["name"] != "回家开灯更新" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "automation-update-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["automationId"] != "auto-1" || result["name"] != "回家开灯更新" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func writeAutomationUpdateSeedList(writer http.ResponseWriter, request *http.Request, automationName string, automationStatus string) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","houseId":"200171","roomId":"401391"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	case "/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"` + automationName + `","houseId":"200171","status":` + automationStatus + `}]}}`))
	default:
		http.NotFound(writer, request)
	}
}
