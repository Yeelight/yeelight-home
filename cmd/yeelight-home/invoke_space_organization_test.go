package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeRoomRenameCreatesPendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401391/w/update") {
			t.Fatalf("room.rename should not write before plan.commit: %#v", gotCalls)
		}
	}
	if strings.Contains(stdout.String(), "token-space-write-secret") || strings.Contains(stderr.String(), "token-space-write-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "room.rename" || record.Payload["name"] != "影音室" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
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
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification := response["clarification"].(map[string]any)
	if clarification["reason"] != "invalid_target_room_reference" {
		t.Fatalf("clarification = %#v", clarification)
	}
}

func TestInvokeAreaUpdateCreatesPendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/area/300001/w/modify") {
			t.Fatalf("area.update should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "area.update" || record.Payload["name"] != "公共区" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
	preview := response["confirmation"].(map[string]any)["payloadPreview"].(map[string]any)["semanticPreview"].(map[string]any)
	if _, ok := preview["current"]; !ok {
		t.Fatalf("missing current preview: %#v", preview)
	}
}

func TestInvokeRoomUpdateCreatesPendingPlanWithoutWriting(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		writeSeededHouseScopedListForConfigureTest(writer, request)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-room-update-plan","locale":"zh-CN","utterance":"把客厅改成会客厅并设置网关","intent":"room.update","parameters":{"houseId":"200171","roomId":"401398","name":"会客厅","gatewayDeviceId":"50018330","seq":2}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/room/401398/w/update") {
			t.Fatalf("room.update should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	if err != nil || !ok || record.Intent != "room.update" || record.Payload["name"] != "会客厅" || record.Payload["gatewayDeviceId"] != "50018330" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokeDeviceMoveRoomBatchCreatesPendingPlanWithoutWriting(t *testing.T) {
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
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range gotCalls {
		if strings.Contains(call, "/device/room/w/batch-modify") {
			t.Fatalf("device.move_room.batch should not write before plan.commit: %#v", gotCalls)
		}
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "confirmation_required" {
		t.Fatalf("response = %#v", response)
	}
	planID := response["confirmation"].(map[string]any)["planId"].(string)
	record, ok, err := app.planStore.Load(planID)
	items, _ := record.Payload["items"].(map[string]any)
	if err != nil || !ok || record.Intent != "device.move_room.batch" || items["50018330"] != "401398" || items["50018430"] != "401398" {
		t.Fatalf("record = %#v ok=%v err=%v", record, ok, err)
	}
}

func TestInvokePlanCommitMovesDeviceFromStoredPlan(t *testing.T) {
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
		case "/apis/iot/v1/device/50018330/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode device update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-space-write-secret", "client-space-write-1", "200171")
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "device.move", map[string]any{
		"houseId":  float64(200171),
		"deviceId": "50018330",
		"id":       "50018330",
		"roomId":   "401392",
	})

	input := `{"contractVersion":"1.0","requestId":"req-device-move-commit","locale":"zh-CN","utterance":"确认移动","intent":"plan.commit","parameters":{"planId":"` + planID + `","roomId":"ignored"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if writeBody["roomId"] != "401392" || writeBody["id"] != "50018330" || writeBody["deviceId"] != nil {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "space-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["entityType"] != "device" || result["entityId"] != "50018330" || result["roomId"] != "401392" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokePlanCommitUpdatesRoomFromStoredPlan(t *testing.T) {
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
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "room.update", map[string]any{
		"houseId":         float64(200171),
		"roomId":          "401391",
		"id":              "401391",
		"name":            "会客厅",
		"gatewayDeviceId": "gw-1",
	})

	input := `{"contractVersion":"1.0","requestId":"req-room-update-commit","locale":"zh-CN","utterance":"确认更新房间","intent":"plan.commit","parameters":{"planId":"` + planID + `","name":"ignored","gatewayDeviceId":"ignored"}}`
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
	if response["status"] != "success" || response["traceId"] != "space-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitMovesDeviceRoomBatchFromStoredPlan(t *testing.T) {
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
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "device.move_room.batch", map[string]any{
		"houseId": float64(200171),
		"items": map[string]any{
			"50018330": "401392",
			"50018430": "401392",
		},
	})

	input := `{"contractVersion":"1.0","requestId":"req-device-batch-move-commit","locale":"zh-CN","utterance":"确认批量移动","intent":"plan.commit","parameters":{"planId":"` + planID + `","items":[{"deviceId":"ignored","roomId":"ignored"}]}}`
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
	if response["status"] != "success" || response["traceId"] != "space-batch-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["capability"] != "device.move_room.batch" || result["itemCount"] != float64(2) || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokePlanCommitUpdatesGroupFromStoredPlan(t *testing.T) {
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
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "group.update", map[string]any{
		"houseId": float64(200171),
		"groupId": "group-1",
		"id":      "group-1",
		"name":    "主灯组",
		"roomId":  "401392",
	})

	input := `{"contractVersion":"1.0","requestId":"req-group-update-commit","locale":"zh-CN","utterance":"确认更新设备组","intent":"plan.commit","parameters":{"planId":"` + planID + `","roomId":"ignored","name":"ignored"}}`
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
	if response["status"] != "success" || response["traceId"] != "space-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePlanCommitUpdatesGroupNameWithoutRoomFromStoredPlan(t *testing.T) {
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
	planID := createHomeOrganizationPlanForTest(t, app, "200171", "group.update", map[string]any{
		"houseId": float64(200171),
		"groupId": "group-1",
		"id":      "group-1",
		"name":    "主灯组",
	})

	input := `{"contractVersion":"1.0","requestId":"req-group-name-update-commit","locale":"zh-CN","utterance":"确认更新设备组名称","intent":"plan.commit","parameters":{"planId":"` + planID + `","roomId":"ignored","name":"ignored"}}`
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
	if response["status"] != "success" || response["traceId"] != "space-organization-commit" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["entityType"] != "group" || result["entityId"] != "group-1" || result["name"] != "主灯组" || result["roomId"] != "401391" {
		t.Fatalf("result = %#v", result)
	}
}
