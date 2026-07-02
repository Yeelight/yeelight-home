package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"device-1","name":"面板","mac":"AA:BB:CC:DD","localToken":"not-allowed","valid":1}}`))
		case "/apis/iot/v1/panel/r/button/info/device-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"click":[{"buttonId":"1","name":"单击","type":2,"valid":1,"sort":3,"buttonEvents":[{"id":"event-1","name":"单击","type":1,"valid":1}]}]}}`))
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
	detail := data["detail"].(map[string]any)
	if _, ok := detail[semantic.FieldValid]; ok {
		t.Fatalf("panel detail leaked raw valid field: %#v", detail)
	}
	buttonsByType := data["buttons"].(map[string]any)
	clickButtons := buttonsByType["click"].([]any)
	button := clickButtons[0].(map[string]any)
	for _, leaked := range []string{semantic.FieldType, semantic.FieldValid, semantic.FieldSort} {
		if _, ok := button[leaked]; ok {
			t.Fatalf("panel button leaked raw %s field: %#v", leaked, button)
		}
	}
	if button[semantic.FieldButtonType] == nil || button[semantic.FieldAvailable] == nil || button[semantic.FieldRank] == nil {
		t.Fatalf("panel button missing semantic projection: %#v", button)
	}
	event := button[semantic.FieldButtonEvents].([]any)[0].(map[string]any)
	for _, leaked := range []string{semantic.FieldType, semantic.FieldValid} {
		if _, ok := event[leaked]; ok {
			t.Fatalf("panel event leaked raw %s field: %#v", leaked, event)
		}
	}
	if event[semantic.FieldEventTypeID] == nil || event[semantic.FieldAvailable] == nil {
		t.Fatalf("panel event missing semantic projection: %#v", event)
	}
}

