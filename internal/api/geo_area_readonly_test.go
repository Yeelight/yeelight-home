package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeoAreaReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/area/r/0/children":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":1,"name":"北京","fullname":"北京-北京","code":"CN-110000","leaf":true,"fetchWeather":true,"latitude":"39.90","longitude":"116.40","accessToken":"not-allowed"}]}`))
		case "/apis/iot/v1/area/r/areas":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":2,"name":"上海","fullName":"上海-上海","code":"CN-310000","secret":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		Parameters: map[string]any{"name": "上海"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-geo-area-secret",
			ClientID:      "client-1",
		},
	}

	children, err := client.RunGeoAreaChildrenList(context.Background(), request)
	if err != nil {
		t.Fatalf("children err = %v", err)
	}
	search, err := client.RunGeoAreaSearch(context.Background(), request)
	if err != nil {
		t.Fatalf("search err = %v", err)
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/iot/v1/area/r/0/children\nPOST /apis/iot/v1/area/r/areas" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range []MetadataReadonlyResult{children, search} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		for _, forbidden := range []string{"token-geo-area-secret", "not-allowed"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
}

func TestGeoAreaReadonlyMissingContextDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	invalidParent, err := client.RunGeoAreaChildrenList(context.Background(), MetadataReadonlyRequest{
		Parameters: map[string]any{"parentId": "abc"},
	})
	if err != nil {
		t.Fatalf("children err = %v", err)
	}
	if !invalidParent.Partial || invalidParent.APICalls != 0 || len(invalidParent.Warnings) != 1 || invalidParent.Warnings[0] != "geo_area_parent_id_invalid" {
		t.Fatalf("invalidParent = %#v", invalidParent)
	}

	search, err := client.RunGeoAreaSearch(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("search err = %v", err)
	}
	if !search.Partial || search.APICalls != 0 || len(search.Warnings) != 1 || search.Warnings[0] != "geo_area_name_missing" {
		t.Fatalf("search = %#v", search)
	}
}
