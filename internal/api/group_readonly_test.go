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
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/group/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"userGroupId":22,"houseId":1001,"name":"二楼","roomIds":[12],"img":"group.png","secret":"not-allowed"}]}}`))
		case "/apis/iot/v2/thing/manage/house/1001/group/r/info/2/5":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":21,"houseId":1001,"name":"一楼","roomId":10,"componentId":2,"secret":"not-allowed"},{"id":22,"houseId":1001,"name":"二楼","roomId":12,"componentId":2,"secret":"not-allowed"}]}}`))
		case "/apis/iot/v2/thing/manage/house/1001/group/22/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":22,"houseId":1001,"name":"二楼","desc":"全屋二楼照明","icon":"layers","roomId":12,"cid":5,"details":[{"deviceId":"device-1"}],"localToken":"not-allowed"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{"groupName": "二楼", "pageNo": 2, "pageSize": 5},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-group-secret",
			ClientID:      "client-1",
		},
	}
	list, err := client.RunGroupList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunGroupList error: %v", err)
	}
	structure, err := client.RunGroupStructureList(context.Background(), request)
	if err != nil {
		t.Fatalf("RunGroupStructureList error: %v", err)
	}
	search, err := client.RunGroupSearch(context.Background(), request)
	if err != nil {
		t.Fatalf("RunGroupSearch error: %v", err)
	}
	detail, err := client.RunGroupDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "1001",
		Parameters:  map[string]any{"groupId": "22"},
		Credentials: request.Credentials,
	})
	if err != nil {
		t.Fatalf("RunGroupDetailGet error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/iot/v2/thing/manage/house/1001/group/r/info/2/5\nPOST /apis/iot/v1/group/r/all\nGET /apis/iot/v2/thing/manage/house/1001/group/r/info/2/5\nGET /apis/iot/v2/thing/manage/house/1001/group/22/r/info" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	for _, result := range []MetadataReadonlyResult{list, structure, search, detail} {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, _ := json.Marshal(result.Data)
		for _, forbidden := range []string{"token-group-secret", "not-allowed", "userGroupId", "img"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
	first := search.Data.(map[string]any)["groups"].([]any)[0].(map[string]any)
	if first["id"] != "22" || first["name"] != "二楼" {
		t.Fatalf("first group = %#v", first)
	}
	detailData := detail.Data.(map[string]any)["detail"].(map[string]any)
	if detailData["description"] != "全屋二楼照明" || detailData["icon"] != "layers" || detailData["roomId"] != "12" || detailData["componentId"] != "5" {
		t.Fatalf("detail metadata = %#v", detailData)
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
	detailMissingHouse, err := client.RunGroupDetailGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{"groupId": "22"}})
	if err != nil {
		t.Fatalf("RunGroupDetailGet missing house error: %v", err)
	}
	if !detailMissingHouse.Partial || detailMissingHouse.APICalls != 0 || len(detailMissingHouse.Warnings) != 1 || detailMissingHouse.Warnings[0] != "house_context_missing" {
		t.Fatalf("detailMissingHouse = %#v", detailMissingHouse)
	}
	detailMissingGroup, err := client.RunGroupDetailGet(context.Background(), MetadataReadonlyRequest{HouseID: "1001", Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("RunGroupDetailGet missing group error: %v", err)
	}
	if !detailMissingGroup.Partial || detailMissingGroup.APICalls != 0 || len(detailMissingGroup.Warnings) != 1 || detailMissingGroup.Warnings[0] != "group_context_missing" {
		t.Fatalf("detailMissingGroup = %#v", detailMissingGroup)
	}
}
