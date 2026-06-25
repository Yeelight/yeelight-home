package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeSummaryClientReturnsRedactedHouseSummaries(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"house-1","houseName":"默认家庭"}]}}`))
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), HomeSummaryCredentials{
		Authorization: "secret-token",
		ClientID:      "client-1",
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if result.HouseCount != 1 {
		t.Fatalf("HouseCount = %d", result.HouseCount)
	}
	if result.Houses[0].ID != "house-1" || result.Houses[0].Name != "默认家庭" {
		t.Fatalf("Houses = %#v", result.Houses)
	}
}

func TestHomeSummaryClientRejectsBusinessFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"message":"denied"}`))
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	if _, err := client.Run(context.Background(), HomeSummaryCredentials{Authorization: "secret-token"}); err == nil {
		t.Fatal("expected business failure")
	}
}

func TestHomeSummaryClientRunListReturnsHouseStatsProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"houseId":1001,"name":"常住房","img":"home.png","description":"主住宅","areaCode":"CN-310000","areaName":"上海","roomNum":3,"deviceNum":12,"gatewayNum":2,"sceneNum":5,"automationNum":4,"areaNum":1,"createUid":1234567890}]}}`))
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunList(context.Background(), HomeSummaryCredentials{Authorization: "secret-token"})
	if err != nil {
		t.Fatalf("RunList error: %v", err)
	}
	if result.APICalls != 1 || result.HouseCount != 1 {
		t.Fatalf("result = %#v", result)
	}
	house := result.Houses[0]
	if house.ID != "1001" || house.Name != "常住房" || house.Icon != "home.png" || house.Desc != "主住宅" || house.AreaCode != "CN-310000" || house.AreaName != "上海" {
		t.Fatalf("house = %#v", house)
	}
	if house.Counts["rooms"] != 3 || house.Counts["devices"] != 12 || house.Counts["gateways"] != 2 || house.Counts["automations"] != 4 {
		t.Fatalf("counts = %#v", house.Counts)
	}
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "1234567890") {
		t.Fatalf("result leaked uid: %s", string(data))
	}
}

func TestHomeSummaryClientRunListParsesDataListWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"house-1","name":"默认家庭","roomNum":1}]}}`))
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunList(context.Background(), HomeSummaryCredentials{Authorization: "secret-token"})
	if err != nil {
		t.Fatalf("RunList error: %v", err)
	}
	if result.HouseCount != 1 || result.Houses[0].ID != "house-1" || result.Houses[0].Name != "默认家庭" {
		t.Fatalf("result = %#v", result)
	}
	if result.Houses[0].Counts["rooms"] != 1 {
		t.Fatalf("counts = %#v", result.Houses[0].Counts)
	}
}

func TestHomeSummaryClientRunSearchUsesFuzzyName(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/fuzzy" {
			http.NotFound(writer, request)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":1002,"name":"父母家","desc":"共享家庭","icon":"parent.png","roomNum":2,"deviceNum":8,"gatewayNum":1,"sceneNum":3,"areaNum":1}]}}`))
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunSearch(context.Background(), map[string]any{"name": "父母", "pageNo": 2, "pageSize": 5}, HomeSummaryCredentials{Authorization: "secret-token"})
	if err != nil {
		t.Fatalf("RunSearch error: %v", err)
	}
	if gotBody["fuzzyName"] != "父母" || gotBody["pageNo"] != float64(2) || gotBody["pageSize"] != float64(5) {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	if result.HouseCount != 1 || result.Houses[0].ID != "1002" || result.Houses[0].Counts["scenes"] != 3 {
		t.Fatalf("result = %#v", result)
	}
}
