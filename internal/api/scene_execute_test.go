package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSceneExecuteClientRunsOpenControlScene(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID: "house-1",
		SceneID: "scene-1",
		Credentials: SceneExecuteCredentials{
			Authorization: "scene-secret-token",
			ClientID:      "client-scene-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer scene-secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-scene-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if result.Region != "dev" || result.HouseID != "house-1" || result.SceneID != "scene-1" || result.Source != "open_control_scene_endpoint" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSceneExecuteClientReportsBusinessFailureWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":"40301","message":"no permission","data":null}`))
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "code=40301") || !strings.Contains(err.Error(), "message=no permission") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "scene-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}
