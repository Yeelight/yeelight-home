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

func TestInvokeRoomRenameDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-rename-plan","locale":"zh-CN","utterance":"把客厅改成影音室","intent":"room.rename","parameters":{"houseId":"200171","roomId":"401398","name":"影音室"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401391/w/update") {
			t.Fatalf("room.rename dry-run should not write: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "token-space-write-secret") || strings.Contains(stderr.String(), "token-space-write-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	preview := result[semantic.FieldPreview].(map[string]any)
	if preview[semantic.FieldIntent] != "room.rename" || result[semantic.FieldDryRun] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomRenameResolvesCurrentRoomName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-rename-by-name","locale":"zh-CN","utterance":"把客厅改成影音室","intent":"room.rename","parameters":{"houseId":"200171","roomName":"客厅","newName":"影音室"}}`
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
	if payloadPreview[semantic.FieldRoomID] != "401398" || payloadPreview[semantic.FieldName] != "影音室" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401398/w/update") {
			t.Fatalf("room.rename dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeRoomRenameResolvesPhoneticRoomName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-rename-phonetic","locale":"zh-CN","utterance":"把客廷改成影音室","intent":"room.rename","parameters":{"houseId":"200171","roomName":"客廷","newName":"影音室"}}`
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
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldRoomID] != "401398" || payloadPreview[semantic.FieldName] != "影音室" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401398/w/update") {
			t.Fatalf("room.rename dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeDeviceMoveRequiresKnownTargetRoom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-move-missing-room","locale":"zh-CN","utterance":"把主灯移动到不存在的房间","intent":"device.move","parameters":{"houseId":"200171","deviceId":"50018330","roomId":"room-missing"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response[semantic.FieldClarification].(map[string]any)
	if clarification[semantic.FieldReason] != "invalid_target_room_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeAreaUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-area-update-plan","locale":"zh-CN","utterance":"把南区改名为公共区并关联客厅","intent":"area.update","parameters":{"houseId":"200171","areaId":"300001","name":"公共区","roomIds":["401398"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/area/300001/w/modify") {
			t.Fatalf("area.update dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	preview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)[semantic.FieldSemanticPreview].(map[string]any)
	if _, ok := preview[semantic.FieldCurrent]; !ok || result[semantic.FieldDryRun] != true {
		t.Fatalf("preview = %#v result=%#v", preview, result)
	}
}

func TestInvokeAreaUpdateResolvesCurrentAreaName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-area-update-by-name","locale":"zh-CN","utterance":"把南区改名为公共区","intent":"area.update","parameters":{"houseId":"200171","areaName":"南区","name":"公共区"}}`
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
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldAreaID] != "300001" || payloadPreview[semantic.FieldName] != "公共区" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/area/300001/w/modify") {
			t.Fatalf("area.update dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeRoomUpdateDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-update-plan","locale":"zh-CN","utterance":"把客厅改成会客厅并设置网关","intent":"room.update","parameters":{"houseId":"200171","roomId":"401398","name":"会客厅","gatewayDeviceId":"50018330","sequence":2}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401398/w/update") {
			t.Fatalf("room.update dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	preview := result[semantic.FieldPreview].(map[string]any)
	if preview[semantic.FieldIntent] != "room.update" || result[semantic.FieldDryRun] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomUpdateResolvesCurrentRoomName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-update-by-name","locale":"zh-CN","utterance":"给客厅补一个描述","intent":"room.update","parameters":{"houseId":"200171","roomName":"客厅","description":"家庭会客空间"}}`
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
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldRoomID] != "401398" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401398/w/update") {
			t.Fatalf("room.update dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeDeviceRenameResolvesCurrentDeviceName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-rename-by-name","locale":"zh-CN","utterance":"把主灯改名为阅读主灯","intent":"device.rename","parameters":{"houseId":"200171","deviceName":"主灯","newName":"阅读主灯"}}`
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
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "50018330" || payloadPreview[semantic.FieldName] != "阅读主灯" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
}

func TestInvokeDeviceMoveResolvesDeviceAndTargetRoomNames(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401398","name":"客厅"},{"id":"401399","name":"书房"}]}}`))
		default:
			writeSeededHouseScopedListForConfigureTest(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-move-by-name","locale":"zh-CN","utterance":"把主灯移动到书房","intent":"device.move","parameters":{"houseId":"200171","deviceName":"主灯","targetRoomName":"书房"}}`
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
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldDeviceID] != "50018330" || payloadPreview[semantic.FieldRoomID] != "401399" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
}

func TestInvokeGroupUpdateDryRunResolvesNameWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-update-by-name","locale":"zh-CN","utterance":"把已有灯组改名为客厅格栅灯组","intent":"group.update","parameters":{"houseId":"200171","groupName":"已有灯组","name":"客厅格栅灯组","targetRoomName":"客厅"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/group/600001/w/modify") {
			t.Fatalf("group.update dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v, calls=%#v", response, gotCalls)
	}
	result := response[semantic.FieldResult].(map[string]any)
	payloadPreview := result[semantic.FieldPreview].(map[string]any)[semantic.FieldPayloadPreview].(map[string]any)
	if payloadPreview[semantic.FieldGroupID] != "600001" || payloadPreview[semantic.FieldName] != "客厅格栅灯组" || payloadPreview[semantic.FieldRoomID] != "401398" || result[semantic.FieldDryRun] != true {
		t.Fatalf("payloadPreview = %#v result=%#v", payloadPreview, result)
	}
}

