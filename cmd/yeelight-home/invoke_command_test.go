package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestInvokeReturnsAuthRequiredForValidRequest(t *testing.T) {
	input := `{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"客厅暗一点","intent":"light.brightness.adjust"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "auth_required" {
		t.Fatalf("status = %v", response["status"])
	}
	if response["requestId"] != "req-1" {
		t.Fatalf("requestId = %v", response["requestId"])
	}
}

func TestInvokeReturnsJSONErrorWhenCloudWriteFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"灯光区"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/scene/w/create":
			_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-json-error", "client-json-error", "200171")
	input := `{"contractVersion":"1.0","requestId":"req-cloud-write-error","locale":"zh-CN","utterance":"建一个打开灯光区的情景","intent":"scene.create","parameters":{"houseId":"200171","name":"临时情景","actions":[{"targetType":"room","targetId":"401391","targetName":"灯光区","set":{"power":true}}]}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitInternalError {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "error" || response["traceId"] != "invoke-error" {
		t.Fatalf("response = %#v", response)
	}
	errPayload, ok := response["error"].(map[string]any)
	if !ok || errPayload["code"] != "invoke_failed" || !strings.Contains(requestString(errPayload["message"]), "scene create returned non-success") {
		t.Fatalf("error = %#v", response["error"])
	}
	result := response["result"].(map[string]any)
	if result[semantic.FieldSafeToRetry] != false || result[semantic.FieldNextAction] != "report_backend_failure_do_not_retry_same_payload" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(stderr.String(), "invoke: scene create returned non-success") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestInvokeHomeSummaryUsesStoredCredentialAndReadOnlyAPI(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"house-1","name":"默认家庭"},{"id":"house-2","name":"父母家"}]}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-1","name":"默认家庭"},{"id":"house-2","name":"父母家"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-home-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-home-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	input := `{"contractVersion":"1.0","requestId":"req-home-1","locale":"zh-CN","utterance":"看看我的家庭","intent":"home.summary","parameters":{"clientId":"client-from-request-must-be-ignored"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-home-secret") || strings.Contains(stderr.String(), "token-home-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 1 || gotCalls[0] != "POST /apis/iot/v1/house/r/all" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotAuthorization != "Bearer token-home-secret" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-home-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("status = %v, response = %#v", response["status"], response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["houseCount"] != float64(2) {
		t.Fatalf("result = %#v", result)
	}
	houses, ok := result["houses"].([]any)
	if !ok || len(houses) != 2 {
		t.Fatalf("houses = %#v", result["houses"])
	}
	first, ok := houses[0].(map[string]any)
	if !ok || first["houseId"] != "house-1" || first["id"] != "house-1" || first["name"] != "默认家庭" {
		t.Fatalf("first house = %#v", houses[0])
	}
}

