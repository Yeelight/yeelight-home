package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIVoiceProductListReturnsRedactedProjection(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/ai/voice/product/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[1001,1002],"accessToken":"not-allowed"}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunAIVoiceProductList(context.Background(), MetadataReadonlyRequest{
		HouseID: "house-1",
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-ai-voice-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("ai voice product list err = %v", err)
	}
	if gotCall != "GET /apis/iot/v1/ai/voice/product/r/list" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	if result.Partial || result.APICalls != 1 || result.Capability != "ai_voice.product.list" {
		t.Fatalf("result = %#v", result)
	}
	data, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	for _, forbidden := range []string{"token-ai-voice-secret", "not-allowed"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, string(data))
		}
	}
}
