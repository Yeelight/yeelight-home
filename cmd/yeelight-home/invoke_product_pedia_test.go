package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestInvokeProductPediaSearchUsesPediaEndpoint(t *testing.T) {
	var gotCall string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCall = request.Method + " " + request.URL.Path
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"total":1,"rows":[{"materialCode":"1-000003268","productName":"青空灯","productModel":"YP-0117","attachments":[{"id":15080,"bizId":2791,"bizType":"productSet","materialCode":"1-000003268","url":"https://rag-resources.yeelight.com/products/sku-res/1-000003268/split/1-000003268_split.pdf","type":"说明书","name":"青空灯说明书","sort":0,"createUid":0,"createTime":1744960450,"updateUid":0,"updateTime":1744960450}]}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newInvokeTestApp(t, "Bearer token-pedia-secret", "client-pedia-1", "")

	input := `{"contractVersion":"1.0","requestId":"req-pedia","locale":"zh-CN","utterance":"查一下青空灯产品资料和说明书","intent":"product.pedia.search","parameters":{"multiField":"青空灯"}}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotCall != "POST /apis/c/v1/pedia/product/r/search" || gotBody["multiField"] != "青空灯" {
		t.Fatalf("gotCall=%q gotBody=%#v", gotCall, gotBody)
	}
	if strings.Contains(stdout.String(), "token-pedia-secret") || strings.Contains(stderr.String(), "token-pedia-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" || response["traceId"] != "product-pedia-search-readonly" {
		t.Fatalf("response = %#v", response)
	}
	result := response["result"].(map[string]any)
	data := result["data"].(map[string]any)
	products := data["products"].([]any)
	product := products[0].(map[string]any)
	if product["productModel"] != "YP-0117" {
		t.Fatalf("product = %#v", product)
	}
	resources := product["resources"].(map[string]any)
	if resources["faqCandidateUrl"] != "https://rag-resources.yeelight.com/products/sku-res/1-000003268/faq/1-000003268/.json" {
		t.Fatalf("resources = %#v", resources)
	}
}

func TestModuleCommandProductSearchMapsKeyword(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"total":0,"rows":[]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer module-pedia-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"product", "search", "--keyword", "青空灯", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotBody["multiField"] != "青空灯" {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
}

func TestModuleCommandProductSearchMapsModelMaterialAndLimit(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"total":1,"rows":[{"materialCode":"1-000003268","productName":"青空灯"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer module-pedia-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "sg"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"product", "pedia", "--material-code", "1-000003268", "--product-model", "YP-0117", "--limit", "1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if gotBody["multiField"] != "1-000003268" {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "success" {
		t.Fatalf("response = %#v", response)
	}
}
