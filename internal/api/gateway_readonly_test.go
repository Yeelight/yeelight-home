package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/gateway/gateway-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"E1 网关","mac":"AA:BB:CC:DD","localToken":"not-allowed","psk":"not-allowed","configs":[{"propId":"ltk","value":"not-allowed"},{"propId":"mibk","value":"not-allowed"}],"supportedBridgeType":["thread"]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/gateway/gateway-1/r/thread-info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"networkName":"yeelight-thread","extendedPanId":"abcd","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/scene/r/gateway-1/related/sceneId":
			_, _ = writer.Write([]byte(`{"success":true,"data":["scene-1","scene-2"]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID: "house-1",
		Parameters: map[string]any{
			"gatewayId": "gateway-1",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-gateway-secret", ClientID: "client-1"},
	}

	detail, err := client.RunGatewayDetailGet(context.Background(), request)
	if err != nil {
		t.Fatalf("detail err = %v", err)
	}
	thread, err := client.RunGatewayThreadGet(context.Background(), request)
	if err != nil {
		t.Fatalf("thread err = %v", err)
	}
	relations, err := client.RunGatewaySceneRelationList(context.Background(), request)
	if err != nil {
		t.Fatalf("relations err = %v", err)
	}

	if len(gotCalls) != 3 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range []MetadataReadonlyResult{detail, thread, relations} {
		if result.Partial || result.APICalls != 1 || result.DeviceID != "gateway-1" {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal result data: %v", err)
		}
		text := string(data)
		for _, forbidden := range []string{"not-allowed", "AA:BB:CC:DD", "token-gateway-secret"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("result leaked %q: %#v", forbidden, result.Data)
			}
		}
	}
}

func TestGatewayListDefaultsToFirstPage(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gateway-1","name":"网关","online":true,"mac":"AA:BB:CC:DD","localToken":"not-allowed","bindKey":"not-allowed","psk":"not-allowed","ltk":"not-allowed","mibk":"not-allowed","midk":"not-allowed","hrbk":"not-allowed","meibk":"not-allowed","configs":[{"propId":"ltk","value":"not-allowed"},{"propId":"mibk","value":"not-allowed"},{"propId":"wifiPassword","value":"not-allowed"}]}],"total":1}}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunGatewayList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-gateway-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("gateway list err = %v", err)
	}
	if gotCall != "GET /apis/iot/v2/thing/manage/house/house-1/gateway/r/info/1/100" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if result.Capability != "gateway.list" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatalf("marshal result data: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{"not-allowed", "AA:BB:CC:DD", "localToken", "bindKey", "psk", "ltk", "mibk", "midk", "hrbk", "meibk", "wifiPassword", "configs"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("gateway list leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "configCount") {
		t.Fatalf("gateway list should keep only configCount summary: %s", text)
	}
}

func TestGatewayStatsRequiresHouseContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunGatewayStatsList(context.Background(), MetadataReadonlyRequest{
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-gateway-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("gateway stats err = %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "house_context_missing" {
		t.Fatalf("result = %#v", result)
	}
}

func TestGatewayStatsListReturnsRedactedProjection(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":"gateway-1","name":"默认网关","deviceNum":9,"roomNum":2,"did":"raw-did","img":"raw.png","isBind":1,"attr":{"secret":"not-allowed"},"subDevices":[{"id":"child"}]}]}}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunGatewayStatsList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-gateway-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("gateway stats err = %v", err)
	}
	if gotCall != "POST /apis/iot/v1/device/r/gatewayswithstats" || result.Capability != "gateway.stats.list" || result.APICalls != 1 {
		t.Fatalf("gotCall=%q result=%#v", gotCall, result)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatalf("marshal result data: %v", err)
	}
	text := string(data)
	for _, forbidden := range []string{"not-allowed", `"did"`, `"img"`, `"isBind"`, `"attr"`, `"subDevices"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("gateway stats leaked %q: %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "deviceCount") || !strings.Contains(text, "roomCount") {
		t.Fatalf("gateway stats missing summary counts: %s", text)
	}
}
