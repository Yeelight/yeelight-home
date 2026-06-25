package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGroupReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	var gotSearchBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/group/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"userGroupId":21,"houseId":1001,"name":"一楼","roomIds":[10,11],"accessToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/group/r/1001/fuzzy":
			if err := json.NewDecoder(request.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":22,"houseId":1001,"nane":"二楼","roomIds":[12],"secret":"not-allowed"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{"name": "二", "pageNo": 2, "pageSize": 5},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-group-secret",
			ClientID:      "client-1",
		},
	}
	list, err := client.RunGroupList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunGroupList error: %v", err)
	}
	search, err := client.RunGroupSearch(context.Background(), request)
	if err != nil {
		t.Fatalf("RunGroupSearch error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "POST /apis/iot/v1/group/r/all\nPOST /apis/iot/v1/group/r/1001/fuzzy" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if gotSearchBody["fuzzyName"] != "二" || gotSearchBody["pageNo"] != float64(2) || gotSearchBody["pageSize"] != float64(5) {
		t.Fatalf("gotSearchBody = %#v", gotSearchBody)
	}
	for _, result := range []MetadataReadonlyResult{list, search} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, _ := json.Marshal(result.Data)
		for _, forbidden := range []string{"token-group-secret", "not-allowed"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
	first := search.Data.(map[string]any)["groups"].([]any)[0].(map[string]any)
	if first["id"] != "22" || first["name"] != "二楼" || first["roomCount"] != 1 {
		t.Fatalf("first group = %#v", first)
	}
}

func TestGroupReadonlyMissingContextDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	list, err := client.RunGroupList(context.Background(), MetadataReadonlyRequest{})
	if err != nil {
		t.Fatalf("RunGroupList error: %v", err)
	}
	if !list.Partial || list.APICalls != 0 || len(list.Warnings) != 1 || list.Warnings[0] != "house_context_missing" {
		t.Fatalf("list = %#v", list)
	}
	search, err := client.RunGroupSearch(context.Background(), MetadataReadonlyRequest{HouseID: "1001", Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("RunGroupSearch error: %v", err)
	}
	if !search.Partial || search.APICalls != 0 || len(search.Warnings) != 1 || search.Warnings[0] != "group_search_keyword_missing" {
		t.Fatalf("search = %#v", search)
	}
}
