package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallJSONWrapsTopLevelArrayResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"pid":1001,"name":"主灯产品"}]`))
	}))
	defer server.Close()

	response, err := callJSON(context.Background(), server.Client(), http.MethodGet, server.URL, nil, requestCredentials{})
	if err != nil {
		t.Fatalf("callJSON error = %v", err)
	}
	rows, ok := response["data"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("wrapped data = %#v", response)
	}
}

func TestCallJSONDoesNotDefaultToSaasBizType(t *testing.T) {
	var gotBizType string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotBizType = request.Header.Get("bizType")
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"ok": true}})
	}))
	defer server.Close()

	if _, err := callJSON(context.Background(), server.Client(), http.MethodPost, server.URL, map[string]any{}, requestCredentials{}); err != nil {
		t.Fatalf("callJSON error = %v", err)
	}
	if gotBizType != "" {
		t.Fatalf("default bizType = %q, want backend default", gotBizType)
	}
}

func TestCallJSONAllowsExplicitBizType(t *testing.T) {
	var gotBizType string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotBizType = request.Header.Get("bizType")
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{"ok": true}})
	}))
	defer server.Close()

	if _, err := callJSON(context.Background(), server.Client(), http.MethodPost, server.URL, map[string]any{}, requestCredentials{BizType: "2"}); err != nil {
		t.Fatalf("callJSON error = %v", err)
	}
	if gotBizType != "2" {
		t.Fatalf("explicit bizType = %q", gotBizType)
	}
}
