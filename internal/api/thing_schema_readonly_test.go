package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestThingSchemaReadonlyAdaptersReturnProjectedSchema(t *testing.T) {
	var gotCalls []string
	var gotQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotQueries = append(gotQueries, request.URL.RawQuery)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/thing/schema/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"name":"主灯产品","category":"light","accessToken":"not-allowed"}]}`))
		case "/apis/iot/v1/thing/schema/r/list/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"desc":"主灯产品","capability":"101","components":[{"cid":2001,"name":"灯","properties":[{"propId":"brightness","desc":"亮度","valueRange":{"min":1,"max":100,"step":1}}]}],"events":[{"eventId":10,"name":"状态变化"}],"localToken":"not-allowed"}]}`))
		case "/apis/iot/v1/thing/schema/r/1001":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"pid":1001,"desc":"主灯产品","capability":"101","components":[{"cid":2001,"name":"灯","supportActions":[{"actionName":"set_power"}]}],"secret":"not-allowed"}}`))
		case "/apis/iot/v1/thing/schema/r/getEvents/1001":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"eventId":10,"eventType":"status","name":"状态变化","desc":"状态变化事件","accessToken":"not-allowed"}]}`))
		case "/apis/iot/v2/thing/schema/product/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"desc":"主灯产品","version":1,"components":[{"cid":2001,"name":"灯"}],"secret":"not-allowed"}]}`))
		case "/apis/iot/v3/thing/schema/product/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"desc":"主灯产品","version":2,"events":[{"eventId":10,"name":"状态变化"}],"localToken":"not-allowed"}]}`))
		case "/apis/iot/v3/thing/schema/product/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"name":"主灯产品","version":2,"accessToken":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{"productId": "1001"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-schema-secret", ClientID: "client-1"},
	}

	results := []MetadataReadonlyResult{}
	for _, run := range []func() (MetadataReadonlyResult, error){
		func() (MetadataReadonlyResult, error) {
			return client.RunThingSchemaList(context.Background(), request)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunThingSchemaDetailList(context.Background(), request)
		},
		func() (MetadataReadonlyResult, error) { return client.RunThingSchemaGet(context.Background(), request) },
		func() (MetadataReadonlyResult, error) {
			return client.RunThingSchemaEventList(context.Background(), request)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunThingProductInfoBatchGet(context.Background(), request)
		},
		func() (MetadataReadonlyResult, error) {
			versioned := request
			versioned.Parameters = map[string]any{"productId": "1001", "version": 2}
			return client.RunThingProductInfoV3BatchGet(context.Background(), versioned)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunThingProductListV3(context.Background(), request)
		},
	} {
		result, err := run()
		if err != nil {
			t.Fatalf("run err = %v", err)
		}
		results = append(results, result)
	}

	expectedCalls := []string{
		"GET /apis/iot/v1/thing/schema/r/list",
		"GET /apis/iot/v1/thing/schema/r/list/detail",
		"GET /apis/iot/v1/thing/schema/r/1001",
		"GET /apis/iot/v1/thing/schema/r/getEvents/1001",
		"GET /apis/iot/v2/thing/schema/product/r/info",
		"GET /apis/iot/v3/thing/schema/product/r/info",
		"GET /apis/iot/v3/thing/schema/product/r/list",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if !strings.Contains(gotQueries[4], "pids=1001") {
		t.Fatalf("unexpected v2 product info query: %s", gotQueries[4])
	}
	if !strings.Contains(gotQueries[5], "pids=1001") || !strings.Contains(gotQueries[5], "version=2") {
		t.Fatalf("unexpected v3 product info query: %s", gotQueries[5])
	}
	for _, result := range results {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		text := string(data)
		for _, forbidden := range []string{"not-allowed", "token-schema-secret"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, text)
			}
		}
		if !strings.Contains(text, "cloud-v1") || !strings.Contains(text, "ttlSeconds") {
			t.Fatalf("missing schema cache metadata: %s", text)
		}
	}
}

func TestThingSchemaGetRequiresProductContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunThingSchemaGet(context.Background(), MetadataReadonlyRequest{
		Parameters:  map[string]any{},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-schema-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("schema get err = %v", err)
	}
	if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != "product_context_missing" {
		t.Fatalf("result = %#v", result)
	}

	v3, err := client.RunThingProductInfoV3BatchGet(context.Background(), MetadataReadonlyRequest{
		Parameters:  map[string]any{"productId": "1001"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-schema-secret", ClientID: "client-1"},
	})
	if err != nil {
		t.Fatalf("v3 product info err = %v", err)
	}
	if !v3.Partial || v3.APICalls != 0 || len(v3.Warnings) != 1 || v3.Warnings[0] != "product_version_context_missing" {
		t.Fatalf("v3 = %#v", v3)
	}
}

func TestThingSchemaReadonlyBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunThingProductListV3(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-schema-secret"},
	})
	if err != nil {
		t.Fatalf("RunThingProductListV3 error = %v", err)
	}
	if !result.Partial || result.APICalls != 1 || result.Capability != "thing.product.list.v3" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}
