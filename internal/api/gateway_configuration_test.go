package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayConfigurationClientUpdatesGatewayWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	detailReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/r/info":
			detailReads++
			if detailReads < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"旧网关"}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"新网关"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewGatewayConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), GatewayConfigurationRequest{
		HouseID:        "200171",
		GatewayID:      "gateway-1",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":   float64(200171),
			"gatewayId": "gateway-1",
			"name":      "新网关",
			"roomIds":   []any{"room-1"},
		},
		Credentials: GatewayConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["gatewayId"] != nil || writeBody["houseId"] != nil || writeBody["name"] != "新网关" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "gateway.detail.get" || result.GatewayID != "gateway-1" {
		t.Fatalf("result = %#v", result)
	}
}
