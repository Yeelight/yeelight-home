package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestDevSeedClientReusesExistingHouseByName(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-existing","name":"Codex Dev Test Home"}]}`))
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:          "Codex Dev Test Home",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureHouse error: %v", err)
	}
	if !slices.Equal(calls, []string{"POST /apis/iot/v1/house/r/list"}) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Created || !result.Verified || result.HouseID != "house-existing" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientCreatesAndVerifiesMissingHouse(t *testing.T) {
	var calls []string
	var createBody map[string]any
	listCalls := 0
	v2PageCalls := 0
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
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"houseId":"house-created","houseName":"Codex Dev Test Home"}]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			v2PageCalls++
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"house-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:           "Codex Dev Test Home",
		Description:    "Runtime dev verification fixture",
		AreaCode:       "CN",
		AreaName:       "中国",
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureHouse error: %v", err)
	}
	expectedCalls := []string{
		"POST /apis/iot/v1/house/r/list",
		"GET /apis/iot/v2/thing/manage/house/r/info/1/100",
		"PUT /apis/iot/v2/thing/manage/house/w/create",
		"POST /apis/iot/v1/house/r/list",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if createBody["name"] != "Codex Dev Test Home" || createBody["desc"] != "Runtime dev verification fixture" {
		t.Fatalf("createBody = %#v", createBody)
	}
	if v2PageCalls != 1 {
		t.Fatalf("v2PageCalls = %d", v2PageCalls)
	}
	if !result.Created || !result.Verified || result.HouseID != "house-created" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientVerifiesCreatedHouseThroughV2PageWhenV1ListIsEmpty(t *testing.T) {
	var calls []string
	v2PageCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"data":"house-created"}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			v2PageCalls++
			if v2PageCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"house-created","name":"Codex Dev Test Home"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:           "Codex Dev Test Home",
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureHouse error: %v", err)
	}
	expectedCalls := []string{
		"POST /apis/iot/v1/house/r/list",
		"GET /apis/iot/v2/thing/manage/house/r/info/1/100",
		"PUT /apis/iot/v2/thing/manage/house/w/create",
		"POST /apis/iot/v1/house/r/list",
		"GET /apis/iot/v2/thing/manage/house/r/info/1/100",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if !result.Created || !result.Verified || result.HouseID != "house-created" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientVerifiesCreatedHouseThroughScopedEntityList(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"data":"house-created"}`))
		case "/apis/iot/v2/thing/manage/house/house-created/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-created/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:           "Codex Dev Test Home",
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureHouse error: %v", err)
	}
	if !result.Created || !result.Verified || result.HouseID != "house-created" || result.VerifiedBy != "entity_list" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "POST /apis/iot/v2/thing/manage/house/house-created/device/r/info/1/100") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientReusesCandidateHouseIDWhenScopedEntityListVerifies(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-candidate/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-candidate/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-candidate/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-candidate/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-candidate/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:             "Codex Dev Test Home",
		CandidateHouseID: "house-candidate",
		AllowWriteDev:    true,
		Credentials:      DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureHouse error: %v", err)
	}
	if result.Created || !result.Verified || result.HouseID != "house-candidate" || result.VerifiedBy != "entity_list_candidate" {
		t.Fatalf("result = %#v", result)
	}
	for _, call := range calls {
		if strings.Contains(call, "/w/create") || strings.Contains(call, "/house/r/list") {
			t.Fatalf("unexpected create/list call: %#v", calls)
		}
	}
}

func TestDevSeedClientRequiresDevWriteGate(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:        "Codex Dev Test Home",
		Credentials: DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil {
		t.Fatal("expected allow-write-dev error")
	}
}

func TestDevSeedClientReportsUnknownWriteResultWhenCreateCannotBeVerified(t *testing.T) {
	listCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			listCalls++
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"code":"200","message":"ok","data":null}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	_, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:           "Codex Dev Test Home",
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil {
		t.Fatal("expected unknown write result error")
	}
	if !strings.Contains(err.Error(), "unknown write result") || !strings.Contains(err.Error(), "dataType=<nil>") {
		t.Fatalf("err = %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("listCalls = %d", listCalls)
	}
}

func TestDevSeedClientRejectsNonDevEndpoint(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil)
	_, err := client.EnsureHouse(context.Background(), DevSeedHouseRequest{
		Name:          "Codex Dev Test Home",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil {
		t.Fatal("expected non-dev endpoint error")
	}
}
