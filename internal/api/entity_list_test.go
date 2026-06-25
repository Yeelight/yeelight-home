package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestEntityListClientReturnsRedactedEntitiesForHouse(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"area-1","name":"一楼","roomIds":["room-1"]}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"客厅","deviceNum":2}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-1","name":"主灯","roomId":"room-1","online":true}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"group-1","name":"灯组","cid":7}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"scene-1","name":"晚安"}]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"auto-1","name":"回家开灯","status":0}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewEntityListClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), EntityListRequest{
		HouseID: "house-1",
		Credentials: EntityListCredentials{
			Authorization: "secret-token",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Total != 6 {
		t.Fatalf("Total = %d", result.Total)
	}
	if result.Counts["area"] != 1 || result.Counts["room"] != 1 || result.Counts["device"] != 1 || result.Counts["group"] != 1 || result.Counts["scene"] != 1 || result.Counts["automation"] != 1 {
		t.Fatalf("Counts = %#v", result.Counts)
	}
	if result.APICalls != 6 {
		t.Fatalf("APICalls = %d", result.APICalls)
	}
	if result.Entities[0].Type != "area" || result.Entities[0].Name != "一楼" {
		t.Fatalf("first entity = %#v", result.Entities[0])
	}
	if result.Entities[2].Type != "device" || result.Entities[2].RoomID != "room-1" {
		t.Fatalf("device entity = %#v", result.Entities[2])
	}
}

func TestEntityListClientUsesHouseListWhenHouseIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-1","name":"默认家庭"}]}`))
	}))
	defer server.Close()

	client := NewEntityListClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), EntityListRequest{
		Credentials: EntityListCredentials{Authorization: "secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Total != 1 || result.Entities[0].Type != "home" || result.Entities[0].Name != "默认家庭" {
		t.Fatalf("result = %#v", result)
	}
	if result.APICalls != 1 {
		t.Fatalf("APICalls = %d", result.APICalls)
	}
}

func TestEntityListClientPaginatesHouseScopedEntities(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[` + testEntityRows("device", "灯", 1, 100) + `]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/2/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"device-101","name":"灯 101","roomId":"room-1"}]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewEntityListClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), EntityListRequest{
		HouseID:     "house-1",
		Credentials: EntityListCredentials{Authorization: "secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/2/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Counts["device"] != 101 || result.Total != 101 {
		t.Fatalf("result = %#v", result)
	}
	if result.APICalls != 7 {
		t.Fatalf("APICalls = %d", result.APICalls)
	}
}

func TestEntityListClientReportsBusinessFailureDiagnostics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":false,"code":"40301","message":"no permission","data":null}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewEntityListClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	_, err := client.Run(context.Background(), EntityListRequest{
		HouseID: "house-1",
		Credentials: EntityListCredentials{
			Authorization: "secret-token",
			ClientID:      "client-1",
		},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "entityType=device") ||
		!strings.Contains(err.Error(), "code=40301") ||
		!strings.Contains(err.Error(), "message=no permission") ||
		!strings.Contains(err.Error(), "dataType=<nil>") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func testEntityRows(idPrefix string, namePrefix string, first int, count int) string {
	var builder strings.Builder
	for index := 0; index < count; index++ {
		if index > 0 {
			builder.WriteString(",")
		}
		id := first + index
		_, _ = fmt.Fprintf(&builder, `{"id":"%s-%d","name":"%s %d","roomId":"room-1"}`, idPrefix, id, namePrefix, id)
	}
	return builder.String()
}
