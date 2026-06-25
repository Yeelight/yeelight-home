package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestHomeLockClientLocksAllWithReadAfterWrite(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯"},{"id":"50018430","name":"筒灯"}]}}`))
		case "/apis/iot/v1/house/200171/lockall":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeLockClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeLockRequest{
		Kind:           HomeLockAll,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Credentials:    SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !slices.Contains(calls, "POST /apis/iot/v1/house/200171/lockall") {
		t.Fatalf("missing lock call: %#v", calls)
	}
	if !result.Verified || result.DeviceCount != 2 || result.VerifiedBy != "entity.list:house_accessible_after_write_ack" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeLockClientUnlocksAll(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/house/200171/unlockall":
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeLockClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeLockRequest{
		Kind:           HomeUnlockAll,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Credentials:    SpaceOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !slices.Contains(calls, "POST /apis/iot/v1/house/200171/unlockall") || result.Capability != "home.unlock_all" {
		t.Fatalf("calls=%#v result=%#v", calls, result)
	}
}
