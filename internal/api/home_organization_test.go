package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestHomeOrganizationClientAddsFavoriteWithReadAfterWrite(t *testing.T) {
	var calls []string
	var writeBody map[string]any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
		case "/apis/iot/v1/favourite/w/insert":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite insert body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"fav-1"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteAdd,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"typeId":  2,
			"resId":   float64(50018330),
			"rank":    1,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	expectedCalls := []string{
		"POST /apis/iot/v1/favourite/r/all",
		"POST /apis/iot/v1/favourite/w/insert",
		"POST /apis/iot/v1/favourite/r/all",
	}
	if !slices.Equal(calls, expectedCalls) {
		t.Fatalf("calls = %#v", calls)
	}
	if writeBody["resId"] != float64(50018330) || writeBody["houseId"] != float64(200171) || writeBody["rank"] != float64(1) {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	if !result.Verified || result.VerifiedBy != "favorite.add_read_after_write" || result.APICalls != 3 {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeOrganizationClientVerifiesFavoriteNestedDeviceResponse(t *testing.T) {
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"id":"50018330","deviceId":"50018330","houseId":"200171","rank":1}],"meshgroups":[],"userscenes":[]}}`))
		case "/apis/iot/v1/favourite/w/insert":
			_, _ = writer.Write([]byte(`{"success":true,"data":"fav-1"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteAdd,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"typeId":  2,
			"resId":   float64(50018330),
			"rank":    1,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !result.Verified || result.VerifiedBy != "favorite.add_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeOrganizationClientUpdatesFavoriteWithoutLeakingFavoriteIDInBody(t *testing.T) {
	var writeBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":8}]}`))
		case "/apis/iot/v1/favourite/fav-1/w/update":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":    float64(200171),
			"favoriteId": "fav-1",
			"typeId":     2,
			"resId":      float64(50018330),
			"rank":       8,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if _, exists := writeBody["favoriteId"]; exists {
		t.Fatalf("writeBody leaked favoriteId = %#v", writeBody)
	}
	if writeBody["rank"] != float64(8) || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
}

func TestHomeOrganizationClientUpdatesFavoriteByResourceWithMergedBatchUpdate(t *testing.T) {
	var writeBody []any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 3 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":"50018330","houseId":"200171","rank":1},{"deviceId":"50018331","houseId":"200171","rank":2}],"meshgroups":[],"userscenes":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":"50018330","houseId":"200171","rank":7},{"deviceId":"50018331","houseId":"200171","rank":2}],"meshgroups":[],"userscenes":[]}}`))
		case "/apis/iot/v1/favourite/w/batchupdate":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite update body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteUpdate,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"typeId":  2,
			"resId":   float64(50018330),
			"rank":    7,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
	foundUpdated := false
	foundPreserved := false
	for _, raw := range writeBody {
		row := raw.(map[string]any)
		switch row["resId"] {
		case float64(50018330):
			foundUpdated = row["rank"] == float64(7)
		case float64(50018331):
			foundPreserved = row["rank"] == float64(2)
		}
	}
	if !foundUpdated || !foundPreserved {
		t.Fatalf("writeBody = %#v", writeBody)
	}
}

func TestHomeOrganizationClientDeletesFavoriteWithReadAfterWrite(t *testing.T) {
	deleteCalls := 0
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/favourite/fav-1/w/delete":
			deleteCalls++
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteDelete,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId":    float64(200171),
			"favoriteId": "fav-1",
			"typeId":     2,
			"resId":      float64(50018330),
			"rank":       1,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if deleteCalls != 1 || !result.Verified || result.VerifiedBy != "favorite.delete_read_after_write" {
		t.Fatalf("deleteCalls=%d result=%#v", deleteCalls, result)
	}
}

func TestHomeOrganizationClientDeletesFavoriteByResourceWhenFavoriteIDMissing(t *testing.T) {
	var writeBody []any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[{"deviceId":"50018330","houseId":"200171","rank":1}],"meshgroups":[],"userscenes":[]}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"devices":[],"meshgroups":[],"userscenes":[]}}`))
		case "/apis/iot/v1/favourite/w/batchdelete":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite delete body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteDelete,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"typeId":  2,
			"resId":   float64(50018330),
			"rank":    1,
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 1 || writeBody[0].(map[string]any)["resId"] != float64(50018330) || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
}

