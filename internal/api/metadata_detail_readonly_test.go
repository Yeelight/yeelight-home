package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataReadonlyReadPathBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":600,"message":"参数格式错误"}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceWeatherGet(context.Background(), MetadataReadonlyRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Parameters: map[string]any{
			"queryType": "default",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceWeatherGet error = %v", err)
	}
	if !result.Partial || result.Capability != "device.weather.get" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if result.Data != nil {
		t.Fatalf("partial business result should not expose raw data: %#v", result.Data)
	}
}

func TestSceneDetailGetReturnsEditablePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/scene/scene-1/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"scene-1","name":"孩子屋开灯","desc":"暖光","details":[{"typeId":2,"resId":50018330,"resName":"孩子屋吸顶灯","action":0,"rank":0,"params":"{\"set\":{\"p\":true,\"ct\":3000,\"l\":60}}","accessToken":"not-allowed"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunSceneDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "200171",
		Parameters:  map[string]any{"sceneId": "scene-1"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunSceneDetailGet error = %v", err)
	}
	data := result.Data.(map[string]any)
	payload := data["editablePayload"].(map[string]any)
	if payload["sceneId"] != "scene-1" || payload["name"] != "孩子屋开灯" {
		t.Fatalf("payload = %#v", payload)
	}
	details := payload["details"].([]any)
	params := details[0].(map[string]any)["params"].(map[string]any)
	set := params["set"].(map[string]any)
	if set["ct"] != float64(3000) || set["l"] != float64(60) {
		t.Fatalf("params = %#v", params)
	}
	if text, ok := data["detail"].(map[string]any)["accessToken"].(string); ok && text != "" {
		t.Fatalf("detail leaked sensitive value: %#v", data["detail"])
	}
	updateShape := data["updateShape"].(map[string]any)
	detailShape := updateShape["details"].([]any)
	paramShape := detailShape[0].(map[string]any)["params"].(map[string]any)
	flow := updateShape["flow"].([]string)
	if paramShape["set"].(map[string]any)["ct"] == nil || !updateShape["completeList"].(bool) || len(flow) == 0 || flow[0] != "call scene.detail.get" {
		t.Fatalf("data = %#v", data)
	}
}

func TestAutomationDetailGetReturnsEditablePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/200171/automation/auto-1/r/info" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"auto-1","name":"主卧每天9点开灯","startTime":"00:00:00","endTime":"23:59:59","repeatType":2,"repeatValue":"0x7f","version":3,"params":"{\"type\":\"and\",\"conditions\":[{\"type\":\"alarm\",\"clock\":\"09:00:00\"}]}","actions":[{"typeId":2,"resId":50018330,"resName":"主卧吸顶灯","rank":0,"params":"{\"set\":{\"p\":true,\"ct\":3000,\"l\":60}}"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunAutomationDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "200171",
		Parameters:  map[string]any{"automationId": "auto-1"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer secret"},
	})
	if err != nil {
		t.Fatalf("RunAutomationDetailGet error = %v", err)
	}
	data := result.Data.(map[string]any)
	payload := data["editablePayload"].(map[string]any)
	if payload["automationId"] != "auto-1" || payload["repeatType"] != float64(2) || payload["version"] != float64(3) {
		t.Fatalf("payload = %#v", payload)
	}
	params := payload["params"].(map[string]any)
	conditions := params["conditions"].([]any)
	if conditions[0].(map[string]any)["clock"] != "09:00:00" {
		t.Fatalf("params = %#v", params)
	}
	actions := payload["actions"].([]any)
	actionParams := actions[0].(map[string]any)["params"].(map[string]any)
	if actionParams["set"].(map[string]any)["ct"] != float64(3000) {
		t.Fatalf("actions = %#v", actions)
	}
	updateShape := data["updateShape"].(map[string]any)
	conditionShape := updateShape["params"].(map[string]any)
	actionShape := updateShape["actions"].([]any)
	flow := updateShape["flow"].([]string)
	if conditionShape["conditions"] == nil || actionShape[0].(map[string]any)["params"] == nil || !updateShape["completeRule"].(bool) || len(flow) == 0 || flow[0] != "call automation.detail.get" {
		t.Fatalf("data = %#v", data)
	}
}
