package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDestructiveDeleteClientRemovesDeviceWithVerification(t *testing.T) {
	deviceVisible := true
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			if deviceVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/device-1/w/info":
			deviceVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewDestructiveDeleteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), DestructiveDeleteRequest{
		Kind:           DestructiveDeleteDevice,
		HouseID:        "200171",
		EntityID:       "device-1",
		VerifyAttempts: 1,
		Credentials:    DestructiveDeleteCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Capability != "device.remove" || result.EntityType != "device" || result.EntityID != "device-1" || !result.Verified || result.VerifiedBy != "entity.list" {
		t.Fatalf("result = %#v", result)
	}
	if len(gotCalls) != 13 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestDestructiveDeleteClientDeletesGatewayWithListVerification(t *testing.T) {
	gatewayVisible := true
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"gateway-1","name":"网关"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/gateway/gateway-1/w/info":
			gatewayVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v2/thing/manage/house/200171/gateway/r/info/1/100":
			if gatewayVisible {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"gateway-1","name":"网关"}]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewDestructiveDeleteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), DestructiveDeleteRequest{
		Kind:           DestructiveDeleteGateway,
		HouseID:        "200171",
		EntityID:       "gateway-1",
		VerifyAttempts: 1,
		Credentials:    DestructiveDeleteCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Capability != "gateway.delete" || result.EntityType != "gateway" || result.EntityID != "gateway-1" || !result.Verified || result.VerifiedBy != "gateway.list" {
		t.Fatalf("result = %#v", result)
	}
}
