package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeCreateClientCreatesAndVerifiesByName(t *testing.T) {
	var calls []string
	listCalls := 0
	var createBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			listCalls++
			if listCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"home-created","name":"新家"}]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"home-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, nil).Run(context.Background(), HomeCreateRequest{
		Name:           "新家",
		Description:    "描述",
		Icon:           "icon-home",
		AreaCode:       "CN-310000",
		AreaName:       "上海",
		VerifyAttempts: 1,
		Credentials: HomeCreateCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Created || !result.Verified || result.HouseID != "home-created" || result.VerifiedBy != "home.summary" {
		t.Fatalf("result = %#v", result)
	}
	if result.APICalls != 4 {
		t.Fatalf("apiCalls = %d calls=%#v", result.APICalls, calls)
	}
	if createBody["name"] != "新家" || createBody["desc"] != "描述" || createBody["icon"] != "icon-home" || createBody["areaCode"] != "CN-310000" || createBody["areaName"] != "上海" {
		t.Fatalf("create body = %#v", createBody)
	}
}

func TestHomeCreateClientReusesExistingHomeByName(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"home-1","name":"已有家庭"}]}`))
	}))
	defer server.Close()

	result, err := NewHomeCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, nil).Run(context.Background(), HomeCreateRequest{
		Name: "已有家庭",
		Credentials: HomeCreateCredentials{
			Authorization: "Bearer token-secret",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Created || !result.Verified || result.HouseID != "home-1" || len(calls) != 1 {
		t.Fatalf("result=%#v calls=%#v", result, calls)
	}
}

func TestHomeCreateClientFallsBackToEntityListVerification(t *testing.T) {
	listCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/house/r/list":
			listCalls++
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case request.URL.Path == "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case request.URL.Path == "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"data":"home-created"}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, nil).Run(context.Background(), HomeCreateRequest{
		Name:           "新家",
		VerifyAttempts: 1,
		Credentials: HomeCreateCredentials{
			Authorization: "Bearer token-secret",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Created || !result.Verified || result.HouseID != "home-created" || result.VerifiedBy != "entity_list" {
		t.Fatalf("result = %#v", result)
	}
}