func TestInvokePanelGetResolvesNaturalPanelName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"panel-device-1","name":"scene_panel-智能情景面板-四键","roomId":"room-panel"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/panel/r/detail/panel-device-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"panel-device-1","name":"智能情景面板","localToken":"not-allowed"}}`))
		case "/apis/iot/v1/panel/r/button/info/panel-device-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"click":[{"buttonId":"1","name":"单击"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-panel-name-secret", "client-panel-name-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-panel-get-name","locale":"zh-CN","utterance":"看看情景面板的按钮配置","intent":"panel.get","parameters":{"houseId":"house-1","panelName":"智能情景面板"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(strings.Join(gotCalls, "\n"), "POST /apis/iot/v1/panel/r/detail/panel-device-1") {
		t.Fatalf("panel.get should resolve panelName before readonly call: %#v", gotCalls)
	}
	if strings.Contains(stdout.String(), "not-allowed") {
		t.Fatalf("sensitive panel data leaked: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["deviceId"] != "panel-device-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeSceneSearchFiltersCloudFuzzyRowsByName(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/scene/1001/r/fuzzy" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode scene search body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"sceneId":31,"houseId":1001,"name":"卧室晚安","details":[{}],"accessToken":"not-allowed"},{"sceneId":32,"houseId":1001,"name":"客厅观影","details":[{}],"accessToken":"not-allowed"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-scene-secret", "client-scene-1", "1001")

	input := `{"contractVersion":"1.0","requestId":"req-scene-search","locale":"zh-CN","utterance":"搜索卧室情景","intent":"scene.search","parameters":{"houseId":"1001","name":"卧室","pageNo":2,"pageSize":5}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, forbidden := range []string{"token-scene-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	if gotBody["name"] != "卧室" || gotBody["pageNo"] != float64(2) || gotBody["pageSize"] != float64(5) {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "scene-search-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	scenes := data["scenes"].([]any)
	if len(scenes) != 1 {
		t.Fatalf("scenes = %#v", scenes)
	}
	scene := scenes[0].(map[string]any)
	if scene["name"] != "卧室晚安" {
		t.Fatalf("scene = %#v", scene)
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
	if metrics[semantic.FieldAPICalls] != float64(1) {
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

func TestInvokeHomeMemberCurrentDefaultsToCurrentAccountUID(t *testing.T) {
	var gotCalls []string
	var gotMemberBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/house/r/memberinfoV2":
			if err := json.NewDecoder(request.Body).Decode(&gotMemberBody); err != nil {
				t.Fatalf("decode member body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"uid":"1234567890","nickname":"业主","phoneNumber":"13800138000","email":"owner@example.com","userRole":"owner","accessToken":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-read-secret", "client-home-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-home-member-current-auto","locale":"zh-CN","utterance":"查看我在当前家庭里的成员身份","intent":"home.member.current.get","parameters":{"houseId":"house-1"}}`
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
	expectedCalls := []string{
		"GET /apis/account/user/info",
		"POST /apis/iot/v1/house/r/memberinfoV2",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotMemberBody["uid"] != "1234567890" {
		t.Fatalf("gotMemberBody = %#v", gotMemberBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	metrics := response["metrics"].(map[string]any)
	if metrics[semantic.FieldAPICalls] != float64(2) {
		t.Fatalf("metrics = %#v", metrics)
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
		_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":"device-1","name":"收藏灯","rank":1,"capabilityPid":198666,"gatewayDeviceId":"gw-1","attr":{"p":1},"isBind":true,"typeName":"筒灯","did":"raw-did"}],"meshgroups":[{"meshgroupId":"group-1","name":"收藏灯组","rank":2,"capabilityPid":198666}],"userscenes":[{"sceneId":"scene-1","name":"收藏情景","rank":3,"details":[{"typeId":2,"resId":"device-1"}]}]}}`))
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
	if len(favorites) != 3 {
		t.Fatalf("favorites = %#v", favorites)
	}
	favorite := favorites[0].(map[string]any)
	if favorite["targetType"] != "device" || favorite["targetId"] != "device-1" {
		t.Fatalf("favorite should expose public target fields: %#v", favorite)
	}
	groupFavorite := favorites[1].(map[string]any)
	if groupFavorite["targetType"] != "meshGroup" || groupFavorite["targetId"] != "group-1" {
		t.Fatalf("group favorite should expose public target fields: %#v", groupFavorite)
	}
	sceneFavorite := favorites[2].(map[string]any)
	if sceneFavorite["targetType"] != "scene" || sceneFavorite["targetId"] != "scene-1" {
		t.Fatalf("scene favorite should expose public target fields: %#v", sceneFavorite)
	}
	for _, forbidden := range []string{`"typeId"`, `"resId"`, `"capabilityPid"`, `"gatewayDeviceId"`, `"attr"`, `"did"`, `"typeName"`, `"isBind"`, `"details"`} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("favorite list leaked %s: %s", forbidden, stdout.String())
		}
	}
}

func TestInvokeFavoriteListSkipsEmptyCloudContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/favourite/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-favorite-empty-secret", "client-fav-empty-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-favorite-empty-list","locale":"zh-CN","utterance":"查看收藏","intent":"favorite.list","parameters":{"houseId":"house-1"}}`
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
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	favorites := data["favorites"].([]any)
	if len(favorites) != 0 {
		t.Fatalf("favorites = %#v", favorites)
	}
}

