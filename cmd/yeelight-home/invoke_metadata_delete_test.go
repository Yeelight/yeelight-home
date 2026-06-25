package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/plan"
)

func TestInvokeMetadataDeleteCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-delete-secret", "client-delete-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-room-delete-plan","locale":"zh-CN","utterance":"删除客厅","intent":"room.delete","parameters":{"name":"客厅"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") || strings.Contains(call, "/w/delete") {
			t.Fatalf("delete should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	confirmation := response["confirmation"].(map[string]any)
	record, ok, err := app.planStore.Load(confirmation["planId"].(string))
	if err != nil || !ok {
		t.Fatalf("Load plan ok=%v err=%v", ok, err)
	}
	if record.Intent != "room.delete" || record.Payload["entityId"] != "room-1" {
		t.Fatalf("record = %#v", record)
	}
}

func TestInvokePlanCommitDeletesMetadataFromStoredPlan(t *testing.T) {
	deleteCalls := 0
	afterDelete := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100":
			if afterDelete {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"餐桌灯组","roomId":"room-1"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/group-1/w/info":
			if request.Method != http.MethodDelete {
				http.NotFound(writer, request)
				return
			}
			deleteCalls++
			afterDelete = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			if strings.Contains(request.URL.Path, "ignored") {
				t.Fatalf("commit request payload leaked into API path: %s", request.URL.Path)
			}
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-delete-secret", "client-delete-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-group-delete-plan","locale":"zh-CN","utterance":"删除餐桌灯组","intent":"group.delete","parameters":{"groupId":"group-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	planID := decodeInvokeResponse(t, stdout.Bytes())["confirmation"].(map[string]any)["planId"].(string)

	stdout.Reset()
	stderr.Reset()
	commit := `{"contractVersion":"1.0","requestId":"req-group-delete-commit","locale":"zh-CN","utterance":"确认删除","intent":"plan.commit","parameters":{"planId":"` + planID + `","groupId":"ignored"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commit), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d", deleteCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "metadata-delete-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "group.delete" || result["verified"] != true || result["verifiedBy"] != "entity.list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeMetadataBatchDeleteCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"},{"id":"room-2","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-delete-secret", "client-batch-delete-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-room-batch-delete-plan","locale":"zh-CN","utterance":"删除客厅和卧室","intent":"room.batch_delete","parameters":{"items":[{"roomId":"room-1"},{"name":"卧室"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") {
			t.Fatalf("batch delete should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	preview := response["confirmation"].(map[string]any)["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if len(preview["deleteTargets"].([]any)) != 2 {
		t.Fatalf("preview = %#v", preview)
	}
	record, ok, err := app.planStore.Load(response["confirmation"].(map[string]any)["planId"].(string))
	if err != nil || !ok || record.Intent != "room.batch_delete" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokePlanCommitBatchDeletesMetadataFromStoredPlan(t *testing.T) {
	deleted := map[string]bool{}
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			if deleted["scene-1"] && deleted["scene-2"] {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			rows := `[{"id":"scene-1","name":"回家"},{"id":"scene-2","name":"晚安"}]`
			if deleted["scene-1"] {
				rows = `[{"id":"scene-2","name":"晚安"}]`
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":` + rows + `}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/scene-1/w/info":
			if request.Method != http.MethodDelete {
				http.NotFound(writer, request)
				return
			}
			deleteCalls++
			deleted["scene-1"] = true
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/scene-2/w/info":
			if request.Method != http.MethodDelete {
				http.NotFound(writer, request)
				return
			}
			deleteCalls++
			deleted["scene-2"] = true
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			if strings.Contains(request.URL.Path, "ignored") {
				t.Fatalf("commit request payload leaked into API path: %s", request.URL.Path)
			}
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-delete-secret", "client-batch-delete-1", "house-1")
	planID := createMetadataBatchDeletePlanForTest(t, app, "house-1", "scene.batch_delete", []any{
		map[string]any{"entityId": "scene-1", "sceneId": "scene-1", "name": "回家"},
		map[string]any{"entityId": "scene-2", "sceneId": "scene-2", "name": "晚安"},
	})

	input := `{"contractVersion":"1.0","requestId":"req-scene-batch-delete-commit","locale":"zh-CN","utterance":"确认批量删除","intent":"plan.commit","parameters":{"planId":"` + planID + `","items":[{"sceneId":"ignored"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if deleteCalls != 2 {
		t.Fatalf("deleteCalls = %d", deleteCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "metadata-batch-delete-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["itemCount"] != float64(2) || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func createMetadataBatchDeletePlanForTest(t *testing.T, app *app, houseID string, intent string, items []any) string {
	t.Helper()
	record, err := plan.NewRecord("default", "dev", houseID, intent, "req-plan", intent+" test", map[string]any{
		"houseId":    houseID,
		"entityType": strings.TrimSuffix(intent, ".batch_delete"),
		"items":      items,
	}, []string{"test precondition"}, time.Now(), pendingPlanTTL)
	if err != nil {
		t.Fatalf("NewRecord error: %v", err)
	}
	if err := app.planStore.Save(record); err != nil {
		t.Fatalf("Save plan error: %v", err)
	}
	return record.ID
}
