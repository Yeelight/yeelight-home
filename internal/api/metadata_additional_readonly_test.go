package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/semantic"
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
			"roomId":      "10",
			"faqId":       "61",
			"componentId": "81",
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

func TestThingComponentGetResolvesNaturalComponentName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/schema/component/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":81,"name":"亮度","properties":[{"token":"nope"}]}]}`))
		case "/apis/iot/v2/thing/schema/component/r/81":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"id":81,"name":"亮度","properties":[{"token":"nope"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunThingComponentGet(context.Background(), MetadataReadonlyRequest{
		Parameters: map[string]any{semantic.FieldComponentName: "亮度"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-component-name-secret",
			ClientID:      "client-component-name-1",
		},
	})
	if err != nil {
		t.Fatalf("component get error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/iot/v2/thing/schema/component/r/list\nGET /apis/iot/v2/thing/schema/component/r/81" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if result.Partial || result.APICalls != 1 || result.Capability != "thing.component.get" {
		t.Fatalf("result = %#v", result)
	}
	data := result.Data.(map[string]any)
	component := data["component"].(map[string]any)
	if component[semantic.FieldID] != "81" || component[semantic.FieldName] != "亮度" {
		t.Fatalf("component = %#v", component)
	}
}

func TestThingComponentGetTreatsNonNumericComponentIDAsName(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/schema/component/r/list" {
			t.Fatalf("natural component id should not be sent as detail path: %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())

	result, err := client.RunThingComponentGet(context.Background(), MetadataReadonlyRequest{
		Parameters: map[string]any{semantic.FieldComponentID: "light"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-component-natural-secret",
			ClientID:      "client-component-natural-1",
		},
	})
	if err != nil {
		t.Fatalf("component get error: %v", err)
	}
	if strings.Join(gotCalls, "\n") != "GET /apis/iot/v2/thing/schema/component/r/list" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if !result.Partial || result.APICalls != 1 || len(result.Warnings) == 0 || result.Warnings[0] != "component_not_found_or_ambiguous" {
		t.Fatalf("result = %#v", result)
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

func TestSceneScopedListBusinessErrorReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/scene/r/list" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误","data":{"details":"hidden"}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunSceneScopedList(context.Background(), MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{"roomId": "10"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-additional-secret",
			ClientID:      "client-additional-1",
		},
	})
	if err != nil {
		t.Fatalf("RunSceneScopedList error = %v", err)
	}
	if !result.Partial || result.Capability != "scene.scoped.list" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_business_response_not_success" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if result.Data != nil {
		t.Fatalf("partial business result should not expose raw data: %#v", result.Data)
	}
}

func TestSceneScopedListFallsBackToAllScenesOnBusinessError(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/scene/r/list":
			_, _ = writer.Write([]byte(`{"success":false,"code":500,"message":"服务器内部错误"}`))
		case "/apis/iot/v1/scene/r/all":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"sceneId":"scene-1","houseId":"1001","roomId":"10","name":"灯光区回家"},{"sceneId":"scene-2","houseId":"1001","roomId":"11","name":"客厅回家"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunSceneScopedList(context.Background(), MetadataReadonlyRequest{
		HouseID:    "1001",
		Parameters: map[string]any{"roomId": "10"},
		Credentials: MetadataReadonlyCredentials{
			Authorization: "Bearer token-additional-secret",
			ClientID:      "client-additional-1",
		},
	})
	if err != nil {
		t.Fatalf("RunSceneScopedList error = %v", err)
	}
	if result.Partial || result.Capability != "scene.scoped.list" || result.APICalls != 2 {
		t.Fatalf("result = %#v", result)
	}
	if strings.Join(gotCalls, "\n") != "/apis/iot/v1/scene/r/list\n/apis/iot/v1/scene/r/all" {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "scene_scoped_list_all_fallback" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	data := result.Data.(map[string]any)
	scenes := data[semantic.FieldScenes].([]any)
	if len(scenes) != 1 || scenes[0].(map[string]any)[semantic.FieldName] != "灯光区回家" {
		t.Fatalf("scenes = %#v", scenes)
	}
}
