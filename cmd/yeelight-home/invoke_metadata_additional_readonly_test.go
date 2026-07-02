package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeAdditionalReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	var gotHouseHeaderCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		if request.Header.Get("houseId") != "" || request.Header.Get("house-id") != "" {
			gotHouseHeaderCalls = append(gotHouseHeaderCalls, request.Method+" "+request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/scene/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":"scene-1","houseId":"house-1","roomId":"room-1","name":"电影","details":[{"secret":"nope"}]}]}}`))
		case "/apis/iot/v1/schedulejob/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"job-1","houseId":"house-1","name":"定时关灯","actions":[{"accessToken":"nope"}]}]}}`))
		case "/apis/iot/v1/messagecenter/r/messages":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"msg-1","title":"设备离线","content":"设备已离线，请检查供电和网关连接"}]}`))
		case "/apis/iot/v1/product-domain/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"domain-1","name":"照明","code":"lighting","token":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/faq-types":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"PRODUCT_USE_HELP","description":"产品使用帮助","secret":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/faq-item-types":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"TEXT","description":"文本","token":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/locales":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"zh-CN","name":"简体中文","authorization":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/page":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"faq-1","pid":"pid-1","title":"如何重置","answer":"按住按键","password":"nope"}]}}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/pageDetail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"faq-2","pid":"pid-1","title":"如何配网","answer":"打开 App","secret":"nope"}]}}`))
		case "/apis/iot/v2/thing/schema/property/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"prop-1","name":"开关","dataType":"bool","secret":"nope"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-additional-invoke-secret", "client-additional-invoke-1", "house-1")

	for _, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-scene-scoped-list","locale":"zh-CN","utterance":"列出客厅情景","intent":"scene.scoped.list","parameters":{"houseId":"house-1","roomId":"room-1"}}`,
		`{"contractVersion":"1.0","requestId":"req-schedule-job-list","locale":"zh-CN","utterance":"列出定时任务","intent":"schedule_job.list","parameters":{"houseId":"house-1"}}`,
		`{"contractVersion":"1.0","requestId":"req-message-list","locale":"zh-CN","utterance":"查看消息","intent":"message.list","parameters":{"houseId":"house-1"}}`,
		`{"contractVersion":"1.0","requestId":"req-product-domain-list","locale":"zh-CN","utterance":"查看产品域","intent":"thing.product_domain.list","parameters":{}}`,
		`{"contractVersion":"1.0","requestId":"req-faq-type-list","locale":"zh-CN","utterance":"查看 FAQ 类型","intent":"thing.product_faq.type.list","parameters":{}}`,
		`{"contractVersion":"1.0","requestId":"req-faq-item-type-list","locale":"zh-CN","utterance":"查看 FAQ 项类型","intent":"thing.product_faq.item_type.list","parameters":{}}`,
		`{"contractVersion":"1.0","requestId":"req-faq-locale-list","locale":"zh-CN","utterance":"查看 FAQ 支持语言","intent":"thing.product_faq.locale.list","parameters":{}}`,
		`{"contractVersion":"1.0","requestId":"req-faq-page-list","locale":"zh-CN","utterance":"分页查看产品 FAQ","intent":"thing.product_faq.page.list","parameters":{"capabilityPid":"pid-1","pageNo":1,"pageSize":10}}`,
		`{"contractVersion":"1.0","requestId":"req-faq-page-detail-list","locale":"zh-CN","utterance":"分页查看产品 FAQ 详情","intent":"thing.product_faq.page_detail.list","parameters":{"capabilityPid":"pid-1","pageNo":1,"pageSize":10}}`,
		`{"contractVersion":"1.0","requestId":"req-property-list","locale":"zh-CN","utterance":"查看物模型属性","intent":"thing.property.list","parameters":{}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-additional-invoke-secret", "accessToken", "secret", "nope"} {
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
		requestID := response["requestId"].(string)
		if strings.HasPrefix(requestID, "req-message") ||
			strings.HasPrefix(requestID, "req-product") ||
			strings.HasPrefix(requestID, "req-faq") ||
			strings.HasPrefix(requestID, "req-property") {
			result := response["result"].(map[string]any)
			if _, ok := result["houseId"]; ok {
				t.Fatalf("house-independent readonly response exposed houseId: %#v", response)
			}
		}
	}
	if len(gotHouseHeaderCalls) != 0 {
		t.Fatalf("house headers should not be sent by these readonly adapters: %#v", gotHouseHeaderCalls)
	}
	wantCalls := []string{
		"POST /apis/iot/v1/scene/r/list",
		"POST /apis/iot/v1/schedulejob/r/list",
		"GET /apis/iot/v1/messagecenter/r/messages",
		"POST /apis/iot/v1/product-domain/r/list",
		"GET /apis/iot/v1/platform/thing/product_faq/r/faq-types",
		"GET /apis/iot/v1/platform/thing/product_faq/r/faq-item-types",
		"GET /apis/iot/v1/platform/thing/product_faq/r/locales",
		"POST /apis/iot/v1/platform/thing/product_faq/r/page",
		"POST /apis/iot/v1/platform/thing/product_faq/r/pageDetail",
		"GET /apis/iot/v2/thing/schema/property/r/list",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestInvokeSceneScopedListResolvesRoomName(t *testing.T) {
	var gotSceneBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"灯光区"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/scene/r/list":
			if err := json.NewDecoder(request.Body).Decode(&gotSceneBody); err != nil {
				t.Fatalf("decode scene scoped body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":"scene-1","houseId":"house-1","roomId":"room-1","name":"灯光区回家"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-scoped-secret", "client-scene-scoped-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-scene-scoped-room-name","locale":"zh-CN","utterance":"灯光去有哪些情景","intent":"scene.scoped.list","parameters":{"houseId":"house-1","roomName":"灯光去"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotSceneBody["roomId"] != "room-1" {
		t.Fatalf("gotSceneBody = %#v", gotSceneBody)
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" || response["traceId"] != "scene-scoped-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
}
