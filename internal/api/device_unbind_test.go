package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeviceUnbindClientUsesOptionsAndVerifiesRemoval(t *testing.T) {
	deviceVisible := true
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
		case "/apis/iot/v1/device/device-1/w/unbind":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			deviceVisible = false
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewDeviceUnbindClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), DeviceUnbindRequest{
		HouseID:          "200171",
		DeviceID:         "device-1",
		ClearMac:         true,
		UnbindRelDevices: true,
		VerifyAttempts:   1,
		Credentials:      DeviceUnbindCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["clearMac"] != true || writeBody["unbindRelDevices"] != true {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "entity.list" || result.DeviceID != "device-1" {
		t.Fatalf("result = %#v", result)
	}
}
