package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProductPediaSearchProjectsProductsAndResources(t *testing.T) {
	var gotAuthorization string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/apis/c/v1/pedia/product/r/search" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		gotAuthorization = request.Header.Get("Authorization")
		if err := json.NewDecoder(request.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"success": true,
			"code": "200",
			"data": {
				"total": 176,
				"rows": [{
					"id": 2791,
					"materialCode": "1-000003268",
					"pid": 198666,
					"productName": "Yeelight Pro Nightingale青空灯夙夜版",
					"productBrand": "易来/YEELIGHT",
					"productModel": "YP-0117",
					"productSku": "YLP-Nightingale青空灯-夙夜版",
					"productSpu": "Nightingale青空灯",
					"productLine": "11自营产品线",
					"productCategoryName": "18青空灯",
					"productLargeClass": "01青空灯灯具",
					"productSmallClass": "01智能青空灯",
					"productShortName": "青空灯夙夜版",
					"productSeries": "S级",
					"barcode": "6924922224129",
					"modelNo": "yeelink.light.lamp24",
					"baseUnit": "PC",
					"productDeclareNo": "9405110000",
					"productDeclareName": "LED青空灯",
					"productDeclareUnit": "个",
					"productStatusName": "停售",
					"productSaleTypeName": "电竞系列",
					"quotationTypeDesc": "电竞",
					"productTypeName": "成品",
					"isSupportYeelightPro": "02支持Yeelight Pro",
					"isSupportHomekit": null,
					"pediaDisplay": true,
					"accessToken": "not-allowed",
					"attachments": [
						{"id":7320,"bizId":2791,"bizType":"productSet","materialCode":"1-000003268","url":"https://cloud-bj-resources.yeelight.com/prod/product-images/a.png","type":"缩略图","name":null,"sort":0,"createUid":256204,"createTime":1657779003,"updateUid":256204,"updateTime":1657779003,"accessToken":"not-allowed"},
						{"id":15080,"bizId":2791,"bizType":"productSet","materialCode":"1-000003268","url":"https://rag-resources.yeelight.com/products/sku-res/1-000003268/split/1-000003268_split.pdf","type":"说明书","name":"青空灯说明书","sort":0,"createUid":0,"createTime":1744960450,"updateUid":0,"updateTime":1744960450}
					]
				}]
			}
		}`))
	}))
	defer server.Close()

	result, err := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).RunProductPediaSearch(context.Background(), MetadataReadonlyRequest{
		Parameters: map[string]any{"keyword": "青空灯"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer pedia-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("RunProductPediaSearch error = %v", err)
	}
	if gotAuthorization != "Bearer pedia-secret" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotBody["multiField"] != "青空灯" {
		t.Fatalf("body = %#v", gotBody)
	}
	if result.Capability != "product.pedia.search" || result.APICalls != 1 || result.Partial {
		t.Fatalf("result = %#v", result)
	}
	data := result.Data.(map[string]any)
	if data["total"] != 176 || data["returned"] != 1 {
		t.Fatalf("data = %#v", data)
	}
	products := data["products"].([]any)
	product := products[0].(map[string]any)
	if product["materialCode"] != "1-000003268" || product["productName"] == "" || product["pediaDisplay"] != true {
		t.Fatalf("product = %#v", product)
	}
	for key, want := range map[string]any{
		"productBrand":        "易来/YEELIGHT",
		"productModel":        "YP-0117",
		"productSku":          "YLP-Nightingale青空灯-夙夜版",
		"productSpu":          "Nightingale青空灯",
		"productLine":         "11自营产品线",
		"productCategoryName": "18青空灯",
		"productLargeClass":   "01青空灯灯具",
		"productSmallClass":   "01智能青空灯",
		"productShortName":    "青空灯夙夜版",
		"productSeries":       "S级",
		"barcode":             "6924922224129",
		"modelNo":             "yeelink.light.lamp24",
		"baseUnit":            "PC",
		"productDeclareNo":    "9405110000",
		"productDeclareName":  "LED青空灯",
		"productDeclareUnit":  "个",
		"productStatusName":   "停售",
		"productSaleTypeName": "电竞系列",
		"quotationTypeDesc":   "电竞",
		"productTypeName":     "成品",
	} {
		if product[key] != want {
			t.Fatalf("product[%s] = %#v, want %#v", key, product[key], want)
		}
	}
	if _, ok := product["isSupportHomekit"]; !ok || product["isSupportHomekit"] != nil {
		t.Fatalf("isSupportHomekit should preserve explicit null: %#v", product)
	}
	if _, ok := product["accessToken"]; ok {
		t.Fatalf("sensitive field leaked: %#v", product)
	}
	productAttachments := product["attachments"].([]any)
	if len(productAttachments) != 2 {
		t.Fatalf("product attachments = %#v", productAttachments)
	}
	resources := product["resources"].(map[string]any)
	if resources["manualCandidateUrl"] != "https://rag-resources.yeelight.com/products/sku-res/1-000003268/split/1-000003268_split.pdf" {
		t.Fatalf("resources = %#v", resources)
	}
	if resources["faqCandidateUrl"] != "https://rag-resources.yeelight.com/products/sku-res/1-000003268/faq/1-000003268/.json" {
		t.Fatalf("resources = %#v", resources)
	}
	manualAttachments := resources["manualAttachments"].([]any)
	if len(manualAttachments) != 1 {
		t.Fatalf("manualAttachments = %#v", manualAttachments)
	}
	attachments := resources["attachments"].([]any)
	firstAttachment := attachments[0].(map[string]any)
	if firstAttachment["id"] != float64(7320) || firstAttachment["bizId"] != float64(2791) || firstAttachment["bizType"] != "productSet" || firstAttachment["createUid"] != float64(256204) {
		t.Fatalf("attachment = %#v", firstAttachment)
	}
	if _, ok := firstAttachment["name"]; !ok || firstAttachment["name"] != nil {
		t.Fatalf("attachment null name not preserved: %#v", firstAttachment)
	}
	if _, ok := firstAttachment["accessToken"]; ok {
		t.Fatalf("sensitive attachment field leaked: %#v", firstAttachment)
	}
}

func TestProductPediaSearchUsesUtteranceFallbackAndRequiresQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["multiField"] != "查一下青空灯说明书" {
			t.Fatalf("body = %#v", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunProductPediaSearch(context.Background(), MetadataReadonlyRequest{
		Utterance:  "查一下青空灯说明书",
		Parameters: map[string]any{},
	})
	if err != nil {
		t.Fatalf("RunProductPediaSearch error = %v", err)
	}
	if result.APICalls != 1 || result.Data.(map[string]any)["query"] != "查一下青空灯说明书" {
		t.Fatalf("result = %#v", result)
	}

	missing, err := client.RunProductPediaSearch(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}})
	if err != nil {
		t.Fatalf("missing query error = %v", err)
	}
	if !missing.Partial || missing.APICalls != 0 || !strings.Contains(strings.Join(missing.Warnings, ","), "product_pedia_query_missing") {
		t.Fatalf("missing = %#v", missing)
	}
}
