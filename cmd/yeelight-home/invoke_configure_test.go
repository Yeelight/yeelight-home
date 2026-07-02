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
	if execution["executionModel"] != "direct" {
		t.Fatalf("execution = %#v", execution)
	}
}

func TestInvokeRoomCreateReusesWriteVerificationTopologyCacheForNextRead(t *testing.T) {
	roomCreated := false
	listCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/w/create":
			roomCreated = true
			_, _ = writer.Write([]byte(`{"success":true,"data":"room-created"}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			listCalls++
			if request.URL.Path == "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100" && roomCreated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-created","name":"书房"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-cache-secret", "client-configure-cache-1", "200171")

	createInput := `{"contractVersion":"1.0","requestId":"req-room-create-cache","locale":"zh-CN","utterance":"创建一个书房","intent":"room.create","parameters":{"houseId":"200171","name":"书房"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(createInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("create exit code = %d, stderr = %s", code, stderr.String())
	}
	createResponse := decodeInvokeResponse(t, stdout.Bytes())
	createMetrics := createResponse["metrics"].(map[string]any)
	if createMetrics["topologyCacheRefreshApiCalls"] != float64(0) || createMetrics["topologyCacheWriteSource"] != "write_verification" {
		t.Fatalf("create metrics = %#v", createMetrics)
	}
	listCallsAfterCreate := listCalls

	stdout.Reset()
	stderr.Reset()
	getInput := `{"contractVersion":"1.0","requestId":"req-room-get-cache","locale":"zh-CN","utterance":"看看书房","intent":"entity.get","parameters":{"houseId":"200171","entityType":"room","name":"书房"}}`
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(getInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("entity get exit code = %d, stderr = %s", code, stderr.String())
	}
	getResponse := decodeInvokeResponse(t, stdout.Bytes())
	if getResponse["status"] != "success" {
		t.Fatalf("getResponse = %#v", getResponse)
	}
	getMetrics := getResponse["metrics"].(map[string]any)
	if getMetrics[semantic.FieldAPICalls] != float64(0) {
		t.Fatalf("get metrics = %#v", getMetrics)
	}
	if listCalls != listCallsAfterCreate {
		t.Fatalf("entity.get should use refreshed topology cache, before=%d after=%d", listCallsAfterCreate, listCalls)
	}
}

func TestInvokeGroupCreateRequiresRoomAndComponent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeEmptyHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-group-missing","locale":"zh-CN","utterance":"创建客厅灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","groupCapability":"light"}}`
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
	if clarification["reason"] != "invalid_group_room_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeGroupCreateDerivesComponentFromSelectedDevices(t *testing.T) {
	var createBody map[string]any
	groupCreated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018376","name":"RGBW灯1","roomId":"401398"},{"id":"50018377","name":"RGBW灯2","roomId":"401398"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			if groupCreated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"4699","name":"RGBW灯组"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/schema/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018376","name":"RGBW灯1","pid":264193,"pcId":41,"subDevices":[{"cid":5,"name":"color light","category":"light","properties":[{"propId":"p"},{"propId":"l"},{"propId":"ct"},{"propId":"c"}]}]},{"id":"50018377","name":"RGBW灯2","pid":264196,"pcId":41,"subDevices":[{"cid":5,"name":"color light","category":"light","properties":[{"propId":"p"},{"propId":"l"},{"propId":"ct"},{"propId":"c"}]}]}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode group create body: %v", err)
			}
			groupCreated = true
			_, _ = writer.Write([]byte(`{"success":true,"data":"4699"}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-derived-component","locale":"zh-CN","utterance":"把客厅两个 RGBW 灯建成灯组","intent":"group.create","parameters":{"houseId":"200171","name":"RGBW灯组","roomName":"客厅","groupCapability":"light","deviceNames":["RGBW灯1","RGBW灯2"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if createBody["cid"] != float64(5) {
		t.Fatalf("group create should derive cid=5 from device schemas: %#v", createBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "metadata-create-execute" {
		t.Fatalf("response = %#v", response)
	}
	if strings.Contains(stdout.String(), "componentId") || strings.Contains(stdout.String(), `"cid"`) {
		t.Fatalf("public response leaked internal group component fields: %s", stdout.String())
	}
}

func TestInvokeGroupCreateRequiresMemberDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/w/create":
			t.Fatal("group.create should not reach cloud without member devices")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-empty-members","locale":"zh-CN","utterance":"在客厅建一个灯组","intent":"group.create","parameters":{"houseId":"200171","name":"客厅灯组","roomName":"客厅","groupCapability":"light"}}`
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
	if clarification["reason"] != "missing_group_members" {
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
