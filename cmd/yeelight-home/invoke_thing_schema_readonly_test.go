package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeThingSchemaGetUsesCloudReadonlyAdapter(t *testing.T) {
	var gotCall string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/thing/schema/r/1001" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"pid":1001,"desc":"主灯产品","components":[{"cid":2001,"name":"灯"}],"localToken":"not-allowed"}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-schema-secret", "client-schema-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-thing-schema","locale":"zh-CN","utterance":"查看产品 1001 的物模型","intent":"thing.schema.get","parameters":{"houseId":"house-1","productId":"1001"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "GET /apis/iot/v1/thing/schema/r/1001" {
		t.Fatalf("gotCall = %q", gotCall)
	}
	for _, forbidden := range []string{"token-schema-secret", "not-allowed"} {
		if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "thing-schema-get-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	if result["cloudWrites"] != false {
		t.Fatalf("result = %#v", result)
	}
	data := result["data"].(map[string]any)
	if data["productId"] != "1001" || data["schema"] == nil {
		t.Fatalf("data = %#v", data)
	}
}

func TestInvokeThingSchemaEventListRequiresProductContextWithoutCloudCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-schema-secret", "client-schema-1", "house-1")

	input := `{"contractVersion":"1.0","requestId":"req-thing-events-missing","locale":"zh-CN","utterance":"查看产品事件","intent":"thing.schema.event.list","parameters":{"houseId":"house-1"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "partial" || response["traceId"] != "thing.schema.event.list-partial" {
		t.Fatalf("response = %#v", response)
	}
	warnings := response["warnings"].([]any)
	if len(warnings) != 1 || warnings[0] != "product_context_missing" {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestInvokeThingProductSchemaReadsUseCloudReadonlyAdapters(t *testing.T) {
	var gotCalls []string
	var gotQueries []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotQueries = append(gotQueries, request.URL.RawQuery)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/schema/product/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"desc":"主灯产品","version":1,"secret":"not-allowed"}]}`))
		case "/apis/iot/v3/thing/schema/product/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"desc":"主灯产品","version":2,"accessToken":"not-allowed"}]}`))
		case "/apis/iot/v3/thing/schema/product/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"pid":1001,"name":"主灯产品","version":2,"localToken":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-product-schema-secret", "client-schema-1", "house-1")

	inputs := []string{
		`{"contractVersion":"1.0","requestId":"req-product-v2","locale":"zh-CN","utterance":"查看产品 1001 的 v2 产品定义","intent":"thing.product.info.batch_get","parameters":{"productId":"1001"}}`,
		`{"contractVersion":"1.0","requestId":"req-product-v3","locale":"zh-CN","utterance":"查看产品 1001 的第 2 版产品定义","intent":"thing.product.info.v3.batch_get","parameters":{"productId":"1001","version":2}}`,
		`{"contractVersion":"1.0","requestId":"req-product-list-v3","locale":"zh-CN","utterance":"查看版本化产品列表","intent":"thing.product.list.v3","parameters":{"houseId":"house-1"}}`,
	}
	for _, input := range inputs {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		for _, forbidden := range []string{"token-product-schema-secret", "not-allowed"} {
			if strings.Contains(stdout.String(), forbidden) || strings.Contains(stderr.String(), forbidden) {
				t.Fatalf("output leaked %q: stdout=%s stderr=%s", forbidden, stdout.String(), stderr.String())
			}
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("invalid json response: %v", err)
		}
		if response["status"] != "success" {
			t.Fatalf("response = %#v", response)
		}
	}
	expectedCalls := []string{
		"GET /apis/iot/v2/thing/schema/product/r/info",
		"GET /apis/iot/v3/thing/schema/product/r/info",
		"GET /apis/iot/v3/thing/schema/product/r/list",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if !strings.Contains(gotQueries[0], "pids=1001") {
		t.Fatalf("unexpected v2 product query: %s", gotQueries[0])
	}
	if !strings.Contains(gotQueries[1], "pids=1001") || !strings.Contains(gotQueries[1], "version=2") {
		t.Fatalf("unexpected v3 product query: %s", gotQueries[1])
	}
}
