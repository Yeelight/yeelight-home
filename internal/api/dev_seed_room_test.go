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

func TestDevSeedClientReusesExistingRoomByName(t *testing.T) {
	var calls []string
	server := newSeedRoomServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-existing","name":"Codex Dev Test Room"}]}}`))
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

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureRoom(context.Background(), DevSeedRoomRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Room",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureRoom error: %v", err)
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/manage/house/house-1/area/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/device/r/info/1/100",
		"GET /apis/iot/v2/thing/manage/house/house-1/group/r/info/1/100",
		"POST /apis/iot/v2/thing/manage/house/house-1/scene/r/info/1/100",
		"POST /apis/iot/v1/automations/r/list",
		"GET /apis/iot/v2/thing/manage/house/house-1/room/r/info/1/100",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if result.Created || !result.Verified || result.RoomID != "room-existing" || result.VerifiedBy != "room_list" {
		t.Fatalf("result = %#v", result)
	}
}

func TestDevSeedClientCreatesAndVerifiesMissingRoom(t *testing.T) {
	var calls []string
	var createBody map[string]any
	roomListCalls := 0
	server := newSeedRoomServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200176/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":200180,"name":"Codex Dev Test Room"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/device/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200176/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200176/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200176/room/w/create":
			if request.Method != http.MethodPut {
				t.Fatalf("method = %s", request.Method)
			}
			if err := json.NewDecoder(request.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":200180}`))
		default:
			http.NotFound(writer, request)
		}
	})
	defer server.Close()

	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.EnsureRoom(context.Background(), DevSeedRoomRequest{
		HouseID:        "200176",
		Name:           "Codex Dev Test Room",
		Description:    "Runtime dev verification room",
		AllowWriteDev:  true,
		VerifyAttempts: 1,
		Credentials:    DevSeedCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("EnsureRoom error: %v", err)
	}
	if createBody["name"] != "Codex Dev Test Room" || createBody["desc"] != "Runtime dev verification room" || createBody["houseId"] != float64(200176) {
		t.Fatalf("createBody = %#v", createBody)
	}
	if !result.Created || !result.Verified || result.RoomID != "200180" || result.VerifiedBy != "room_list" {
		t.Fatalf("result = %#v", result)
	}
	if !slices.Contains(calls, "PUT /apis/iot/v2/thing/manage/house/200176/room/w/create") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestDevSeedClientRequiresRoomHouseIDAndWriteGate(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil)
	_, err := client.EnsureRoom(context.Background(), DevSeedRoomRequest{
		Name:        "Codex Dev Test Room",
		Credentials: DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "--allow-write-dev") {
		t.Fatalf("err = %v", err)
	}

	_, err = client.EnsureRoom(context.Background(), DevSeedRoomRequest{
		Name:          "Codex Dev Test Room",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "house id is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestDevSeedClientRejectsRoomSeedForNonDevEndpoint(t *testing.T) {
	client := NewDevSeedClient(Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil)
	_, err := client.EnsureRoom(context.Background(), DevSeedRoomRequest{
		HouseID:       "house-1",
		Name:          "Codex Dev Test Room",
		AllowWriteDev: true,
		Credentials:   DevSeedCredentials{Authorization: "secret-token"},
	})
	if err == nil || !strings.Contains(err.Error(), "only allowed for dev") {
		t.Fatalf("err = %v", err)
	}
}

func newSeedRoomServer(t *testing.T, handler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		handler(writer, request)
	}))
}