func TestHomeOrganizationClientBatchDeletesFavoritesWithSingleVerificationPass(t *testing.T) {
	var writeBody []any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1},{"id":"fav-2","houseId":200171,"typeId":6,"resId":"700001","rank":2}]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/favourite/w/batchdelete":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite batch delete body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteBatchDelete,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"items": []any{
				map[string]any{"houseId": float64(200171), "favoriteId": "fav-1", "typeId": 2, "resId": float64(50018330), "rank": 1},
				map[string]any{"houseId": float64(200171), "favoriteId": "fav-2", "typeId": 6, "resId": "700001", "rank": 2},
			},
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 2 || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v", writeBody, result)
	}
	if _, leaked := writeBody[0].(map[string]any)["deleteTarget"]; leaked {
		t.Fatalf("writeBody leaked preview fields: %#v", writeBody)
	}
}

func TestHomeOrganizationClientConfiguresHomeSortWithArrayBody(t *testing.T) {
	var writeBody []any
	nodeReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/node/r/1/401391/device":
			if request.Header.Get("houseId") != "200171" || request.Header.Get("house-id") != "200171" {
				t.Fatalf("sort readback missing house headers: houseId=%q house-id=%q", request.Header.Get("houseId"), request.Header.Get("house-id"))
			}
			nodeReadCalls++
			if nodeReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"name":"主灯","roomId":401391,"rank":1}]}`))
		case "/apis/iot/v2/thing/manage/house/200171/area/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/group/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/200171/scene/r/info/1/100",
			"/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/200171/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":50018330,"name":"主灯","roomId":401391}]}}`))
		case "/apis/iot/v1/sort/r/getSort":
			t.Fatalf("device-room sort should use node sorted device readback before getSort fallback")
		case "/apis/iot/v1/sort/200171/w/1/401391/add":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode sort body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationSortConfigure,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"type":   "1",
			"target": "401391",
			"roomId": "401391",
			"typeId": 2,
			"resId":  float64(50018330),
			"items": []any{
				map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 1},
			},
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 1 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	row := writeBody[0].(map[string]any)
	if row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)] != float64(2) ||
		row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] != float64(50018330) ||
		row[semantic.FieldRank] != float64(1) {
		t.Fatalf("writeBody row = %#v", row)
	}
	for _, leaked := range []string{semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldType, semantic.FieldTarget, semantic.FieldTargetType, semantic.FieldTargetID, semantic.FieldTargetName} {
		if _, ok := row[leaked]; ok {
			t.Fatalf("sort write body leaked %s: %#v", leaked, row)
		}
	}
	if !result.Verified || result.VerifiedBy != "home.sort.configure_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBuildHomeSortAddBodyAcceptsPublicEntityItems(t *testing.T) {
	body, err := buildHomeSortAddBody([]any{
		map[string]any{
			semantic.FieldEntityType: "device",
			semantic.FieldID:         "50018330",
			semantic.FieldRank:       1,
		},
	})
	if err != nil {
		t.Fatalf("buildHomeSortAddBody error: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("body = %#v", body)
	}
	row := body[0].(map[string]any)
	if row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)] != homeSortGroupTypeDevice ||
		row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] != "50018330" ||
		row[semantic.FieldRank] != 1 {
		t.Fatalf("row = %#v", row)
	}
	for _, leaked := range []string{semantic.FieldEntityType, semantic.FieldID, semantic.FieldTargetID, semantic.FieldTargetType} {
		if _, ok := row[leaked]; ok {
			t.Fatalf("sort write body leaked public helper field %s: %#v", leaked, row)
		}
	}
}

