package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeEntityGetMatchesAliasName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"device-sensor-1","name":"人在传感器","roomId":"room-1"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-alias-secret", "client-entity-alias-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-alias","locale":"zh-CN","utterance":"看看人在感应器","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"device","name":"人在感应器"}}`
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
	if result["matchedBy"] != "alias_name" {
		t.Fatalf("result = %#v", result)
	}
	entity := result["entity"].(map[string]any)
	if entity["entityId"] != "device-sensor-1" || entity["name"] != "人在传感器" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestInvokeEntityGetMatchesPinyinInitialName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-initial-secret", "client-entity-initial-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-initial","locale":"zh-CN","utterance":"看看 kt","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"room","name":"kt"}}`
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
	if result["matchedBy"] != "initial_name" {
		t.Fatalf("result = %#v", result)
	}
	entity := result["entity"].(map[string]any)
	if entity["entityId"] != "room-1" || entity["name"] != "客厅" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestInvokeEntityGetMatchesFullPinyinName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-pinyin-secret", "client-entity-pinyin-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-pinyin","locale":"zh-CN","utterance":"看看 keting","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"room","name":"keting"}}`
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

func TestInvokeEntityGetMatchesMixedTokenTypoName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-light","name":"灯光区"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"device-rgbw-1","name":"light-色彩灯通用固件 - RGBW-264193-01","roomId":"room-light"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-mixed-token-secret", "client-entity-mixed-token-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-mixed-token","locale":"zh-CN","utterance":"帮我找一下灯光区那个 RGBW 色采灯","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"device","roomName":"灯光区","deviceName":"RGBW 色采灯"}}`
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
	if result["matchedBy"] != "token_name" {
		t.Fatalf("result = %#v", result)
	}
	entity := result["entity"].(map[string]any)
	if entity["entityId"] != "device-rgbw-1" || entity["name"] != "light-色彩灯通用固件 - RGBW-264193-01" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestInvokeEntityGetSuggestsTypoCandidateWithoutAutoAccept(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"device-grid-1","name":"格栅灯","roomId":"room-1"},{"deviceId":"device-downlight-1","name":"筒灯","roomId":"room-1"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-suggestion-secret", "client-entity-suggestion-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-get-suggestion","locale":"zh-CN","utterance":"看看格栏灯","intent":"entity.get","parameters":{"houseId":"house-1","entityType":"device","name":"格栏灯"}}`
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
	if clarification["reason"] != "entity_not_found" {
		t.Fatalf("clarification = %#v", clarification)
	}
	candidates := clarification["candidates"].([]any)
	if len(candidates) != 1 {
		t.Fatalf("candidates = %#v", candidates)
	}
	candidate := candidates[0].(map[string]any)
	if candidate["entityId"] != "device-grid-1" || candidate["name"] != "格栅灯" {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestInvokeEntityListUsesShortKeywordContainment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-light","name":"灯光区"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"deviceId":"device-rgbw-1","name":"light-色彩灯通用固件 - RGBW-264193-01","roomId":"room-light"},{"deviceId":"device-curtain-1","name":"窗帘通用固件","roomId":"room-light"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
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
	app := newInvokeTestApp(t, "Bearer token-entity-list-keyword-secret", "client-entity-list-keyword-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-entity-list-keyword","locale":"zh-CN","utterance":"看看灯光区有哪些灯","intent":"entity.list","parameters":{"houseId":"house-1","entityType":"device","roomName":"灯光区","name":"灯"}}`
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
	if entity["entityId"] != "device-rgbw-1" {
		t.Fatalf("entity = %#v", entity)
	}
}
