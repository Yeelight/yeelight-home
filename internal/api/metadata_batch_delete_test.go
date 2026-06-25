package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetadataBatchDeleteClientFansOutToSingleDeleteAdapter(t *testing.T) {
	deleted := map[string]bool{}
	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			if deleted["room-1"] && deleted["room-2"] {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			rows := `[{"id":"room-1","name":"客厅"},{"id":"room-2","name":"卧室"}]`
			if deleted["room-1"] {
				rows = `[{"id":"room-2","name":"卧室"}]`
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":` + rows + `}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/room-1/w/info":
			if request.Method != http.MethodDelete {
				http.NotFound(writer, request)
				return
			}
			deleteCalls++
			deleted["room-1"] = true
			_, _ = writer.Write([]byte(`{"success":true}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/room-2/w/info":
			if request.Method != http.MethodDelete {
				http.NotFound(writer, request)
				return
			}
			deleteCalls++
			deleted["room-2"] = true
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewMetadataBatchDeleteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), MetadataBatchDeleteRequest{
		Kind:           MetadataBatchDeleteRoom,
		HouseID:        "house-1",
		VerifyAttempts: 1,
		VerifyInterval: time.Millisecond,
		Items: []MetadataBatchDeleteItem{
			{EntityID: "room-1"},
			{EntityID: "room-2"},
		},
		Credentials: MetadataDeleteCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if deleteCalls != 2 || result.ItemCount != 2 || !result.Verified || len(result.Results) != 2 {
		t.Fatalf("deleteCalls=%d result=%#v", deleteCalls, result)
	}
}
