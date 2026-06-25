package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTransportUsesEndpointAndAuthorizationHeader(t *testing.T) {
	var gotPath string
	var gotAuthorization string
	var gotClientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":0,"data":{"homes":1}}`))
	}))
	defer server.Close()

	transport := NewHTTPTransport(Endpoint{Region: "custom", BaseURL: server.URL + "/apis/iot"}, StaticTokenSource("Bearer access-token"), server.Client())
	transport.ClientID = "client-123"
	response, err := transport.Call(context.Background(), Operation{
		SemanticOperation: "home.summary",
		Status:            "draft",
		Risk:              "R0",
		Path:              "/home/summary",
		Method:            "GET",
	}, Request{SemanticOperation: "home.summary"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if gotPath != "/apis/iot/home/summary" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAuthorization != "Bearer access-token" {
		t.Fatalf("authorization = %s", gotAuthorization)
	}
	if gotClientID != "client-123" {
		t.Fatalf("Client-Id = %s", gotClientID)
	}
	if response.Status != "success" {
		t.Fatalf("status = %s", response.Status)
	}
}

func TestHTTPTransportErrorDoesNotExposeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	transport := NewHTTPTransport(Endpoint{Region: "custom", BaseURL: server.URL}, StaticTokenSource("secret-token"), server.Client())
	_, err := transport.Call(context.Background(), Operation{
		SemanticOperation: "home.summary",
		Status:            "draft",
		Risk:              "R0",
		Path:              "/home/summary",
		Method:            "GET",
	}, Request{SemanticOperation: "home.summary"})
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked token: %v", err)
	}
}
