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

func TestInvokeHomeUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-space-secret", "client-home-space-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-home-update-plan","locale":"zh-CN","utterance":"把家庭名改成新家","intent":"home.update","parameters":{"houseId":"200171","name":"新家","buildingName":"一号楼"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/modify") {
			t.Fatalf("home.update dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "home.update" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomBatchCreateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-room-batch-secret", "client-room-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-batch-create-plan","locale":"zh-CN","utterance":"批量创建书房和茶室","intent":"room.batch_create","parameters":{"houseId":"200171","rooms":[{"name":"书房"},{"name":"茶室"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/w/batch_create") {
			t.Fatalf("room.batch_create dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	if preview["intent"] != "room.batch_create" || result["dryRun"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomAreaConfigureRejectsUnknownArea(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-room-area-secret", "client-room-area-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-area-missing","locale":"zh-CN","utterance":"把客厅加入不存在的区域","intent":"room.area.configure","parameters":{"houseId":"200171","roomId":"401398","addAreaIds":["area-missing"]}}`
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
	if clarification["reason"] != "invalid_area_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeRoomAreaConfigureResolvesRoomAndAreaNames(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-room-area-secret", "client-room-area-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-area-by-name","locale":"zh-CN","utterance":"把客厅加入南区","intent":"room.area.configure","parameters":{"houseId":"200171","roomName":"客厅","addAreaNames":["南区"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v, calls=%#v", response, gotCalls)
	}
	result := response[semantic.FieldResult].(map[string]any)
	preview := result[semantic.FieldPreview].(map[string]any)
	payloadPreview := preview[semantic.FieldPayloadPreview].(map[string]any)
	semanticPreview := payloadPreview[semantic.FieldSemanticPreview].(map[string]any)
	current := semanticPreview[semantic.FieldCurrent].(map[string]any)
	planned := semanticPreview[semantic.FieldPlanned].(map[string]any)
	addAreaIDs := planned[semantic.FieldAddAreaIDs].([]any)
	if payloadPreview[semantic.FieldRoomID] != "401398" || current[semantic.FieldName] != "客厅" || addAreaIDs[0] != "300001" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v semanticPreview=%#v result=%#v", payloadPreview, semanticPreview, result)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/area/w/") || strings.Contains(call, "/room/w/") {
			t.Fatalf("room.area.configure dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeHomeUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	detailCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/r/info":
			detailCalls++
			name := "旧家"
			if detailCalls > 1 {
				name = "新家"
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"200171","name":"` + name + `"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode home update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-update-secret", "client-home-update-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-home-update-execute","locale":"zh-CN","utterance":"把家庭名改成新家","intent":"home.update","parameters":{"houseId":"200171","name":"新家"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["name"] != "新家" || writeBody["houseId"] != nil || writeBody["id"] != float64(200171) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-space-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeRoomBatchUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"会客厅"}]}}`))
		case "/apis/iot/v1/room/w/batchupdate":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room batch update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-room-batch-secret", "client-room-batch-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-batch-update-execute","locale":"zh-CN","utterance":"把客厅改成会客厅","intent":"room.batch_update","parameters":{"houseId":"200171","rooms":[{"roomId":"401391","name":"会客厅"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	rooms := writeBody["rooms"].([]any)
	first := rooms[0].(map[string]any)
	if first["name"] != "会客厅" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "home-space-configuration-execute" {
		t.Fatalf("response = %#v", response)
	}
}
