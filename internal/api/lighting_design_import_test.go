package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeLightingDesignImportPayloadCreatesGatewayRoomsSlotsAndGroups(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "客厅",
				"items": []any{
					map[string]any{"name": "吸顶灯", "quantity": float64(1)},
					map[string]any{"name": "黑色格栅灯", "quantity": float64(2), "color": "黑色", "category": "格栅灯"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if payload["houseId"] != float64(200191) || payload["clearAll"] != false {
		t.Fatalf("payload base fields = %#v", payload)
	}
	if got := len(payload["gateways"].([]any)); got != 1 {
		t.Fatalf("gateways = %d", got)
	}
	if got := len(payload["rooms"].([]any)); got != 1 {
		t.Fatalf("rooms = %d", got)
	}
	devices := payload["devices"].([]any)
	if len(devices) != 3 {
		t.Fatalf("devices = %#v", devices)
	}
	first := devices[0].(map[string]any)
	if first["pid"] != int64(198666) || first["connectType"] != -1 {
		t.Fatalf("function slot fields = %#v", first)
	}
	firstAttrs := first["attrs"].(map[string]any)
	if firstAttrs["materialCode"] != "1-000000031" || firstAttrs["productName"] != "Yeelight Pro M20 吸顶灯 C450" {
		t.Fatalf("product attrs = %#v", firstAttrs)
	}
	secondAttrs := devices[1].(map[string]any)["attrs"].(map[string]any)
	if _, ok := secondAttrs["materialCode"]; ok {
		t.Fatalf("fuzzy grid light should remain candidate-only without AI product selection: %#v", secondAttrs)
	}
	candidates := secondAttrs["productCandidates"].([]any)
	if len(candidates) == 0 || candidates[0].(map[string]any)["materialCode"] != "1-000002044" {
		t.Fatalf("grid light candidates = %#v", secondAttrs)
	}
	groups := payload["deviceGroups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("groups = %#v", groups)
	}
	group := groups[0].(map[string]any)
	if group["localName"] != "客厅格栅灯组" {
		t.Fatalf("group = %#v", group)
	}
}

func TestNormalizeLightingDesignImportPayloadEnrichesNaturalDesignSlotProducts(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "客厅",
				"items": []any{
					map[string]any{"name": "吸顶灯"},
					map[string]any{"name": "黑色格栅灯", "quantity": float64(2)},
					map[string]any{"name": "白色嵌入式射灯", "quantity": float64(2), "color": "白色", "installStyle": "嵌入式", "category": "射灯"},
				},
			},
			map[string]any{
				"name": "次卧",
				"items": []any{
					map[string]any{"name": "筒灯", "quantity": float64(4)},
				},
			},
			map[string]any{
				"name": "主卧",
				"items": []any{
					map[string]any{"name": "方形吸顶灯"},
					map[string]any{"name": "36°射灯", "quantity": float64(4), "beamAngle": "36°", "category": "射灯"},
					map[string]any{"name": "爱思系列筒射灯", "quantity": float64(3), "series": "爱思系列", "category": "筒射灯"},
				},
			},
			map[string]any{
				"name": "卫生间",
				"items": []any{
					map[string]any{"name": "夙夜版青空灯"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	devices := payload["devices"].([]any)
	if len(devices) != 18 {
		t.Fatalf("devices=%d %#v", len(devices), devices)
	}
	assertResolvedProduct := func(name string, materialCode string) {
		t.Helper()
		for _, raw := range devices {
			device := raw.(map[string]any)
			if strings.Contains(device["localName"].(string), name) {
				attrs := device["attrs"].(map[string]any)
				if attrs["materialCode"] != materialCode {
					t.Fatalf("%s materialCode=%#v attrs=%#v", name, attrs["materialCode"], attrs)
				}
				if device["pid"] == int64(-1) {
					t.Fatalf("%s should have resolved pid: %#v", name, device)
				}
				return
			}
		}
		t.Fatalf("device %s not found in %#v", name, devices)
	}
	assertCandidateOnly := func(name string, materialCode string) {
		t.Helper()
		for _, raw := range devices {
			device := raw.(map[string]any)
			if strings.Contains(device["localName"].(string), name) {
				attrs := device["attrs"].(map[string]any)
				if _, ok := attrs["materialCode"]; ok {
					t.Fatalf("%s should remain candidate-only without AI-selected materialCode: %#v", name, attrs)
				}
				candidates := attrs["productCandidates"].([]any)
				if len(candidates) == 0 {
					t.Fatalf("%s should have product candidates: %#v", name, attrs)
				}
				first := candidates[0].(map[string]any)
				if first["materialCode"] != materialCode {
					t.Fatalf("%s first candidate=%#v attrs=%#v", name, first["materialCode"], attrs)
				}
				return
			}
		}
		t.Fatalf("device %s not found in %#v", name, devices)
	}
	assertResolvedProduct("吸顶灯", "1-000000031")
	assertCandidateOnly("黑色格栅灯", "1-000002044")
	assertCandidateOnly("筒灯", "1-000003857")
	assertCandidateOnly("36°射灯", "1-000005105")
	assertCandidateOnly("爱思系列筒射灯", "1-000004861")
	assertResolvedProduct("夙夜版青空灯", "1-000003810")
}

func TestNormalizeLightingDesignImportPayloadPreservesAISelectedProducts(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "主卧",
				"items": []any{
					map[string]any{
						"name":         "36°射灯",
						"quantity":     float64(4),
						"materialCode": "1-000004714",
						"notes":        "AI 按主卧重点照明选择 S 系列 75 开孔 36° 15w 深空灰候选",
					},
					map[string]any{
						"name":         "爱思系列筒射灯",
						"quantity":     float64(3),
						"materialCode": "1-000004861",
						"notes":        "AI 按用户爱思系列描述选择 S 系列 75 开孔 36° 12w 深空灰候选",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	devices := payload["devices"].([]any)
	if len(devices) != 7 {
		t.Fatalf("devices=%d %#v", len(devices), devices)
	}
	for _, item := range []struct {
		name         string
		materialCode string
	}{
		{name: "36°射灯", materialCode: "1-000004714"},
		{name: "爱思系列筒射灯", materialCode: "1-000004861"},
	} {
		for _, raw := range devices {
			device := raw.(map[string]any)
			if !strings.Contains(device["localName"].(string), item.name) {
				continue
			}
			attrs := device["attrs"].(map[string]any)
			if attrs["productMatchConfidence"] != "explicit" || attrs["materialCode"] != item.materialCode {
				t.Fatalf("%s attrs=%#v", item.name, attrs)
			}
			if device["pid"] == int64(-1) || device["pcId"] != int64(4) {
				t.Fatalf("%s product identity not applied to device: %#v", item.name, device)
			}
		}
	}
}

func TestLightingDesignImportClientSyncsMetadataAndVerifies(t *testing.T) {
	var syncBody map[string]any
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/design/syncMetadata":
			if err := json.NewDecoder(request.Body).Decode(&syncBody); err != nil {
				t.Fatalf("decode sync body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceLocalIdToCloudSlotIds":{"1003":5001},"roomLocalIdToCloudSlotIds":{"1002":4001}}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"吸顶灯","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewLightingDesignImportClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), LightingDesignImportRequest{
		HouseID: "200191",
		Intent:  DeviceSlotCreateCapability,
		Payload: map[string]any{
			"rooms": []any{
				map[string]any{"name": "客厅", "items": []any{map[string]any{"name": "吸顶灯"}}},
			},
		},
		VerifyAttempts: 1,
		Credentials: LightingDesignImportCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v calls=%#v", err, calls)
	}
	if !result.Verified || result.VerifiedBy != "entity.list" || result.Counts["devices"] != 1 {
		t.Fatalf("result = %#v", result)
	}
	if syncBody["clearAll"] != false || len(syncBody["devices"].([]any)) != 1 {
		t.Fatalf("sync body = %#v", syncBody)
	}
	if result.APICalls != 7 {
		t.Fatalf("apiCalls=%d calls=%#v", result.APICalls, calls)
	}
}
