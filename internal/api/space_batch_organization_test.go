package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSpaceBatchOrganizationClientMovesDevicesWithReadAfterWrite(t *testing.T) {
	var writeBody map[string]any
	deviceListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			deviceListCalls++
			if deviceListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401391"},{"id":"50018430","name":"筒灯","roomId":"401391"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401392"},{"id":"50018430","name":"筒灯","roomId":"401392"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/room/w/batch-modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode batch move body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewSpaceBatchOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceBatchOrganizationRequest{
		Kind:           SpaceBatchDeviceMoveRoom,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"items": map[string]any{
				"50018330": "401392",
				"50018430": "401392",
			},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	items := writeBody["items"].(map[string]any)
	if writeBody["houseId"] != float64(200171) || items["50018330"] != "401392" || items["50018430"] != "401392" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.ItemCount != 2 || result.VerifiedBy != "entity.list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSpaceBatchOrganizationClientRejectsTooManyItems(t *testing.T) {
	items := map[string]any{}
	for index := 0; index < 21; index++ {
		items["device-"+string(rune('a'+index))] = "401392"
	}
	_, err := NewSpaceBatchOrganizationClient(Endpoint{Region: "dev", BaseURL: "http://example.invalid"}, nil).Run(context.Background(), SpaceBatchOrganizationRequest{
		Kind:    SpaceBatchDeviceMoveRoom,
		HouseID: "200171",
		Payload: map[string]any{"items": items},
		Credentials: SpaceOrganizationCredentials{
			Authorization: "secret-token",
			ClientID:      "client-1",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "limit exceeded") {
		t.Fatalf("err = %v", err)
	}
}

func TestSpaceBatchOrganizationClientRejectsUnknownDeviceOrRoom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯","roomId":"401391"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewSpaceBatchOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), SpaceBatchOrganizationRequest{
		Kind:           SpaceBatchDeviceMoveRoom,
		HouseID:        "200171",
		VerifyAttempts: 1,
		VerifyInterval: time.Millisecond,
		Payload: map[string]any{
			"items": []any{
				map[string]any{"deviceId": "50018330", "roomId": "room-missing"},
			},
		},
		Credentials: SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "room room-missing not found before write") {
		t.Fatalf("err = %v", err)
	}
}
