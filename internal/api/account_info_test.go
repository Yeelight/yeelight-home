package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountInfoClientReturnsRedactedSummary(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/account/user/info" {
			http.NotFound(writer, request)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code": "200",
			"data": map[string]any{
				"userId":   "1234567890",
				"nickname": "测试用户",
				"phone":    "13800138000",
				"email":    "user@example.com",
				"token":    "should-not-appear",
			},
		})
	}))
	defer server.Close()

	result, err := NewAccountInfoClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), AccountInfoCredentials{
		Authorization: "Bearer token-account-secret",
		ClientID:      "client-account-1",
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer token-account-secret" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if result.Region != "dev" || result.APICalls != 1 || result.RawShape != "map[string]interface {}" {
		t.Fatalf("result = %#v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"token-account-secret", "should-not-appear", "13800138000", "user@example.com", "1234567890"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("redacted result leaked %q: %s", forbidden, text)
		}
	}
	if result.Summary["displayName"] != "测试用户" || result.Summary["phoneMasked"] != "***8000" || result.Summary["emailMasked"] != "u***@example.com" {
		t.Fatalf("summary = %#v", result.Summary)
	}
}
