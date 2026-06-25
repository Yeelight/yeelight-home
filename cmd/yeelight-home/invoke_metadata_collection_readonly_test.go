package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeSceneAndAutomationListUseCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/scene/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":21,"houseId":"house-1","name":"晚安","details":[{"accessToken":"nope"}]}]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"auto-1","houseId":"house-1","name":"回家开灯","status":1,"params":"hidden","actions":[{"secret":"nope"}]}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-list-secret", "client-list-1", "house-1")

	for _, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-scene-list","locale":"zh-CN","utterance":"列出情景","intent":"scene.list","parameters":{"houseId":"house-1"}}`,
		`{"contractVersion":"1.0","requestId":"req-automation-list","locale":"zh-CN","utterance":"列出自动化","intent":"automation.list","parameters":{"houseId":"house-1"}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-list-secret", "accessToken", "hidden", "secret", "nope"} {
			if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
				t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
			}
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if response["status"] != "success" {
			t.Fatalf("response = %#v", response)
		}
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/scene/r/all\nPOST /apis/iot/v1/automations/r/list" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestInvokeDeviceVirtualCountAndNodeSortedDeviceList(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/device/r/house-1/virturlNum":
			_, _ = writer.Write([]byte(`{"success":true,"data":2}`))
		case "/apis/iot/v1/node/r/1/room-1/device":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"dev-1","alias":"主灯","mac":"AA:BB:CC","rank":1}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-node-list-secret", "client-node-list-1", "house-1")

	for _, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-virtual-count","locale":"zh-CN","utterance":"虚拟设备数量","intent":"device.virtual_count.get","parameters":{"houseId":"house-1"}}`,
		`{"contractVersion":"1.0","requestId":"req-node-device-list","locale":"zh-CN","utterance":"房间设备排序","intent":"node.sorted_device.list","parameters":{"houseId":"house-1","resType":"1","resId":"room-1"}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		if strings.Contains(stdout.String(), "token-node-list-secret") || strings.Contains(stdout.String(), "AA:BB:CC") {
			t.Fatalf("sensitive output leaked: %s", stdout.String())
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if response["status"] != "success" {
			t.Fatalf("response = %#v", response)
		}
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/device/r/house-1/virturlNum\nPOST /apis/iot/v1/node/r/1/room-1/device" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}
