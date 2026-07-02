package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoomReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	var gotSearchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/room/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"roomId":10,"houseId":1001,"name":"客厅","img":"living.png","deviceIds":[1,2],"gatewayDeviceIds":[9],"accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/room/1001/r/fuzzy":
			if err := json.NewDecoder(request.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"roomId":11,"houseId":1001,"name":"卧室","deviceIds":[3],"secret":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{"name": "卧", "pageNo": 2, "pageSize": 5},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-room-secret",
			ClientID:      "client-1",
		},
	}
	list, err := client.RunRoomList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRoomList error: %v", err)
	}
	search, err := client.RunRoomSearch(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRoomSearch error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/room/r/all\nPOST /apis/iot/v1/room/1001/r/fuzzy" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotSearchBody["fuzzyName"] != "卧" || gotSearchBody["pageNo"] != float64(2) || gotSearchBody["pageSize"] != float64(5) {
		t.Fatalf("gotSearchBody = %#v", gotSearchBody)
	}
	for _, result := range []MetadataReadonlyResult{list, search} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, _ := json.Marshal(result.Data)
		for _, forbidden := range []string{"token-room-secret", "not-allowed"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
	rooms := list.Data.(map[string]any)["rooms"].([]any)
	first := rooms[0].(map[string]any)
	if first["id"] != "10" || first["name"] != "客厅" || first["deviceCount"] != 2 {
		t.Fatalf("first room = %#v", first)
	}
}

func TestRunRoomSearchFallsBackToLocalPhoneticMatch(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/room/1001/r/fuzzy":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/room/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"roomId":10,"houseId":1001,"name":"客厅"},{"roomId":11,"houseId":1001,"name":"主卧"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	search, err := client.RunRoomSearch(context.Background(), MetadataReadonlyRequest{
		HouseID:     "1001",
		Parameters:  map[string]any{"name": "客廷"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-room-secret"},
	})
	if err != nil {
		t.Fatalf("RunRoomSearch error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/room/1001/r/fuzzy\nPOST /apis/iot/v1/room/r/all" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if search.Partial || search.APICalls != 2 || len(search.Warnings) != 1 || search.Warnings[0] != "room_search_local_fuzzy_fallback" {
		t.Fatalf("search = %#v", search)
	}
	rooms := search.Data.(map[string]any)["rooms"].([]any)
	first := rooms[0].(map[string]any)
	if first["id"] != "10" || first["name"] != "客厅" {
		t.Fatalf("first room = %#v", first)
	}
}

func TestRunRoomDetailReturnsPublicProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/room/10/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"roomId":10,"houseId":1001,"name":"灯光区","attr":{"secret":"not-allowed"},"devices":[{"deviceId":5001,"name":"主灯","roomId":10,"pid":198666,"attr":{"p":1},"did":"raw-did","isBind":1,"typeName":"色温灯"}],"userscenes":[{"sceneId":7001,"name":"观影模式","details":[{"typeId":2,"resId":5001,"params":"{\"set\":{\"p\":1,\"l\":60}}"}]}],"accessToken":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunRoomDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "1001",
		Parameters:  map[string]any{"roomId": "10"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-room-secret"},
	})
	if err != nil {
		t.Fatalf("RunRoomDetailGet error: %v", err)
	}
	if result.Partial || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	data, _ := json.Marshal(result.Data)
	for _, forbidden := range []string{"token-room-secret", "not-allowed", "accessToken", `"attr"`, `"did"`, `"isBind"`, `"typeName"`, `"details"`, `"typeId"`, `"resId"`, `"params"`} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("room detail leaked %q: %s", forbidden, string(data))
		}
	}
	detail := result.Data.(map[string]any)["detail"].(map[string]any)
	if detail["id"] != "10" || detail["name"] != "灯光区" || detail["deviceCount"] != 1 {
		t.Fatalf("detail = %#v", detail)
	}
	devices := detail["devices"].([]any)
	device := devices[0].(map[string]any)
	if device["id"] != "5001" || device["name"] != "主灯" || device["capabilityPid"] != "198666" {
		t.Fatalf("device = %#v", device)
	}
	scenes := detail["scenes"].([]any)
	scene := scenes[0].(map[string]any)
	if scene["id"] != "7001" || scene["name"] != "观影模式" || scene["actionCount"] != 1 {
		t.Fatalf("scene = %#v", scene)
	}
}

func TestRoomReadonlyMissingContextDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	list, err := client.RunRoomList(context.Background(), MetadataReadonlyRequest{})
	if err != nil {
		t.Fatalf("RunRoomList error: %v", err)
	}
	if !list.Partial || list.APICalls != 0 || len(list.Warnings) != 1 || list.Warnings[0] != "house_context_missing" {
		t.Fatalf("list = %#v", list)
	}
	search, err := client.RunRoomSearch(context.Background(), MetadataReadonlyRequest{HouseID: "1001", Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("RunRoomSearch error: %v", err)
	}
	if !search.Partial || search.APICalls != 0 || len(search.Warnings) != 1 || search.Warnings[0] != "room_search_keyword_missing" {
		t.Fatalf("search = %#v", search)
	}
}
