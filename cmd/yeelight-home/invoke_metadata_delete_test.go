package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeMetadataDeleteDryRunPreviewsWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") || strings.Contains(call, "/w/delete") {
			t.Fatalf("delete dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "room.delete" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeMetadataDeleteExecutesDirectly(t *testing.T) {
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
				t.Fatalf("execution request payload leaked into API path: %s", request.URL.Path)
			}
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-delete-secret", "client-delete-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-group-delete-execute","locale":"zh-CN","utterance":"删除餐桌灯组","intent":"group.delete","parameters":{"houseId":"house-1","groupId":"group-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d", deleteCalls)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "metadata-delete-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "group.delete" || result["verified"] != true || result["verifiedBy"] != "entity.list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeMetadataBatchDeleteDryRunPreviewsWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/info") {
			t.Fatalf("batch delete dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if len(preview["deleteTargets"].([]any)) != 2 || result["dryRun"] != true {
		t.Fatalf("preview = %#v result=%#v", preview, result)
	}
}

func TestInvokeMetadataBatchDeleteExecutesDirectly(t *testing.T) {
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
				t.Fatalf("execution request payload leaked into API path: %s", request.URL.Path)
			}
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-batch-delete-secret", "client-batch-delete-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-scene-batch-delete-execute","locale":"zh-CN","utterance":"批量删除回家和晚安情景","intent":"scene.batch_delete","parameters":{"houseId":"house-1","items":[{"sceneId":"scene-1"},{"sceneId":"scene-2"}]}}`
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
	if response["status"] != "success" || response["traceId"] != "metadata-batch-delete-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["itemCount"] != float64(2) || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}
