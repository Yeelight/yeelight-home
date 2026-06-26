package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSmokeClientCallsAccountAndHouseListWithRedactedSummary(t *testing.T) {
	var calls []string
	var authorization string
	var clientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		authorization = request.Header.Get("Authorization")
		clientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_ = json.NewEncoder(writer).Encode(map[string]any{"code": "200", "data": map[string]any{"nickname": "测试用户"}})
		case "/apis/iot/v1/house/r/all":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"list": []any{}}})
		case "/apis/iot/v1/house/r/list":
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": []map[string]any{{"id": "house-1", "name": "默认家庭"}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSmokeClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SmokeCredentials{
		Authorization: "Bearer token-secret-123456",
		ClientID:      "client-123",
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %#v", calls)
	}
	if calls[0] != "GET /apis/account/user/info" {
		t.Fatalf("account call = %q", calls[0])
	}
	if calls[1] != "POST /apis/iot/v1/house/r/all" || calls[2] != "POST /apis/iot/v1/house/r/list" {
		t.Fatalf("house calls = %#v", calls)
	}
	if authorization != "Bearer token-secret-123456" {
		t.Fatalf("Authorization = %q", authorization)
	}
	if clientID != "client-123" {
		t.Fatalf("Client-Id = %q", clientID)
	}
	if !result.AccountOK || !result.HouseListOK || result.HouseCount != 1 || result.HouseListSource != "/v1/house/r/list" || result.HouseListAPICalls != 2 {
		t.Fatalf("result = %#v", result)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if string(data) == "" || strings.Contains(string(data), "token-secret-123456") {
		t.Fatalf("smoke result leaked token: %s", string(data))
	}
}
