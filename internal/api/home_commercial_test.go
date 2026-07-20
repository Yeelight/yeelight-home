package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBizTypeSupportsConsumerAndCommercialAliases(t *testing.T) {
	for input, want := range map[string]string{"": "0", "普通家庭": "0", "consumer": "0", "1": "1", "商照家庭": "1", "commercial": "1"} {
		got, err := NormalizeBizType(input)
		if err != nil || got != want {
			t.Fatalf("NormalizeBizType(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := NormalizeBizType("invalid"); err == nil {
		t.Fatal("NormalizeBizType accepted invalid value")
	}
}

func TestCommercialHomeListUsesSaaSProjectDiscoveryAndRoleFilter(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		if request.Header.Get("bizType") != BizTypeCommercial {
			t.Fatalf("bizType = %q", request.Header.Get("bizType"))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case commercialSaaSRolePath:
			_, _ = writer.Write([]byte(`{"success":true,"data":"commercial_saas_admin"}`))
		case commercialProjectRolePath:
			_, _ = writer.Write([]byte(`{"success":true,"data":{"project-1":1,"project-2":2,"project-3":3}}`))
		case commercialProjectPagePath:
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body["pageNo"] != float64(1) || body["pageSize"] != float64(999) {
				t.Fatalf("project page body = %#v, err = %v", body, err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"houseId":"project-1","name":"旗舰展厅"},{"houseId":"project-2","name":"办公项目"},{"houseId":"project-3","name":"无权限项目"}]}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewHomeSummaryClient(Endpoint{Region: "cn", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunList(context.Background(), HomeSummaryCredentials{Authorization: "Bearer test", BizType: BizTypeCommercial})
	if err != nil {
		t.Fatalf("RunList error: %v", err)
	}
	if result.HouseCount != 2 || result.APICalls != 3 || result.Houses[0].ID != "project-1" || result.Houses[1].ID != "project-2" {
		t.Fatalf("result = %#v", result)
	}
	wantCalls := "GET " + commercialSaaSRolePath + "\nGET " + commercialProjectRolePath + "\nPOST " + commercialProjectPagePath
	if strings.Join(calls, "\n") != wantCalls {
		t.Fatalf("calls = %q", strings.Join(calls, "\n"))
	}
}

func TestCommercialSaaSUserDoesNotListProjects(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		_, _ = writer.Write([]byte(`{"success":true,"data":"commercial_saas_user"}`))
	}))
	defer server.Close()
	client := NewHomeSummaryClient(Endpoint{Region: "cn", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunList(context.Background(), HomeSummaryCredentials{Authorization: "Bearer test", BizType: BizTypeCommercial})
	if err != nil || result.HouseCount != 0 || result.APICalls != 1 || calls != 1 {
		t.Fatalf("result=%#v calls=%d err=%v", result, calls, err)
	}
}

func TestCommercialProjectRoleAllowedSupportsWrappedShapes(t *testing.T) {
	for _, value := range []any{
		float64(1), "2", map[string]any{"role": float64(1)}, map[string]any{"roleCode": "2"}, []any{map[string]any{"code": "3"}, map[string]any{"value": "1"}},
	} {
		if !commercialProjectRoleAllowed(value) {
			t.Fatalf("expected role to be allowed: %#v", value)
		}
	}
	for _, value := range []any{nil, float64(0), "3", map[string]any{"role": 3}, []any{map[string]any{"code": "4"}}} {
		if commercialProjectRoleAllowed(value) {
			t.Fatalf("expected role to be rejected: %#v", value)
		}
	}
}

func TestCallJSONUsesRequestScopedBizType(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got = request.Header.Get("bizType")
		_, _ = writer.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	ctx := WithBizType(context.Background(), BizTypeCommercial)
	if _, err := callJSON(ctx, server.Client(), http.MethodPost, server.URL, map[string]any{}, requestCredentials{}); err != nil {
		t.Fatalf("callJSON error: %v", err)
	}
	if got != BizTypeCommercial {
		t.Fatalf("bizType = %q", got)
	}
}
