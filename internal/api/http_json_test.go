package api

import (
	"context"
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
