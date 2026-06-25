package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeSpaceConfigurationClientUpdatesHomeWithDetailVerification(t *testing.T) {
	var writeBody map[string]any
	detailCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/r/info":
			detailCalls++
			name := "旧家庭"
			if detailCalls > 1 {
				name = "新家庭"
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"200171","name":"` + name + `"}}`))
		case "/apis/iot/v2/thing/manage/house/200171/w/modify":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode home update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeSpaceConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeSpaceConfigurationRequest{
		Kind:           HomeSpaceHomeUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"name":    "新家庭",
			"desc":    "主要住宅",
		},
		Credentials: HomeSpaceConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["houseId"] != nil || writeBody["id"] != float64(200171) || writeBody["name"] != "新家庭" || writeBody["desc"] != "主要住宅" {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.Capability != "home.update" || result.VerifiedBy != "home.detail.get" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeSpaceConfigurationClientBatchCreatesRooms(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := newHomeSpaceEntityServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
				return true
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"room-1","name":"书房"},{"id":"room-2","name":"茶室"}]}}`))
			return true
		case "/apis/iot/v2/thing/manage/house/200171/room/w/batch_create":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room batch create body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
			return true
		default:
			return false
		}
	})
	defer server.Close()

	result, err := NewHomeSpaceConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeSpaceConfigurationRequest{
		Kind:           HomeSpaceRoomBatchCreate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"rooms": []any{
				map[string]any{"name": "书房", "desc": "阅读"},
				map[string]any{"name": "茶室"},
			},
		},
		Credentials: HomeSpaceConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	rooms := writeBody["rooms"].([]any)
	if len(rooms) != 2 || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
}

func TestHomeSpaceConfigurationClientBatchUpdatesRoomsFromStoredPayload(t *testing.T) {
	var writeBody map[string]any
	roomListCalls := 0
	server := newHomeSpaceEntityServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			roomListCalls++
			if roomListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
				return true
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"会客厅"},{"id":"401392","name":"卧室"}]}}`))
			return true
		case "/apis/iot/v1/room/w/batchupdate":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room batch update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
			return true
		default:
			return false
		}
	})
	defer server.Close()

	result, err := NewHomeSpaceConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeSpaceConfigurationRequest{
		Kind:           HomeSpaceRoomBatchUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"rooms": []any{
				map[string]any{"roomId": "401391", "name": "会客厅"},
				map[string]any{"roomId": "401392"},
			},
		},
		Credentials: HomeSpaceConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	rooms := writeBody["rooms"].([]any)
	second := rooms[1].(map[string]any)
	if second["name"] != "卧室" || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
}

func TestHomeSpaceConfigurationClientConfiguresRoomAreaWithAckVerification(t *testing.T) {
	var writeBody map[string]any
	server := newHomeSpaceEntityServer(t, func(writer http.ResponseWriter, request *http.Request) bool {
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/room/401391/w/areas":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode room area body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
			return true
		default:
			return false
		}
	})
	defer server.Close()

	result, err := NewHomeSpaceConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeSpaceConfigurationRequest{
		Kind:           HomeSpaceRoomAreaConfigure,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"roomId":         "401391",
			"addAreaList":    []any{"300001"},
			"removeAreaList": []any{"300002"},
		},
		Credentials: HomeSpaceConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if writeBody["id"] != float64(401391) || writeBody["roomId"] != nil || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
}

func TestHomeSpaceConfigurationClientRejectsUnknownRoomArea(t *testing.T) {
	server := newHomeSpaceEntityServer(t, func(http.ResponseWriter, *http.Request) bool { return false })
	defer server.Close()
	_, err := NewHomeSpaceConfigurationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeSpaceConfigurationRequest{
		Kind:    HomeSpaceRoomAreaConfigure,
		HouseID: "200171",
		Payload: map[string]any{
			"roomId":      "401391",
			"addAreaList": []any{"missing-area"},
		},
		Credentials: HomeSpaceConfigurationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "area missing-area not found before write") {
		t.Fatalf("err = %v", err)
	}
}

func newHomeSpaceEntityServer(t *testing.T, handler func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"300001","name":"一楼"},{"id":"300002","name":"二楼"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100":
			if handler(writer, request) {
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"401391","name":"客厅"},{"id":"401392","name":"卧室"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":"50018330","name":"主灯"}]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		default:
			if handler(writer, request) {
				return
			}
			http.NotFound(writer, request)
		}
	}))
}