func TestHomeOrganizationClientVerifiesRoomSceneSortWithSpecificEndpoint(t *testing.T) {
	var writeBody []any
	sceneReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/room/scene":
			sceneReadCalls++
			if sceneReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"roomId":401391,"sceneOrder":{"1005999":3}}]}`))
		case "/apis/iot/v1/sort/r/getSort":
			t.Fatalf("scene-room sort should use room scene readback before getSort fallback")
		case "/apis/iot/v1/sort/200171/w/2/401391/add":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode sort body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationSortConfigure,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"type":   "2",
			"target": "401391",
			"roomId": "401391",
			"typeId": 6,
			"resId":  "1005999",
			"items": []any{
				map[string]any{"typeId": 6, "resId": "1005999", "rank": 3},
			},
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 1 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
	row := writeBody[0].(map[string]any)
	if row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)] != float64(6) ||
		row[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] != "1005999" ||
		row[semantic.FieldRank] != float64(3) {
		t.Fatalf("writeBody row = %#v", row)
	}
	for _, leaked := range []string{semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldType, semantic.FieldTarget, semantic.FieldTargetType, semantic.FieldTargetID, semantic.FieldTargetName} {
		if _, ok := row[leaked]; ok {
			t.Fatalf("sort write body leaked %s: %#v", leaked, row)
		}
	}
	if !result.Verified || result.VerifiedBy != "home.sort.configure_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeOrganizationClientRejectsHomeSortWhenReadEndpointFails(t *testing.T) {
	var writeBody []any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/getSort":
			_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
		case "/apis/iot/v1/sort/200171/w/1/401391/add":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode sort body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	_, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationSortConfigure,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"type":   "1",
			"target": "401391",
			"roomId": "401391",
			"items": []any{
				map[string]any{"typeId": 2, "resId": float64(50018330), "rank": 1},
			},
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err == nil || !strings.Contains(err.Error(), "home sort list returned non-success") {
		t.Fatalf("err = %v", err)
	}
	if len(writeBody) != 0 {
		t.Fatalf("writeBody = %#v", writeBody)
	}
}

func TestHomeOrganizationClientReportsVerificationMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v1/favourite/w/insert":
			_, _ = writer.Write([]byte(`{"success":true,"data":"fav-1"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteAdd,
		HouseID:        "200171",
		VerifyAttempts: 1,
		VerifyInterval: time.Millisecond,
		Payload: map[string]any{
			"houseId": float64(200171),
			"typeId":  2,
			"resId":   float64(50018330),
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Verified || result.Warning != "write_verification_mismatch" || result.VerifiedBy != "favorite.add_read_after_write" {
		t.Fatalf("result = %#v", result)
	}
}

func TestHomeOrganizationClientBatchAddsFavoritesWithSingleVerificationPass(t *testing.T) {
	var calls []string
	var writeBody []any
	favoriteListCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/favourite/r/all":
			favoriteListCalls++
			if favoriteListCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"fav-1","houseId":200171,"typeId":2,"resId":50018330,"rank":1},{"id":"fav-2","houseId":200171,"typeId":6,"resId":"700001","rank":2}]}`))
		case "/apis/iot/v1/favourite/w/batchinsert":
			if err := json.NewDecoder(request.Body).Decode(&writeBody); err != nil {
				t.Fatalf("decode favorite batch insert body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
		Kind:           HomeOrganizationFavoriteBatchAdd,
		HouseID:        "200171",
		VerifyAttempts: 1,
		Payload: map[string]any{
			"houseId": float64(200171),
			"items": []any{
				map[string]any{"houseId": float64(200171), "typeId": 2, "resId": float64(50018330), "rank": 1},
				map[string]any{"houseId": float64(200171), "typeId": 6, "resId": "700001", "rank": 2},
			},
		},
		Credentials: HomeOrganizationCredentials{Authorization: "secret-token", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(writeBody) != 2 || result.APICalls != 3 || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeBody=%#v result=%#v calls=%#v", writeBody, result, calls)
	}
}
