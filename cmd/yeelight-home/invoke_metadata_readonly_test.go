package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeAccountInfoReturnsRedactedResult(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"code":"200","data":{"userId":"1234567890","nickname":"测试用户","phone":"13800138000","email":"user@example.com","accessToken":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-account-secret", "client-account-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-account-info","locale":"zh-CN","utterance":"查看账号信息","intent":"account.info"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/account/user/info" {
		t.Fatalf("account.info should not require houseId or call house-scoped APIs: %#v", gotCalls)
	}
	output := stdout.String()
	for _, forbidden := range []string{"token-account-secret", "not-allowed", "13800138000", "user@example.com", "1234567890"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, output)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "account-info-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokePanelGetUsesCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/detail/device-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"device-1","name":"面板","mac":"AA:BB:CC:DD","localToken":"not-allowed"}}`))
		case "/apis/iot/v1/panel/r/button/info/device-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"click":[{"buttonId":"1","name":"单击"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-secret", "client-panel-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-panel-get","locale":"zh-CN","utterance":"查看面板配置","intent":"panel.get","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 2 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if strings.Contains(stdout.String(), "not-allowed") || strings.Contains(stdout.String(), "AA:BB:CC:DD") {
		t.Fatalf("sensitive panel data leaked: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "panel-get-partial" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
	data, ok := result["data"].(map[string]any)
	if !ok || data["detail"] == nil || data["buttons"] == nil {
		t.Fatalf("data = %#v", result["data"])
	}
}

func TestInvokeHomeMemberListUsesCloudReadonlyAdapterWithRedaction(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/memberlistV2" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","userRole":"owner"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-member-secret", "client-member-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-home-member","locale":"zh-CN","utterance":"查看家庭成员","intent":"home.member.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 1 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, forbidden := range []string{"token-member-secret", "1234567890", "13800138000", "owner@example.com"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, stdout.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "home-member-list-partial" {
		t.Fatalf("response = %#v", response)
	}
	metrics := response["metrics"].(map[string]any)
	if metrics["apiCalls"] != float64(1) {
		t.Fatalf("metrics = %#v", metrics)
	}
}

func TestInvokeHomeMemberCurrentAndStatUseReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/memberinfoV2":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","userRole":"owner","accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/house/house-1/r/stat":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"roomCount":2,"deviceCount":8,"secret":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-read-secret", "client-home-1", "house-1")

	inputs := []string{
		`{"contractVersion":"1.0","requestId":"req-home-member-current","locale":"zh-CN","utterance":"查看当前家庭成员信息","intent":"home.member.current.get","parameters":{"houseId":"house-1","uid":"1234567890"}}`,
		`{"contractVersion":"1.0","requestId":"req-home-stat","locale":"zh-CN","utterance":"查看家庭统计","intent":"home.stat.get","parameters":{"houseId":"house-1"}}`,
	}
	for _, input := range inputs {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-home-read-secret", "not-allowed", "1234567890", "13800138000", "owner@example.com"} {
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
		"POST /apis/iot/v1/house/r/memberinfoV2",
		"POST /apis/iot/v1/house/house-1/r/stat",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestInvokeAIVoiceProductListUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/ai/voice/product/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[1001,1002],"accessToken":"not-allowed"}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-ai-voice-product-secret", "client-ai-voice-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-ai-voice-product","locale":"zh-CN","utterance":"哪些产品支持 AI 语音识别","intent":"ai_voice.product.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v1/ai/voice/product/r/list" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-ai-voice-product-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "ai_voice-product-list-readonly" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeGeoAreaReadonlyUsesCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/area/r/0/children":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":1,"name":"北京","fullname":"北京-北京","code":"CN-110000","accessToken":"not-allowed"}]}`))
		case "/apis/iot/v1/area/r/areas":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":2,"name":"上海","fullname":"上海-上海","code":"CN-310000","secret":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-geo-area-secret", "client-geo-area-1", "house-1")

	inputs := []string{
		`{"contractVersion":"1.0","requestId":"req-geo-area-children","locale":"zh-CN","utterance":"列出可选城市区域","intent":"geo_area.children.list","parameters":{"parentId":0}}`,
		`{"contractVersion":"1.0","requestId":"req-geo-area-search","locale":"zh-CN","utterance":"搜索上海区域编码","intent":"geo_area.search","parameters":{"name":"上海"}}`,
	}
	for _, input := range inputs {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-geo-area-secret", "not-allowed"} {
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
		if result["cloudWrites"] != false || len(data["areas"].([]any)) != 1 {
			t.Fatalf("result = %#v", result)
		}
	}
	expectedCalls := []string{
		"GET /apis/iot/v1/area/r/0/children",
		"POST /apis/iot/v1/area/r/areas",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestInvokeFavoriteListUsesCloudReadonlyAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/favourite/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"resId":"device-1","typeId":2,"rank":1}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-favorite-secret", "client-fav-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-favorite-list","locale":"zh-CN","utterance":"查看收藏","intent":"favorite.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
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
	favorites := data["favorites"].([]any)
	if len(favorites) != 1 {
		t.Fatalf("favorites = %#v", favorites)
	}
}

func TestInvokeHomeSortListRequiresSortContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-secret", "client-sort-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-sort-list","locale":"zh-CN","utterance":"查看首页排序","intent":"home.sort.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "partial" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	unknowns := result["unknownEvidence"].([]any)
	if len(unknowns) != 1 || unknowns[0] != "home_sort_query_context_missing" {
		t.Fatalf("unknownEvidence = %#v", unknowns)
	}
}

func TestInvokeAutomationListPageUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/automations/house-1/r/list/2/10" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"auto-1","name":"晚安","houseId":"house-1","accessToken":"not-allowed","actions":"[]"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-list-secret", "client-auto-list-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-auto-list-page","locale":"zh-CN","utterance":"分页查看自动化","intent":"automation.list.page","parameters":{"houseId":"house-1","pageNo":2,"pageSize":10}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 1 || gotCalls[0] != "GET /apis/iot/v1/automations/house-1/r/list/2/10" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, forbidden := range []string{"token-auto-list-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, stdout.String())
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
	automations := data["automations"].(map[string]any)["rows"].([]any)
	if len(automations) != 1 || result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeKnobGetReturnsPartialWhenOneAdapterFails(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/knobs/device-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"device-1","configType":"single","localToken":"not-allowed"}}`))
		case "/apis/iot/v1/multi-knob/device-1/detail":
			_, _ = writer.Write([]byte(`{"success":false,"code":"404","msg":"not found"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-knob-secret", "client-knob-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-knob-get","locale":"zh-CN","utterance":"查看旋钮","intent":"knob.get","targets":[{"entityType":"device","id":"device-1"}],"parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 2 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if strings.Contains(stdout.String(), "not-allowed") {
		t.Fatalf("token-like knob data leaked: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "partial" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeMetadataLocalPlanDoesNotPersist(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-favorite-secret", "client-fav-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-favorite-plan","locale":"zh-CN","utterance":"帮我规划收藏","intent":"favorite.plan","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "favorite-plan-local" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["persistentWrites"] != false || result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeAutomationCapabilitiesReturnsRuntimeBoundary(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-auto-cap-secret", "client-auto-1", "house-1")
	input := `{"contractVersion":"1.0","requestId":"req-auto-cap","locale":"zh-CN","utterance":"自动化支持什么","intent":"automation.capabilities"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "automation-capabilities-local" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
}
