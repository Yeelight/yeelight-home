package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeUpgradeProgressGetUsesTargetDevice(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/upgrade/r/progress" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1","state":1,"localToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-upgrade-secret", "client-upgrade-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-upgrade-progress","locale":"zh-CN","utterance":"查看主灯升级进度","intent":"upgrade.progress.get","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/iot/v1/upgrade/r/progress" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-upgrade-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "upgrade-progress-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false || result["deviceId"] != "device-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeProgressGetUsesProgressKey(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/progress/r/job-1" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"status":1,"progress":"80%","accessToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-progress-secret", "client-progress-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-progress","locale":"zh-CN","utterance":"查看同步任务进度","intent":"progress.get","parameters":{"houseId":"house-1","key":"job-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/iot/v1/progress/r/job-1" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if strings.Contains(stdout.String(), "not-allowed") || strings.Contains(stdout.String(), "token-progress-secret") {
		t.Fatalf("output leaked secret: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "progress-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeMaintenanceReadNextUsesSemanticAdapters(t *testing.T) {
	var gotCalls []string
	var gotQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotQueries = append(gotQueries, request.URL.RawQuery)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/appupgrade/r/latestfile":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"type":"1","osType":"1","version":8,"digitalVersion":"8.0.0","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/ota/upgrade/r/batchListFilesByVersion":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"firmwareType":"main","version":44,"secret":"not-allowed"}]}`))
		case "/apis/iot/v1/nodeConfig/r/node_property":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"property":"power","range":"0/1","localToken":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-maintenance-next-secret", "client-maintenance-next-1", "house-1")

	inputs := []string{
		`{"contractVersion":"1.0","requestId":"req-app-upgrade","locale":"zh-CN","utterance":"查看用户版安卓 App 最新升级版本","intent":"app_upgrade.latest.get","parameters":{"appType":"yeelight","osType":"android","languageCode":"zh-CN"}}`,
		`{"contractVersion":"1.0","requestId":"req-ota-version","locale":"zh-CN","utterance":"按版本查看固件文件","intent":"ota.version_file.batch_list","parameters":{"firmwareType":"main","version":44,"languageCode":"zh","script":"Hans","region":"CN"}}`,
		`{"contractVersion":"1.0","requestId":"req-node-property","locale":"zh-CN","utterance":"查看主灯节点属性配置","intent":"node.property_config.get","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"nodeType":"device"}}`,
	}
	for _, input := range inputs {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-maintenance-next-secret", "not-allowed"} {
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
	expectedCalls := []string{
		"POST /apis/iot/v1/appupgrade/r/latestfile",
		"POST /apis/iot/v1/ota/upgrade/r/batchListFilesByVersion",
		"POST /apis/iot/v1/nodeConfig/r/node_property",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if !strings.Contains(gotQueries[1], "language=zh") || !strings.Contains(gotQueries[1], "region=CN") || !strings.Contains(gotQueries[1], "script=Hans") {
		t.Fatalf("unexpected ota query: %s", gotQueries[1])
	}
	if !strings.Contains(gotQueries[2], "nodeId=device-1") || !strings.Contains(gotQueries[2], "nodeType=device") {
		t.Fatalf("unexpected node property query: %s", gotQueries[2])
	}
}

func TestInvokeMaintenanceReadonlyAuthBoundaryReturnsPartialJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "unauthorized token-secret-should-not-leak", http.StatusUnauthorized)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-maintenance-auth-secret", "client-maintenance-auth-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-ota-unauthorized","locale":"zh-CN","utterance":"按版本查看固件文件","intent":"ota.version_file.batch_list","parameters":{"firmwareType":"main","version":44,"languageCode":"zh","script":"Hans","region":"CN"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, forbidden := range []string{"token-maintenance-auth-secret", "token-secret-should-not-leak"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "partial" {
		t.Fatalf("response = %#v", response)
	}
	warnings, ok := response["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "cloud_authorization_boundary" {
		t.Fatalf("warnings = %#v", response["warnings"])
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false || result["rawShape"] != nil {
		t.Fatalf("result = %#v", result)
	}
	data := result["data"].(map[string]any)
	if data["httpStatus"] != float64(http.StatusUnauthorized) {
		t.Fatalf("data = %#v", data)
	}
}
