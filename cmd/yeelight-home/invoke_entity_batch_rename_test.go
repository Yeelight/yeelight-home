package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeEntityRenameBatchCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-rename-batch-secret", "client-rename-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-rename-batch-plan","locale":"zh-CN","utterance":"把主灯改名为阅读主灯，把已有情景改名为睡前晚安","intent":"entity.rename.batch","parameters":{"houseId":"200171","items":[{"entityType":"device","id":"50018330","newName":"阅读主灯"},{"entityType":"scene","currentName":"已有情景","newName":"睡前晚安"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/name/w/modify") {
			t.Fatalf("entity.rename.batch should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "entity.rename.batch" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	items := record.Payload["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["name"] != "阅读主灯" || items[1].(map[string]any)["id"] != "700001" {
		t.Fatalf("items = %#v", items)
	}
}

func TestInvokePlanCommitEntityRenameBatchFromStoredPlan(t *testing.T) {
	var writeBody []any
	deviceListCalls := 0
	sceneListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"阅读主灯"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			sceneListCalls++
			if sceneListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"700001","name":"已有情景"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"700001","name":"睡前晚安"}]}}`))
		case "/apis/iot/v1/ai/200171/name/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode rename body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-rename-batch-secret", "client-rename-batch-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "entity.rename.batch", map[string]any{
		"houseId": float64(200171),
		"items": []any{
			map[string]any{"id": "50018330", "typeId": 2, "name": "阅读主灯"},
			map[string]any{"id": "700001", "typeId": 6, "name": "睡前晚安"},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-rename-batch-commit","locale":"zh-CN","utterance":"确认批量改名","intent":"plan.commit","parameters":{"planId":"` + planID + `","items":[{"id":"ignored","name":"ignored"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(writeBody) != 2 || writeBody[0].(map[string]any)["name"] != "阅读主灯" || writeBody[0].(map[string]any)["id"] != float64(50018330) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "entity-batch-rename-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "entity.rename.batch" || result["itemCount"] != float64(2) || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeEntityRenameBatchRejectsUnsupportedRoomRename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-rename-batch-secret", "client-rename-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-rename-room-batch","locale":"zh-CN","utterance":"批量改房间名","intent":"entity.rename.batch","parameters":{"houseId":"200171","items":[{"entityType":"room","id":"401398","newName":"会客厅"}]}}`
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
	reason := response["clarification"].(map[string]any)["reason"]
	if reason != "invalid_entity_rename_resource_type" {
		t.Fatalf("reason = %#v", reason)
	}
}
