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
