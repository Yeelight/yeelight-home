package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeRoomCreateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeEmptyHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-preview","locale":"zh-CN","utterance":"创建一个书房","intent":"room.create","parameters":{"houseId":"200171","name":"书房"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/w/create") {
			t.Fatalf("dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomCreateExecutesDirectlyAndVerifies(t *testing.T) {
	var createBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 4 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-created","name":"书房"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("room create method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"room-created"}`))
		default:
			writeEmptyHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-direct","locale":"zh-CN","utterance":"创建一个书房","intent":"room.create","parameters":{"houseId":"200171","name":"书房"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["name"] != "书房" || createBody["houseId"] != float64(200171) {
		t.Fatalf("createBody = %#v", createBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "room-create-execute" {
		t.Fatalf("response = %#v", response)
	}
	execution := response["execution"].(map[string]any)
	if execution["executionMode"] != "direct" {
		t.Fatalf("execution = %#v", execution)
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
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_group_create_payload" {
		t.Fatalf("clarification = %#v", clarification)
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
