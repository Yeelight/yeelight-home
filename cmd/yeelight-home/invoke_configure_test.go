package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func TestInvokeRoomCreateCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-plan","locale":"zh-CN","utterance":"创建一个书房","intent":"room.create","parameters":{"houseId":"200171","name":"书房"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-configure-secret") || strings.Contains(stderr.String(), "token-configure-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/w/create") {
			t.Fatalf("room.create should not write before plan.commit: %#v", gotCalls)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "confirmation_required" || response["traceId"] != "pending-plan-created" {
		t.Fatalf("response = %#v", response)
	}
	confirmation, ok := response["confirmation"].(map[string]any)
	if !ok || confirmation["planId"] == "" || confirmation["commitIntent"] != "plan.commit" {
		t.Fatalf("confirmation = %#v", response["confirmation"])
	}
	planID := confirmation["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok {
		t.Fatalf("Load plan error=%v ok=%v", err, ok)
	}
	if record.Intent != "room.create" || record.HouseID != "200171" || record.Payload["name"] != "书房" {
		t.Fatalf("record = %#v", record)
	}
}

func TestInvokePlanCommitCreatesRoomFromStoredPlan(t *testing.T) {
	var createBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-created","name":"书房"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"room-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	planID := createRoomPlanForTest(t, app, "200171", "书房")

	input := `{"contractVersion":"1.0","requestId":"req-room-commit","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"` + planID + `","name":"ignored"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["name"] != "书房" || createBody["houseId"] != float64(200171) {
		t.Fatalf("createBody = %#v", createBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "room-create-commit" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok || result["roomId"] != "room-created" || result["verified"] != true {
		t.Fatalf("result = %#v", response["result"])
	}
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Status != "committed" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeAreaCreateCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-area-plan","locale":"zh-CN","utterance":"创建一楼区域","intent":"area.create","parameters":{"houseId":"200171","name":"一楼","roomIds":["401398"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/area/w/create") {
			t.Fatalf("area.create should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" || response["traceId"] != "pending-plan-created" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "area.create" || record.Payload["name"] != "一楼" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeGroupCreateRequiresRoomAndComponent(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-group-missing","locale":"zh-CN","utterance":"创建客厅灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "clarification_required" || response["traceId"] != "configure-clarification" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "invalid_group_create_payload" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func TestInvokeGroupCreateCreatesPendingPlan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-plan","locale":"zh-CN","utterance":"创建客厅灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomId":"401398","cid":"7","deviceIds":["50018430"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "group.create" || record.Payload["roomId"] != float64(401398) || record.Payload["cid"] != float64(7) {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeSceneCreateCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-scene-plan","locale":"zh-CN","utterance":"创建回家灯光","intent":"scene.create","parameters":{"houseId":"200171","name":"回家灯光","deviceId":"50018330","deviceName":"主灯","params":{"set":{"p":true}}}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/scene/w/create") {
			t.Fatalf("scene.create should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "scene.create" || record.Payload["name"] != "回家灯光" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	details, ok := record.Payload["details"].([]any)
	if !ok || len(details) != 1 {
		t.Fatalf("details = %#v", record.Payload["details"])
	}
	detail, ok := details[0].(map[string]any)
	if !ok || detail["params"] != `{"set":{"p":true}}` {
		t.Fatalf("detail = %#v", details[0])
	}
}

func TestInvokePlanCommitCreatesAreaFromStoredPlan(t *testing.T) {
	var createBody map[string]any
	areaListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			areaListCalls++
			if areaListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"area-created","name":"一楼"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"area-created"}`))
		default:
			writeEmptyHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	planID := createAreaPlanForTest(t, app, "200171", "一楼")

	input := `{"contractVersion":"1.0","requestId":"req-area-commit","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["name"] != "一楼" || createBody["houseId"] != float64(200171) {
		t.Fatalf("createBody = %#v", createBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "metadata-create-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["entityType"] != "area" || result["entityId"] != "area-created" || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeAutomationCreatePlanCommitIsBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-automation-plan","locale":"zh-CN","utterance":"每天晚上十点关灯","intent":"automation.create","parameters":{"houseId":"200171","name":"每天关灯","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","params":{"type":"and","conditions":[{"type":"timer","clock":"22:00:00"}]},"actions":[{"typeId":2,"resId":"50018330","resName":"主灯","rank":0,"params":"{\"set\":{\"p\":false}}"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-automation-commit","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "blocked" || response["traceId"] != "plan-commit-blocked" {
		t.Fatalf("response = %#v", response)
	}
	errorResult, ok := response["error"].(map[string]any)
	if !ok || errorResult["code"] != "automation_commit_disabled" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func TestInvokePlanCommitRequiresExistingPlan(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-missing-plan","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"plan_missing"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "blocked" {
		t.Fatalf("response = %#v", response)
	}
	errorResult, ok := response["error"].(map[string]any)
	if !ok || errorResult["code"] != "plan_not_found" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func TestInvokePlanCancelMarksPendingPlanCanceled(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	planInput := `{"contractVersion":"1.0","requestId":"req-cancel-plan","locale":"zh-CN","utterance":"记住我喜欢客厅亮度 45","intent":"memory.remember","parameters":{"houseId":"house-1","scopeType":"room","scopeRef":"客厅","preferenceType":"brightness","preferenceValue":"45","evidence":"用户明确说明"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(planInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	cancelInput := `{"contractVersion":"1.0","requestId":"req-cancel-commit","locale":"zh-CN","utterance":"取消计划","intent":"plan.cancel","parameters":{"planId":"` + planID + `","houseId":"house-1"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(cancelInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("cancel exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "plan-cancel-local" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["status"] != "canceled" || result["planId"] != planID {
		t.Fatalf("result = %#v", result)
	}

	stdout.Reset()
	stderr.Reset()
	commitInput := `{"contractVersion":"1.0","requestId":"req-cancel-after-commit","locale":"zh-CN","utterance":"确认执行","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "blocked" {
		t.Fatalf("response = %#v", response)
	}
	errorResult := response["error"].(map[string]any)
	if errorResult["code"] != "plan_not_committable" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func TestInvokeExecutionUndoCancelsPendingPlanOnly(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	planInput := `{"contractVersion":"1.0","requestId":"req-undo-plan","locale":"zh-CN","utterance":"记住我喜欢客厅亮度 35","intent":"memory.remember","parameters":{"houseId":"house-1","scopeType":"room","scopeRef":"客厅","preferenceType":"brightness","preferenceValue":"35","evidence":"用户明确说明"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(planInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	planID := response["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	undoInput := `{"contractVersion":"1.0","requestId":"req-undo-plan-cancel","locale":"zh-CN","utterance":"撤销刚才的计划","intent":"execution.undo","parameters":{"planId":"` + planID + `","houseId":"house-1"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(undoInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("undo exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "execution-undo-plan-cancel" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["status"] != "canceled" || result["undoType"] != "pending_plan_cancel" || result["persistentWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeExecutionUndoRequiresPlanID(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-memory-secret", "client-memory-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-undo-missing-plan","locale":"zh-CN","utterance":"撤销刚才的操作","intent":"execution.undo","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("undo exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "blocked" || response["traceId"] != "plan-cancel-blocked" {
		t.Fatalf("response = %#v", response)
	}
	errorResult := response["error"].(map[string]any)
	if errorResult["code"] != "undo_requires_plan_id" {
		t.Fatalf("error = %#v", response["error"])
	}
}

func writeEmptyHouseScopedListForConfigureTest(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
		"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
		"/apis/iot/v1/automations/r/list":
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	default:
		http.NotFound(writer, request)
	}
}

func decodeInvokeResponse(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	return response
}

func createRoomPlanForTest(t *testing.T, app *app, houseID string, name string) string {
	t.Helper()
	payload, err := api.BuildRoomCreatePayload(houseID, name, "", "")
	if err != nil {
		t.Fatalf("BuildRoomCreatePayload error: %v", err)
	}
	record, err := plan.NewRecord("default", "dev", houseID, "room.create", "req-room-plan-seed", "创建房间 "+name, payload, []string{"test precondition"}, time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}
	return record.ID
}

func createAreaPlanForTest(t *testing.T, app *app, houseID string, name string) string {
	t.Helper()
	payload, err := api.BuildAreaCreatePayload(houseID, name, "", "", "", nil)
	if err != nil {
		t.Fatalf("BuildAreaCreatePayload error: %v", err)
	}
	record, err := plan.NewRecord("default", "dev", houseID, "area.create", "req-area-plan-seed", "创建区域 "+name, payload, []string{"test precondition"}, time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}
	return record.ID
}
