package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanelReadAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/panel/r/list/house-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"panel-1","name":"面板","mac":"AA:BB:CC:DD","localToken":"not-allowed"}]}`))
		case "/apis/iot/v1/panel/r/button/info/panel-1/click":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"click":[{"id":"button-1","name":"单击","accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/ai/house-1/control/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"screen-1","list":[{"resId":"device-1","resType":2}]}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	credentials := MetadataReadonlyCredentials{Authorization: "Bearer token-panel-secret", ClientID: "client-1"}

	panelList, err := client.RunPanelList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: credentials,
	})
	if err != nil {
		t.Fatalf("panel list err = %v", err)
	}
	buttons, err := client.RunPanelButtonTypeGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "panel-1",
		Parameters:  map[string]any{"buttonType": "click"},
		Credentials: credentials,
	})
	if err != nil {
		t.Fatalf("panel button err = %v", err)
	}
	screenControls, err := client.RunScreenControlList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{},
		Credentials: credentials,
	})
	if err != nil {
		t.Fatalf("screen controls err = %v", err)
	}

	if len(gotCalls) != 3 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range []MetadataReadonlyResult{panelList, buttons, screenControls} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		for _, forbidden := range []string{"not-allowed", "AA:BB:CC:DD", "token-panel-secret"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
}

func TestPanelButtonTypeRequiresTypeWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunPanelButtonTypeGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "panel-1",
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-panel-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("panel button err = %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "button_type_context_missing" {
		t.Fatalf("result = %#v", result)
	}
}
