package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDeviceDetailGetProjectsSemanticFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/device-1/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"alias":"主灯","attr":{"p":1,"l":42,"ct":3000,"mac":"AA:BB:CC:DD","localToken":"secret"},"capability":"p,l,ct","connectType":1,"deviceId":"device-1","did":"raw-did","gatewayDeviceId":"gw-1","houseId":"house-1","isBind":1,"isVirtual":1,"name":"主灯","roomId":"room-1","shadow":{"properties":{"p":true,"l":58,"ct":3200,"o":true,"localToken":"secret"}},"typeName":"色温灯"}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-detail-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceDetailGet error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"brightness":58`, `"colorTemperature":3200`, `"online":true`, `"attributes"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected detail missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-detail-secret", "secret", "AA:BB:CC:DD", `"attr"`, `"did"`, `"isBind"`, `"typeName"`, `"p"`, `"l"`, `"ct"`, "raw-did"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected detail leaked %q: %s", forbidden, text)
		}
	}
}

func TestRunDeviceAttrListProjectsSemanticFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/r/attrs" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"attributes":[{"id":"device-1","p":1,"l":42,"ct":3000,"o":true,"mac":"AA:BB:CC:DD","ssid":"private-wifi","localToken":"secret","did":"raw-did","isBind":1}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceAttrList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-attr-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceAttrList error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"power":1`, `"brightness":42`, `"colorTemperature":3000`, `"online":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected attrs missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-attr-secret", "secret", "AA:BB:CC:DD", "private-wifi", `"p"`, `"l"`, `"ct"`, `"attr"`, `"did"`, `"isBind"`, "raw-did"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected attrs leaked %q: %s", forbidden, text)
		}
	}
}
