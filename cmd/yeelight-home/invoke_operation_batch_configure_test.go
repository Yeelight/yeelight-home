package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeOperationBatchConfigureCreatesSinglePendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/create") || strings.Contains(call, "/w/update") {
			t.Fatalf("batch configure should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	if confirmation["intent"] != "operation.batch.configure" || confirmation["commitIntent"] != "plan.commit" {
		t.Fatalf("confirmation = %#v", confirmation)
	}
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok || record.Intent != "operation.batch.configure" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	steps := record.Payload["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("steps = %#v", steps)
	}
}

func TestInvokeOperationBatchConfigureCommitUsesStoredPayloadOnly(t *testing.T) {
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
	app := newInvokeTestApp(t, "Bearer token-batch-commit-secret", "client-batch-commit-1", "200171")

	planInput := `{"contractVersion":"1.0","requestId":"req-batch-plan-commit","locale":"zh-CN","utterance":"建书房并把主灯改名","intent":"operation.batch.configure","parameters":{"houseId":"200171","operations":[{"intent":"room.create","parameters":{"name":"书房"}},{"intent":"device.rename","parameters":{"deviceId":"50018330","name":"书房主灯"}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(planInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	planID := decodeInvokeResponse(t, stdout.Bytes())["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-batch-commit","locale":"zh-CN","utterance":"确认批量执行","intent":"plan.commit","parameters":{"planId":"` + planID + `","operations":[{"intent":"room.create","parameters":{"name":"恶意覆盖"}}]}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["name"] != "书房" {
		t.Fatalf("createBody should use stored payload: %#v", createBody)
	}
	if renameBody["name"] != "书房主灯" {
		t.Fatalf("renameBody should use stored payload: %#v", renameBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "operation-batch-configure-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["stepCount"] != float64(2) {
		t.Fatalf("result = %#v", result)
	}
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Status != "committed" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
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
	records, err := app.planStore.List()
	if err != nil || len(records) != 0 {
		t.Fatalf("records = %#v err=%v", records, err)
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
	records, err := app.planStore.List()
	if err != nil || len(records) != 0 {
		t.Fatalf("records = %#v err=%v", records, err)
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
