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

func TestRoomCreateClientCreatesAndVerifiesMissingRoom(t *testing.T) {
	var calls []string
	var createBody map[string]any
	roomListCalls := 0
	server := newRoomCreateServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":401999,"name":"Codex Plan Room"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":401999}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewRoomCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), RoomCreateRequest{
		HouseID:        "200171",
		Name:           "Codex Plan Room",
		Description:    "Runtime configure lane room",
		VerifyAttempts: 1,
		Credentials:    RoomCreateCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if createBody["name"] != "Codex Plan Room" || createBody["desc"] != "Runtime configure lane room" || createBody["houseId"] != float64(200171) {
		t.Fatalf("createBody = %#v", createBody)
	}
	if !result.Created || !result.Verified || result.RoomID != "401999" || result.VerifiedBy != "room_list" || result.APICalls != 9 {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "PUT /apis/iot/v2/thing/manage/house/200171/room/w/create") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestRoomCreateClientReusesExistingRoomByName(t *testing.T) {
	var calls []string
	server := newRoomCreateServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-existing","name":"Codex Plan Room"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewRoomCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), RoomCreateRequest{
		HouseID:     "house-1",
		Name:        "Codex Plan Room",
		Credentials: RoomCreateCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Created || !result.Verified || result.RoomID != "room-existing" || result.APICalls != 7 {
		t.Fatalf("result = %#v", result)
	}
	if slices.Contains(calls, "PUT /apis/iot/v2/thing/manage/house/house-1/room/w/create") {
		t.Fatalf("unexpected create call: %#v", calls)
	}
}

func TestRoomCreateClientFindsExistingRoomOnLaterPage(t *testing.T) {
	var calls []string
	server := newRoomCreateServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[` + testEntityRows("room", "房间", 1, 100) + `]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/2/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-existing","name":"Codex Plan Room"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/w/create":
			t.Fatal("room create should not write when name exists on later page")
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewRoomCreateClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), RoomCreateRequest{
		HouseID:     "house-1",
		Name:        "Codex Plan Room",
		Credentials: RoomCreateCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/2/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/2/100",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Created || !result.Verified || result.RoomID != "room-existing" || result.APICalls != 9 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRoomCreateClientRejectsMissingFields(t *testing.T) {
	client := NewRoomCreateClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.Run(context.Background(), RoomCreateRequest{Name: "Room", Credentials: RoomCreateCredentials{Authorization: "secret-token"}})
	if err == nil || !strings.Contains(err.Error(), "house id is required") {
		t.Fatalf("err = %v", err)
	}
	_, err = client.Run(context.Background(), RoomCreateRequest{HouseID: "house-1", Credentials: RoomCreateCredentials{Authorization: "secret-token"}})
	if err == nil || !strings.Contains(err.Error(), "room name is required") {
		t.Fatalf("err = %v", err)
	}
}

func newRoomCreateServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		handler(writer, request)
	}))
}