func TestInvokeDeviceMoveRoomBatchDryRunPreviewsWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-batch-move-plan","locale":"zh-CN","utterance":"把两盏灯批量移动到客厅","intent":"device.move_room.batch","parameters":{"houseId":"200171","items":[{"deviceId":"50018330","roomId":"401398"},{"deviceId":"50018430","roomId":"401398"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/device/room/w/batch-modify") {
			t.Fatalf("device.move_room.batch dry-run should not write: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	preview := result[semantic.FieldPreview].(map[string]any)
	if preview[semantic.FieldIntent] != "device.move_room.batch" || result[semantic.FieldDryRun] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeDeviceMoveExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	deviceListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401392"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/room/w/batch-modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode device batch move body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-move-execute","locale":"zh-CN","utterance":"把主灯移动到卧室","intent":"device.move","parameters":{"houseId":"200171","deviceId":"50018330","roomId":"401392"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	items := writeBody["items"].(map[string]any)
	if writeBody["houseId"] != float64(200171) || items["50018330"] != "401392" || items["ignored"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "space-batch-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldCapability] != "device.move" || result[semantic.FieldItemCount] != float64(1) || result[semantic.FieldVerified] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeRoomUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gw-1","name":"网关"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"会客厅"}]}}`))
		case "/apis/iot/v1/room/401391/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-update-execute","locale":"zh-CN","utterance":"把客厅改成会客厅并设置网关","intent":"room.update","parameters":{"houseId":"200171","roomId":"401391","name":"会客厅","gatewayDeviceId":"gw-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["roomId"] != nil || writeBody["id"] != "401391" || writeBody["name"] != "会客厅" || writeBody["gatewayDeviceId"] != "gw-1" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "space-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeDeviceMoveRoomBatchResolvesNaturalNames(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-device-batch-name-secret", "client-device-batch-name-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-batch-move-by-name","locale":"zh-CN","utterance":"把主灯批量移动到客厅","intent":"device.move_room.batch","parameters":{"houseId":"200171","items":[{"deviceName":"主灯","targetRoomName":"客厅"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	payloadPreview := preview["payloadPreview"].(map[string]any)
	items := payloadPreview["items"].(map[string]any)
	if items["50018330"] != "401398" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/device/room/w/batch-modify") {
			t.Fatalf("dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeDeviceMoveRoomBatchAcceptsDeviceNamesShortcut(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-device-batch-shortcut-secret", "client-device-batch-shortcut-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-batch-move-by-shortcut","locale":"zh-CN","utterance":"把主灯和筒灯一起移动到客厅","intent":"device.move_room.batch","parameters":{"houseId":"200171","deviceNames":["主灯","筒灯"],"targetRoomName":"客厅"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin", "--dry-run"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "invoke-preview" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	preview := result["preview"].(map[string]any)
	payloadPreview := preview["payloadPreview"].(map[string]any)
	items := payloadPreview["items"].(map[string]any)
	if items["50018330"] != "401398" || items["50018430"] != "401398" {
		t.Fatalf("payloadPreview = %#v", payloadPreview)
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/device/room/w/batch-modify") {
			t.Fatalf("dry-run should not write: %#v", gotCalls)
		}
	}
}

func TestInvokeDeviceMoveRoomBatchExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	deviceListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401391"},{"id":"50018430","name":"筒灯","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401392"},{"id":"50018430","name":"筒灯","roomId":"401392"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/room/w/batch-modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode batch move body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-batch-move-execute","locale":"zh-CN","utterance":"把两盏灯批量移动到卧室","intent":"device.move_room.batch","parameters":{"houseId":"200171","items":[{"deviceId":"50018330","roomId":"401392"},{"deviceId":"50018430","roomId":"401392"}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	items := writeBody["items"].(map[string]any)
	if writeBody["houseId"] != float64(200171) || items["50018330"] != "401392" || items["50018430"] != "401392" || items["ignored"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "space-batch-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldCapability] != "device.move_room.batch" || result[semantic.FieldItemCount] != float64(2) || result[semantic.FieldVerified] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeGroupUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	groupListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			groupListCalls++
			if groupListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"主灯组","roomId":"401392"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode group update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-update-execute","locale":"zh-CN","utterance":"把灯组改名为主灯组并移动到卧室","intent":"group.update","parameters":{"houseId":"200171","groupId":"group-1","name":"主灯组","roomId":"401392"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["groupId"] != nil || writeBody["id"] != "group-1" || writeBody["name"] != "主灯组" || writeBody["roomId"] != "401392" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "space-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeGroupNameUpdateExecutesDirectly(t *testing.T) {
	var writeBody map[string]any
	groupListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			groupListCalls++
			if groupListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"主灯组","roomId":"401391"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/group-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode group update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-name-update-execute","locale":"zh-CN","utterance":"把灯组改名为主灯组","intent":"group.update","parameters":{"houseId":"200171","groupId":"group-1","name":"主灯组"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["groupId"] != nil || writeBody["id"] != "group-1" || writeBody["name"] != "主灯组" || writeBody["roomId"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response[semantic.FieldStatus] != "success" || response[semantic.FieldTraceID] != "space-organization-execute" {
		t.Fatalf("response = %#v", response)
	}
	result := response[semantic.FieldResult].(map[string]any)
	if result[semantic.FieldEntityType] != "group" || result[semantic.FieldEntityID] != "group-1" || result[semantic.FieldName] != "主灯组" || result[semantic.FieldRoomID] != "401391" {
		t.Fatalf("result = %#v", result)
	}
}