func TestInvokeHomeSummaryIgnoresSelectedHouseWhenAccountListsAreEmpty(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all", "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/house-selected/r/info":
			t.Fatalf("home.summary must not fall back to selected house detail")
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-home-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-selected"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	input := `{"contractVersion":"1.0","requestId":"req-home-selected","locale":"zh-CN","utterance":"看看我的家庭","intent":"home.summary"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/house/r/all\nPOST /apis/iot/v1/house/r/list" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	result := response["result"].(map[string]any)
	houses := result["houses"].([]any)
	metrics := response["metrics"].(map[string]any)
	if len(houses) != 0 || result["houseCount"] != float64(0) || result["source"] != "/v1/house/r/all+/v1/house/r/list" || metrics[semantic.FieldAPICalls] != float64(2) {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeHomeListReturnsAllAccountHomesWithSelectedHouse(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-selected","houseName":"当前家庭"},{"houseId":"house-other","houseName":"另一个家"}]}}`))
		case "/apis/iot/v1/house/r/list", "/apis/iot/v1/house/house-selected/r/info":
			t.Fatalf("home.list must not narrow to selected house: %s", request.URL.Path)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-home-list-secret", "client-home-list-1", "house-selected")

	input := `{"contractVersion":"1.0","requestId":"req-home-list-selected","locale":"zh-CN","utterance":"列出我的全部家庭","intent":"home.list"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/house/r/all" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	result := response["result"].(map[string]any)
	houses := result["houses"].([]any)
	if result["houseCount"] != float64(2) || len(houses) != 2 || result["source"] != "/v1/house/r/all" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeHomeListAndSearchUseStoredCredentialAndReadOnlyAPI(t *testing.T) {
	var gotCalls []string
	var gotSearchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":1001,"name":"常住房","img":"home.png","description":"主住宅","areaCode":"CN-310000","areaName":"上海","roomNum":3,"deviceNum":12,"gatewayNum":2,"sceneNum":5,"automationNum":4,"areaNum":1,"accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/house/r/fuzzy":
			if err := json.NewDecoder(request.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":1002,"name":"父母家","desc":"共享家庭","icon":"parent.png","roomNum":2,"deviceNum":8,"gatewayNum":1,"sceneNum":3,"areaNum":1}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-home-list-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-home-list-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	for _, input := range []string{
		`{"contractVersion":"1.0","requestId":"req-home-list","locale":"zh-CN","utterance":"列出我的家庭","intent":"home.list"}`,
		`{"contractVersion":"1.0","requestId":"req-home-search","locale":"zh-CN","utterance":"搜索父母家","intent":"home.search","parameters":{"name":"父母","pageNo":2,"pageSize":5}}`,
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-home-list-secret", "not-allowed"} {
			if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
				t.Fatalf("leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
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
		houses := result["houses"].([]any)
		if result["houseCount"] != float64(1) || len(houses) != 1 {
			t.Fatalf("result = %#v", result)
		}
		first := houses[0].(map[string]any)
		if first["id"] == "" || first["name"] == "" {
			t.Fatalf("first house = %#v", first)
		}
		if _, ok := first["counts"].(map[string]any); !ok {
			t.Fatalf("missing counts: %#v", first)
		}
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/house/r/all\nPOST /apis/iot/v1/house/r/list\nPOST /apis/iot/v1/house/r/fuzzy" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotSearchBody["fuzzyName"] != "父母" || gotSearchBody["pageNo"] != float64(2) || gotSearchBody["pageSize"] != float64(5) {
		t.Fatalf("gotSearchBody = %#v", gotSearchBody)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	callsBeforeMissingKeyword := len(gotCalls)
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(`{"contractVersion":"1.0","requestId":"req-home-search-missing","locale":"zh-CN","utterance":"搜索家庭","intent":"home.search"}`), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if len(gotCalls) != callsBeforeMissingKeyword {
		t.Fatalf("missing keyword should not call cloud: %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "clarification_required" || response["traceId"] != "home-search-clarification" {
		t.Fatalf("response = %#v", response)
	}
}

func TestInvokeEntityListUsesStoredCredentialAndReadOnlyAPI(t *testing.T) {
	var gotAuthorization string
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotAuthorization = request.Header.Get("Authorization")
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"晚安"}]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-entity-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-entity-1", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	input := `{"contractVersion":"1.0","requestId":"req-entity-1","locale":"zh-CN","utterance":"列出我的设备和房间","intent":"entity.list"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-entity-secret") || strings.Contains(stderr.String(), "token-entity-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 6 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotAuthorization != "Bearer token-entity-secret" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("status = %v, response = %#v", response["status"], response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	if result["total"] != float64(2) {
		t.Fatalf("result = %#v", result)
	}
	entities, ok := result["entities"].([]any)
	if !ok || len(entities) != 2 {
		t.Fatalf("entities = %#v", result["entities"])
	}
	metrics, ok := response["metrics"].(map[string]any)
	if !ok || metrics[semantic.FieldAPICalls] != float64(6) {
		t.Fatalf("metrics = %#v", response["metrics"])
	}
}

func TestInvokeEntityListFiltersByEntityTypeAndRoomName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-light","name":"灯光区"},{"id":"room-panel","name":"开关面板区"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-rgbw","name":"light-色彩灯通用固件 - RGBW-264193-01","roomId":"room-light"},{"id":"device-panel","name":"scene_panel-情景面板","roomId":"room-panel"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-entity-filter-secret", "client-entity-filter-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-filter","locale":"zh-CN","utterance":"列一下灯光区设备","intent":"entity.list","parameters":{"houseId":"house-1","entityType":"device","roomName":"灯光区"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["total"] != float64(1) {
		t.Fatalf("result = %#v", result)
	}
	entities := result["entities"].([]any)
	entity := entities[0].(map[string]any)
	if entity["entityId"] != "device-rgbw" || entity["entityType"] != "device" {
		t.Fatalf("entities = %#v", entities)
	}
	counts := result["counts"].(map[string]any)
	if counts["device"] != float64(1) || counts["room"] != nil {
		t.Fatalf("counts = %#v", counts)
	}
}

func TestInvokeEntityGetReturnsMatchedEntityFromReadOnlyList(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-entity-get-secret", "client-entity-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-1","locale":"zh-CN","utterance":"看看主灯","intent":"entity.get","targets":[{"entityType":"device","id":"device-1"}]}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-entity-get-secret") || strings.Contains(stderr.String(), "token-entity-get-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(gotCalls) != 6 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "entity-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %#v", response["result"])
	}
	entity, ok := result["entity"].(map[string]any)
	if !ok ||
		entity["entityId"] != "device-1" ||
		entity["entityType"] != "device" ||
		entity["id"] != "device-1" ||
		entity["type"] != "device" ||
		entity["roomId"] != "room-1" ||
		entity["online"] != true {
		t.Fatalf("entity = %#v", result["entity"])
	}
	if result["matchedBy"] != "id" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInvokeEntityGetMatchesPhoneticNameTypo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-entity-phonetic-secret", "client-entity-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-phonetic","locale":"zh-CN","utterance":"看看客廷","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"room","name":"客廷"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	response := decodeInvokeResponse(t, stdout.Bytes())
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["matchedBy"] != "phonetic_name" {
		t.Fatalf("result = %#v", result)
	}
	entity := result["entity"].(map[string]any)
	if entity["entityId"] != "room-1" || entity["name"] != "客厅" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestInvokeEntityGetAsksClarificationForClosePhoneticCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"},{"id":"room-2","name":"客停"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-entity-phonetic-secret", "client-entity-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-phonetic-ambiguous","locale":"zh-CN","utterance":"看看客廷","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"room","name":"客廷"}}`
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
	if clarification["reason"] != "ambiguous_target" {
		t.Fatalf("clarification = %#v", clarification)
	}
	candidates := clarification["candidates"].([]any)
	if len(candidates) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
}

func TestInvokeEntityGetRequiresTargetWhenMissing(t *testing.T) {
	app := newInvokeTestApp(t, "Bearer token-entity-get-secret", "client-entity-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-missing","locale":"zh-CN","utterance":"看看它","intent":"entity.get"}`
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
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "missing_target" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func TestInvokeEntityGetAsksClarificationWhenEntityNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-entity-get-secret", "client-entity-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-not-found","locale":"zh-CN","utterance":"看看卧室灯","intent":"entity.get","parameters":{"entityName":"卧室灯","entityType":"device"}}`
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
	if response["status"] != "clarification_required" {
		t.Fatalf("response = %#v", response)
	}
	clarification, ok := response["clarification"].(map[string]any)
	if !ok || clarification["reason"] != "entity_not_found" {
		t.Fatalf("clarification = %#v", response["clarification"])
	}
}

func newInvokeTestApp(t *testing.T, token string, clientID string, houseID string) *app {
	t.Helper()
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: token}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: clientID, HouseID: houseID}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	return app
}
