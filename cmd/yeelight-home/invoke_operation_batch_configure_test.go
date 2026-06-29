package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeOperationBatchConfigureDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-secret", "client-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-batch-plan","locale":"zh-CN","utterance":"建书房并把主灯改名","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":[{"intent":"room.create","parameters":{"name":"书房"}},{"intent":"device.rename","parameters":{"deviceId":"50018330","name":"书房主灯"}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/create") || strings.Contains(call, "/w/update") {
			t.Fatalf("batch configure dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "operation.batch.configure" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeOperationBatchConfigureExecutesDirectly(t *testing.T) {
	var createBody map[string]any
	var renameBody map[string]any
	roomCreated := false
	deviceRenamed := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/w/create":
			roomCreated = true
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode room create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"room-created"}`))
		case "/apis/iot/v1/device/50018330/w/update":
			deviceRenamed = true
			if err := json.NewDecoder(request.Body).Decode(&renameBody); err != nil {
				t.Fatalf("decode device rename body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			writeBatchConfigureEntityLists(writer, request, roomCreated, deviceRenamed)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-execute-secret", "client-batch-execute-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-batch-plan-execute","locale":"zh-CN","utterance":"建书房并把主灯改名","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":[{"intent":"room.create","parameters":{"name":"书房"}},{"intent":"device.rename","parameters":{"deviceId":"50018330","name":"书房主灯"}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["name"] != "书房" {
		t.Fatalf("createBody = %#v", createBody)
	}
	if renameBody["name"] != "书房主灯" {
		t.Fatalf("renameBody = %#v", renameBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "operation-batch-configure-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["stepCount"] != float64(2) {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeOperationBatchConfigureRejectsStrictDeleteIntent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-delete-secret", "client-batch-delete-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-batch-delete-reject","locale":"zh-CN","utterance":"建房间顺便删除旧情景","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":[{"intent":"room.create","parameters":{"name":"书房"}},{"intent":"scene.delete","parameters":{"sceneId":"700001"}}]}}`
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
	if clarification["reason"] != "operation_batch_contains_strict_or_destructive_intent" {
		t.Fatalf("clarification = %#v", clarification)
	}
	if clarification["payloadShape"] == nil || clarification["examples"] == nil || !strings.Contains(requestString(clarification["nextStep"]), "multiple reversible") {
		t.Fatalf("clarification missing operation batch guide = %#v", clarification)
	}
	if app.preparedOperation != nil {
		t.Fatalf("rejected batch must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func TestInvokeOperationBatchConfigureInvalidPayloadReturnsPayloadGuide(t *testing.T) {
	t.Setenv("YEELIGHT_API_BASE_URL", "http://127.0.0.1:1/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-invalid-secret", "client-batch-invalid-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-batch-invalid","locale":"zh-CN","utterance":"批量配置一下","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":{"intent":"room.create","parameters":{"name":"书房"}}}}`
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
	if clarification["reason"] != "invalid_operation_batch_payload" || clarification["payloadShape"] == nil || clarification["examples"] == nil {
		t.Fatalf("clarification = %#v", clarification)
	}
	if !strings.Contains(requestString(clarification["nextStep"]), "one batch for one user request") {
		t.Fatalf("clarification nextStep = %#v", clarification["nextStep"])
	}
}

func TestInvokeOperationBatchConfigureRejectsAccountScopedHomeCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-home-create-secret", "client-batch-home-create-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-batch-home-create-reject","locale":"zh-CN","utterance":"新建家庭并顺便设计房间","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":[{"intent":"home.create","parameters":{"name":"新家"}},{"intent":"room.create","parameters":{"name":"书房"}}]}}`
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
	if clarification["reason"] != "operation_batch_contains_account_scoped_intent" {
		t.Fatalf("clarification = %#v", clarification)
	}
	if app.preparedOperation != nil {
		t.Fatalf("rejected batch must not retain prepared operation: %#v", app.preparedOperation)
	}
}

func writeBatchConfigureEntityLists(writer http.ResponseWriter, request *http.Request, roomCreated bool, deviceRenamed bool) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"300001","name":"南区"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
		if roomCreated {
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"},{"id":"room-created","name":"书房"}]}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"}]}}`))
	case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
		name := "主灯"
		if deviceRenamed {
			name = "书房主灯"
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"` + name + `","roomId":"401398"},{"id":"50018430","name":"筒灯","roomId":"401398"}]}}`))
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
