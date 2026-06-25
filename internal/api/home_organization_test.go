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
	sortReadCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/sort/r/getSort":
			sortReadCalls++
			if sortReadCalls < 2 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"typeId":2,"resId":50018330,"rank":1}]}`))
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
	if !result.Verified || result.VerifiedBy != "home.sort.configure_read_after_write" {
		t.Fatalf("result = %#v", result)
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

	_, err := NewHomeOrganizationClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), HomeOrganizationRequest{
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
	if err == nil || !strings.Contains(err.Error(), "write verification mismatch") {
		t.Fatalf("err = %v", err)
	}
}

func TestHomeOrganizationClientBatchAddsFavoritesWithSingleVerificationPass(t *testing.T) {
	var calls []string
	writeCount := 0
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
		case "/apis/iot/v1/favourite/w/insert":
			writeCount++
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
	if writeCount != 2 || result.APICalls != 4 || result.ItemCount != 2 || !result.Verified {
		t.Fatalf("writeCount=%d result=%#v calls=%#v", writeCount, result, calls)
	}
}
