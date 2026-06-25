package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeDeviceListUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":31,"did":9001,"pid":101,"type":1,"name":"主灯","houseId":1001,"roomId":10,"capability":"p,l,ct","localToken":"not-allowed","mac":"AA:BB:CC:DD","deviceKey":"secret-key","shadow":{"p":true}}],"meshgroups":[{"meshGroupId":41,"name":"筒灯组","deviceIds":[31,33],"secret":"nope"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-device-list-secret", "client-device-list-1", "1001")

	input := `{"contractVersion":"1.0","requestId":"req-device-list","locale":"zh-CN","utterance":"列出这个家的设备","intent":"device.list","parameters":{"houseId":"1001"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/iot/v1/device/r/all" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-device-list-secret", "not-allowed", "AA:BB:CC:DD", "secret-key", "shadow", "nope"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "device-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	devices := data["devices"].([]any)
	if len(devices) != 1 || devices[0].(map[string]any)["name"] != "主灯" {
		t.Fatalf("devices = %#v", data["devices"])
	}
}
