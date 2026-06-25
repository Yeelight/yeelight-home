package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdditionalReadonlyAdaptersProjectSafeData(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/scene/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":21,"houseId":1001,"roomId":10,"name":"电影","details":[{"secret":"nope"}],"params":"hidden"}]}}`))
		case "/apis/iot/v1/schedulejob/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":31,"houseId":1001,"name":"定时关灯","repeatType":1,"actions":[{"accessToken":"nope"}],"params":"hidden"}]}}`))
		case "/apis/iot/v1/messagecenter/r/messages":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":41,"title":"设备离线","content":"abcdefghijklmnopqrstuvwxyz","secret":"nope"}]}`))
		case "/apis/iot/v1/product-domain/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":51,"name":"照明","code":"lighting","token":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":61,"pid":100,"title":"如何重置","answer":"按住按键","password":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/61/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":61,"title":"如何重置","answer":"按住按键","authorization":"nope"}}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/faq-types":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"PRODUCT_USE_HELP","description":"产品使用帮助","enDescription":"Product help","secret":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/faq-item-types":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"TEXT","description":"文本","token":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/locales":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"code":"zh-CN","name":"简体中文","authorization":"nope"}]}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/page":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":62,"pid":100,"title":"分页 FAQ","answer":"分页回答","secret":"nope"}],"total":1}}`))
		case "/apis/iot/v1/platform/thing/product_faq/r/pageDetail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":63,"pid":100,"title":"分页详情 FAQ","items":[{"token":"nope"}],"answer":"详情回答"}],"total":1}}`))
		case "/apis/iot/v2/thing/schema/category/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":71,"name":"灯","status":1,"secret":"nope"}]}`))
		case "/apis/iot/v2/thing/schema/component/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":81,"name":"亮度","properties":[{"token":"nope"}]}]}`))
		case "/apis/iot/v2/thing/schema/component/r/81":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":81,"name":"亮度","properties":[{"token":"nope"}]}}`))
		case "/apis/iot/v2/thing/schema/property/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":91,"name":"开关","dataType":"bool","secret":"nope"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{
		HouseID: "1001",
		Parameters: map[string]any{
			"roomId": "10",
			"faqId":  "61",
			"cid":    "81",
		},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-additional-secret",
			ClientID:      "client-additional-1",
		},
	}

	results := []MetadataReadonlyResult{}
	for _, run := range []func(context.Context, MetadataReadonlyRequest) (MetadataReadonlyResult, error){
		client.RunSceneScopedList,
		client.RunScheduleJobList,
		client.RunMessageList,
		client.RunProductDomainList,
		client.RunProductFAQList,
		client.RunProductFAQDetailGet,
		client.RunProductFAQTypeList,
		client.RunProductFAQItemTypeList,
		client.RunProductFAQLocaleList,
		client.RunProductFAQPageList,
		client.RunProductFAQPageDetailList,
		client.RunThingCategoryList,
		client.RunThingComponentList,
		client.RunThingComponentGet,
		client.RunThingPropertyList,
	} {
		result, err := run(context.Background(), request)
		if err != nil {
			t.Fatalf("readonly adapter error: %v", err)
		}
		results = append(results, result)
	}

	wantCalls := []string{
		"POST /apis/iot/v1/scene/r/list",
		"POST /apis/iot/v1/schedulejob/r/list",
		"GET /apis/iot/v1/messagecenter/r/messages",
		"POST /apis/iot/v1/product-domain/r/list",
		"POST /apis/iot/v1/platform/thing/product_faq/r/list",
		"GET /apis/iot/v1/platform/thing/product_faq/r/61/detail",
		"GET /apis/iot/v1/platform/thing/product_faq/r/faq-types",
		"GET /apis/iot/v1/platform/thing/product_faq/r/faq-item-types",
		"GET /apis/iot/v1/platform/thing/product_faq/r/locales",
		"POST /apis/iot/v1/platform/thing/product_faq/r/page",
		"POST /apis/iot/v1/platform/thing/product_faq/r/pageDetail",
		"GET /apis/iot/v2/thing/schema/category/r/list",
		"GET /apis/iot/v2/thing/schema/component/r/list",
		"GET /apis/iot/v2/thing/schema/component/r/81",
		"GET /apis/iot/v2/thing/schema/property/r/list",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	data, _ := json.Marshal(results)
	for _, forbidden := range []string{"token-additional-secret", "accessToken", "authorization", "password", "secret", "nope", "hidden"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, string(data))
		}
	}
	if results[0].Capability != "scene.scoped.list" || results[1].Capability != "schedule_job.list" || results[2].Capability != "message.list" {
		t.Fatalf("unexpected capabilities: %#v", results[:3])
	}
	jobs := results[1].Data.(map[string]any)["scheduleJobs"].([]any)
	if jobs[0].(map[string]any)["actionCount"] != 1 {
		t.Fatalf("schedule projection = %#v", jobs)
	}
}

func TestAdditionalReadonlyAdaptersRequireContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	request := MetadataReadonlyRequest{Parameters: map[string]any{}}

	for _, run := range []struct {
		name string
		fn   func(context.Context, MetadataReadonlyRequest) (MetadataReadonlyResult, error)
		want string
	}{
		{name: "scene scoped", fn: client.RunSceneScopedList, want: "scene_scope_context_missing"},
		{name: "schedule job", fn: client.RunScheduleJobList, want: "house_context_missing"},
		{name: "faq detail", fn: client.RunProductFAQDetailGet, want: "faq_context_missing"},
		{name: "component", fn: client.RunThingComponentGet, want: "component_context_missing"},
	} {
		result, err := run.fn(context.Background(), request)
		if err != nil {
			t.Fatalf("%s error: %v", run.name, err)
		}
		if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != run.want {
			t.Fatalf("%s result = %#v", run.name, result)
		}
	}
}
