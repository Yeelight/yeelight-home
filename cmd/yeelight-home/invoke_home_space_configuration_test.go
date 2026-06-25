package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeHomeUpdateCreatesPendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/w/modify") {
			t.Fatalf("home.update should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "home.update" || record.Payload["name"] != "新家" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeRoomBatchCreateCreatesPendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/w/batch_create") {
			t.Fatalf("room.batch_create should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	rooms := record.Payload["rooms"].([]any)
	if err != nil || !ok || record.Intent != "room.batch_create" || len(rooms) != 2 {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
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

	input := `{"contractVersion":"1.0","requestId":"req-room-area-missing","locale":"zh-CN","utterance":"把客厅加入不存在的区域","intent":"room.area.configure","parameters":{"houseId":"200171","roomId":"401398","addAreaList":["area-missing"]}}`
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

func TestInvokePlanCommitHomeUpdateUsesStoredPayload(t *testing.T) {
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
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-update-secret", "client-home-update-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "home.update", map[string]any{
		"houseId": float64(200171),
		"name":    "新家",
	})

	input := `{"contractVersion":"1.0","requestId":"req-home-update-commit","locale":"zh-CN","utterance":"确认更新家庭","intent":"plan.commit","parameters":{"planId":"` + planID + `","name":"ignored"}}`
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
	if response["status"] != "success" || response["traceId"] != "home-space-configuration-commit" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitRoomBatchUpdateUsesStoredPayload(t *testing.T) {
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
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "room.batch_update", map[string]any{
		"houseId": float64(200171),
		"rooms": []any{
			map[string]any{"roomId": "401391", "name": "会客厅"},
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-room-batch-update-commit","locale":"zh-CN","utterance":"确认批量更新房间","intent":"plan.commit","parameters":{"planId":"` + planID + `","rooms":[{"roomId":"401391","name":"ignored"}]}}`
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
	if response["status"] != "success" || response["traceId"] != "home-space-configuration-commit" {
		t.Fatalf("response = %#v", response)
	}
}
