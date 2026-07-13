package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestDeviceListReadonlyReturnsRedactedProjection(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":31,"did":9001,"pid":101,"type":1,"name":"主灯","alias":"客厅主灯","img":"light.png","houseId":1001,"roomId":10,"capability":"p,l,ct","online":false,"localToken":"not-allowed","mac":"AA:BB:CC:DD","deviceKey":"secret-key","shadow":{"p":true},"attr":{"secret":"nope"}},{"deviceId":32,"name":"网关","type":0,"deviceIds":[31],"roomIds":[10,11]}],"meshgroups":[{"meshGroupId":41,"name":"筒灯组","deviceIds":[31,33],"secret":"nope"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceList(context.Background(), MetadataReadonlyRequest{
		HouseID: "1001",
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-device-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("RunDeviceList error: %v", err)
	}
	if gotCall != "POST /apis/iot/v1/device/r/all" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if result.Partial || result.APICalls != 1 || result.Capability != "device.list" {
		t.Fatalf("result = %#v", result)
	}
	data, _ := json.Marshal(result.Data)
	for _, forbidden := range []string{"token-device-secret", "not-allowed", "AA:BB:CC:DD", "secret-key", "shadow", "attr", "nope"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, string(data))
		}
	}
	resultData := result.Data.(map[string]any)
	devices := resultData[semantic.FieldDevices].([]any)
	first := devices[0].(map[string]any)
	if first[semantic.FieldID] != "31" || first[semantic.FieldName] != "主灯" || first[semantic.FieldRoomID] != "10" || first[semantic.FieldCapability] != "p,l,ct" {
		t.Fatalf("first device = %#v", first)
	}
	if online, ok := first[semantic.FieldOnline].(bool); !ok || online {
		t.Fatalf("first device online = %#v, device = %#v", first[semantic.FieldOnline], first)
	}
	gateway := devices[1].(map[string]any)
	if gateway[semantic.FieldChildDeviceCount] != 1 {
		t.Fatalf("gateway = %#v", gateway)
	}
	groups := resultData[semantic.FieldGroups].([]any)
	if groups[0].(map[string]any)[semantic.FieldDeviceCount] != 2 {
		t.Fatalf("groups = %#v", groups)
	}
}

func TestDeviceListReadonlyMissingHouseDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceList(context.Background(), MetadataReadonlyRequest{})
	if err != nil {
		t.Fatalf("RunDeviceList error: %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "house_context_missing" {
		t.Fatalf("result = %#v", result)
	}
}
