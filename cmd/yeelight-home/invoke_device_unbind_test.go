package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceUnbindRequiresLocalApproval(t *testing.T) {
	deviceVisible := true
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			if deviceVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401398"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/device/50018330/w/unbind":
			deviceVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-unbind-secret", "client-unbind-1", "200171")

	input := `{"contractVersion":"1.0","requestId":"req-device-unbind","locale":"zh-CN","utterance":"解绑主灯","intent":"device.unbind","parameters":{"houseId":"200171","deviceId":"50018330","clearMac":true,"unbindRelDevices":true}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("plan exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	confirmation := response["confirmation"].(map[string]any)
	if response["status"] != "confirmation_required" || confirmation["risk"] != "R3" || confirmation["approvalRequired"] != true {
		t.Fatalf("response = %#v", response)
	}
	planID := confirmation["planId"].(string)
	commitInput := `{"contractVersion":"1.0","requestId":"req-device-unbind-commit","locale":"zh-CN","utterance":"确认","intent":"plan.commit","parameters":{"planId":"` + planID + `"}}`

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("blocked exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "blocked" || response["error"].(map[string]any)["code"] != "local_approval_required" {
		t.Fatalf("blocked response = %#v", response)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"approve", "--json", "--plan-id", planID, "--challenge", confirmation["approvalChallenge"].(string)}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("approve exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"invoke", "--stdin"}, strings.NewReader(commitInput), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	response = decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "device-unbind-commit" {
		t.Fatalf("commit response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["deviceId"] != "50018330" || result["clearMac"] != true || result["unbindRelDevices"] != true || result["verified"] != true {
		t.Fatalf("result = %#v", result)
	}
}
