package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvokeGroupMembersUpdateResolvesCompleteDeviceSelection(t *testing.T) {
	var writeBody map[string]any
	updated := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"5001","name":"左灯"},{"id":"5002","name":"旧灯"},{"id":"5003","name":"右灯"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"9001","name":"客厅格栅灯组"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/9001/r/info":
			if updated {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"id":9001,"name":"客厅格栅灯组","cid":5,"devices":[{"deviceId":5001,"name":"左灯"},{"deviceId":5003,"name":"右灯"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":9001,"name":"客厅格栅灯组","cid":5,"devices":[{"deviceId":5001,"name":"左灯"},{"deviceId":5002,"name":"旧灯"}]}}`))
		case "/apis/iot/v2/thing/schema/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"5003","name":"右灯","subDevices":[{"cid":5,"name":"color light","category":"light","properties":[{"propId":"p"},{"propId":"l"}]}]}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/9001/w/devices":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			updated = true
			_, _ = writer.Write([]byte(`{"success":true,"data":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-configure-secret", "client-configure-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-group-members","locale":"zh-CN","utterance":"把客厅格栅灯组成员改成左灯和右灯","intent":"group.members.update","parameters":{"houseId":"200171","groupName":"客厅格栅灯组","deviceNames":["左灯","右灯"]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, bytes.NewBufferString(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "group-members-update-execute" {
		t.Fatalf("response = %#v", response)
	}
	addList := writeBody["addDeviceList"].([]any)
	removeList := writeBody["removeDeviceList"].([]any)
	if addList[0] != float64(5003) || removeList[0] != float64(5002) {
		t.Fatalf("write body = %#v", writeBody)
	}
}
