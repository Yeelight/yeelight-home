package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeGroupListAndSearchUseCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/1001/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":21,"houseId":1001,"name":"一楼","roomId":10,"secret":"not-allowed"},{"id":22,"houseId":1001,"name":"二楼","roomId":12,"secret":"not-allowed"}]}}`))
		case "/apis/iot/v2/thing/manage/house/1001/group/r/info/2/5":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":22,"houseId":1001,"name":"二楼","roomId":12,"secret":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-group-secret", "client-group-1", "1001")

	for index, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-group-list","locale":"zh-CN","utterance":"列出这个家的设备组","intent":"group.list","parameters":{"houseId":"1001"}}`,
		`{"contractVersion":"1.0","requestId":"req-group-search","locale":"zh-CN","utterance":"搜索二楼设备组","intent":"group.search","parameters":{"houseId":"1001","groupName":"二楼","pageNo":2,"pageSize":5}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-group-secret", "not-allowed"} {
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
		result := response["result"].(map[string]any)
		data := result["data"].(map[string]any)
		groups := data["groups"].([]any)
		expectedCount := 2
		if index == 1 {
			expectedCount = 1
		}
		if len(groups) != expectedCount {
			t.Fatalf("groups = %#v", data["groups"])
		}
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/iot/v2/thing/manage/house/1001/group/r/info/1/100\nGET /apis/iot/v2/thing/manage/house/1001/group/r/info/2/5" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestInvokeGroupDetailGetUsesThingManageReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/1001/group/22/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":22,"houseId":1001,"name":"二楼","configs":[{"property":"name","desc":"名称","value":"二楼","access":6,"format":"string","scale":1,"zoom":0}],"devices":[{"deviceId":"device-1","name":"灯1","pid":"198666","pcid":"31"}],"accessToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-group-secret", "client-group-1", "1001")

	input := `{"contractVersion":"1.0","requestId":"req-group-detail","locale":"zh-CN","utterance":"查看二楼分组详情","intent":"group.detail.get","parameters":{"houseId":"1001","groupId":"22"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v2/thing/manage/house/1001/group/22/r/info" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-group-secret", "not-allowed", "\"desc\"", "\"scale\"", "\"zoom\""} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "group-detail-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	if result["cloudWrites"] != false || data["detail"] == nil {
		t.Fatalf("result = %#v", result)
	}
	detail := data["detail"].(map[string]any)
	configs := detail["configs"].([]any)
	config := configs[0].(map[string]any)
	if config["description"] != "名称" || config["property"] != "name" || config["value"] != "二楼" {
		t.Fatalf("config = %#v", config)
	}
	if detail["configCount"] == nil || detail["deviceCount"] == nil {
		t.Fatalf("detail = %#v", detail)
	}
}
