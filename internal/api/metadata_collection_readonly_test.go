package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetadataCollectionReadonlySceneAutomationAndNodeProjection(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/scene/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":21,"houseId":1001,"name":"晚安","roomId":10,"details":[{"secret":"nope"}],"attr":{"token":"nope"}}]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":31,"houseId":1001,"name":"回家开灯","status":1,"params":"secret-json","actions":[{"accessToken":"nope"}]}]}`))
		case "/apis/iot/v1/device/r/1001/virturlNum":
			_, _ = writer.Write([]byte(`{"success":true,"data":7}`))
		case "/apis/iot/v1/node/r/1/10/device":
			if request.Header.Get("houseId") != "1001" || request.Header.Get("house-id") != "1001" {
				t.Fatalf("node sorted device list missing house headers: houseId=%q house-id=%q", request.Header.Get("houseId"), request.Header.Get("house-id"))
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":41,"alias":"筒灯","mac":"AA:BB:CC","rank":2,"capability":"p,l"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID: "1001",
		Parameters: map[string]any{
			"resType": "1",
			"resId":   "10",
		},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-collection-secret",
			ClientID:      "client-collection-1",
		},
	}

	sceneResult, err := client.RunSceneList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunSceneList error: %v", err)
	}
	automationResult, err := client.RunAutomationList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunAutomationList error: %v", err)
	}
	virtualResult, err := client.RunDeviceVirtualCountGet(context.Background(), request)
	if err != nil {
		t.Fatalf("RunDeviceVirtualCountGet error: %v", err)
	}
	nodeResult, err := client.RunNodeSortedDeviceList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunNodeSortedDeviceList error: %v", err)
	}

	wantCalls := []string{
		"POST /apis/iot/v1/scene/r/all",
		"POST /apis/iot/v1/automations/r/list",
		"POST /apis/iot/v1/device/r/1001/virturlNum",
		"POST /apis/iot/v1/node/r/1/10/device",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	data, _ := json.Marshal([]MetadataReadonlyResult{sceneResult, automationResult, virtualResult, nodeResult})
	for _, forbidden := range []string{"token-collection-secret", "AA:BB:CC", "secret-json", "accessToken", "nope", "attr", "actions", "params"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, string(data))
		}
	}
	if sceneResult.Capability != "scene.list" || automationResult.Capability != "automation.list" || virtualResult.Capability != "device.virtual_count.get" || nodeResult.Capability != "node.sorted_device.list" {
		t.Fatalf("unexpected capabilities: %#v %#v %#v %#v", sceneResult, automationResult, virtualResult, nodeResult)
	}
	scenes := sceneResult.Data.(map[string]any)["scenes"].([]any)
	if scenes[0].(map[string]any)["actionCount"] != 1 {
		t.Fatalf("scene projection = %#v", scenes)
	}
	devices := nodeResult.Data.(map[string]any)["devices"].([]any)
	if devices[0].(map[string]any)["rank"] != "2" {
		t.Fatalf("node projection = %#v", devices)
	}
}

func TestNodeSortedDeviceListRequiresNodeContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunNodeSortedDeviceList(context.Background(), MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-node-secret",
		},
	})
	if err != nil {
		t.Fatalf("RunNodeSortedDeviceList error: %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "node_context_missing" {
		t.Fatalf("result = %#v", result)
	}
}

func TestNodeSortedDeviceListBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":601,"message":"非法参数：houseId"}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunNodeSortedDeviceList(context.Background(), MetadataReadonlyRequest{
		HouseID: "1001",
		Parameters: map[string]any{
			"resType": "1",
			"resId":   "10",
		},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-node-secret",
		},
	})
	if err != nil {
		t.Fatalf("RunNodeSortedDeviceList error: %v", err)
	}
	if !result.Partial || result.APICalls != 1 || len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("result = %#v", result)
	}
}