func TestInvokeHomeSortListReadsCloudWithTypeAndTarget(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/sort/r/getSort" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode sort list body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"typeId":2,"resId":50018330,"rank":1}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-secret", "client-sort-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-sort-list","locale":"zh-CN","utterance":"查看房间设备排序","intent":"home.sort.list","parameters":{"houseId":"house-1","sortType":"device_room","roomId":"room-1"}}`
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
	sortType, typeOK := requestInt(gotBody["type"])
	if !typeOK || sortType != 1 || requestString(gotBody["target"]) != "room-1" || requestString(gotBody["roomId"]) != "room-1" {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	sortRows := data["sort"].([]any)
	if len(sortRows) != 1 {
		t.Fatalf("sortRows = %#v", sortRows)
	}
	sortRow := sortRows[0].(map[string]any)
	if sortRow["targetType"] != "device" || sortRow["targetId"] != float64(50018330) {
		t.Fatalf("sort row should expose public target fields: %#v", sortRow)
	}
	for _, forbidden := range []string{`"typeId"`, `"resId"`} {
		if strings.Contains(stdout.String(), forbidden) {
			t.Fatalf("home sort leaked %s: %s", forbidden, stdout.String())
		}
	}
	if len(sortRows) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeHomeSortListResolvesRoomNameBeforeCloudRead(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/sort/r/getSort":
			if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode sort list body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-sort-secret", "client-sort-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-sort-list-room-name","locale":"zh-CN","utterance":"查看客厅设备排序","intent":"home.sort.list","parameters":{"houseId":"house-1","sortType":"device_room","roomName":"客厅"}}`
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
	if requestString(gotBody["target"]) != "room-1" || requestString(gotBody["roomId"]) != "room-1" {
		t.Fatalf("gotBody = %#v", gotBody)
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
	automations := data["automations"].(map[string]any)[semantic.FieldEntries].([]any)
	if len(automations) != 1 || result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeAutomationSupportedV2ProjectsPublicResponse(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/automations/r/supported/v2" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":8784640,"actions":[{"id":1,"type":"event","desc":[{"languageId":"1","value":"Button event"},{"languageId":"2","value":"按键事件"}],"argsDesc":[{"type":"eventId","dataType":"int","unit":"","valueRange":"1,2"}],"supportVersion":"v1,v2"}]}]}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-auto-supported-secret", "client-auto-supported-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-auto-supported-v2","locale":"zh-CN","utterance":"看看自动化支持哪些事件","intent":"automation.supported.v2.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != 1 || gotCalls[0] != "POST /apis/iot/v1/automations/r/supported/v2" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	output := stdout.String()
	for _, forbidden := range []string{"token-auto-supported-secret", `"pid"`, `"actions"`, `"desc"`, `"argsDesc"`, `"supportVersion"`, `"dataType"`} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("output leaked %q: %s", forbidden, output)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	supported := data[semantic.FieldSupportedV2].([]any)
	row := supported[0].(map[string]any)
	conditions := row[semantic.FieldConditions].([]any)
	condition := conditions[0].(map[string]any)
	if row[semantic.FieldCapabilityPID] != float64(8784640) || condition[semantic.FieldName] != "按键事件" || condition[semantic.FieldConditionKind] != "event" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeKnobGetReturnsPartialWhenOneAdapterFails(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/knobs/device-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"device-1","configType":"single","localToken":"not-allowed","sequence":"9","connectType":"1","valid":1}}`))
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
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	single := data[semantic.FieldSingle].(map[string]any)
	for _, leaked := range []string{semantic.FieldSequence, semantic.FieldConnectType, semantic.FieldValid} {
		if _, ok := single[leaked]; ok {
			t.Fatalf("knob detail leaked raw %s field: %#v", leaked, single)
		}
	}
	if single[semantic.FieldAvailable] == nil {
		t.Fatalf("knob detail missing semantic availability: %#v", single)
	}
}

func TestInvokeKnobGetResolvesNaturalKnobName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"knob-device-1","name":"knob_switch-Yeelight Pro M20 旋钮开关","roomId":"room-knob"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/knobs/knob-device-1/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":1,"did":"knob-device-1","configType":"single","localToken":"not-allowed"}}`))
		case "/apis/iot/v1/multi-knob/knob-device-1/detail":
			_, _ = writer.Write([]byte(`{"success":false,"code":"404","msg":"not found"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-knob-name-secret", "client-knob-name-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-knob-get-name","locale":"zh-CN","utterance":"看看 M20 旋扭绑定了什么","intent":"knob.get","parameters":{"houseId":"house-1","knobName":"M20 旋扭开关"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(strings.Join(gotCalls, "\n"), "GET /apis/iot/v1/knobs/knob-device-1/detail") {
		t.Fatalf("knob.get should resolve knobName before readonly call: %#v", gotCalls)
	}
	if strings.Contains(stdout.String(), "not-allowed") {
		t.Fatalf("sensitive knob data leaked: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "partial" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["deviceId"] != "knob-device-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeMetadataLocalGuidanceDoesNotPersist(t *testing.T) {
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
