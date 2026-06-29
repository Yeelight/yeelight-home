package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeLightingDesignImportPayloadCreatesGatewayRoomsAndSlotsWithoutImplicitGroups(t *testing.T) {
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
	if first["pid"] != int64(-1) || first["connectType"] != -1 {
		t.Fatalf("function slot fields = %#v", first)
	}
	firstAttrs := first["attrs"].(map[string]any)
	if _, ok := firstAttrs["materialCode"]; ok {
		t.Fatalf("product attrs = %#v", firstAttrs)
	}
	firstCandidates := firstAttrs["productCandidates"].([]any)
	if len(firstCandidates) == 0 || firstCandidates[0].(map[string]any)["materialCode"] != "1-000000031" {
		t.Fatalf("ceiling light candidates = %#v", firstAttrs)
	}
	secondAttrs := devices[1].(map[string]any)["attrs"].(map[string]any)
	if _, ok := secondAttrs["materialCode"]; ok {
		t.Fatalf("fuzzy grid light should remain candidate-only without AI product selection: %#v", secondAttrs)
	}
	candidates := secondAttrs["productCandidates"].([]any)
	if len(candidates) == 0 || candidates[0].(map[string]any)["materialCode"] != "1-000002044" {
		t.Fatalf("grid light candidates = %#v", secondAttrs)
	}
	if groups, ok := payload["deviceGroups"]; ok {
		t.Fatalf("Runtime must not auto-create groups without explicit groups[] or deviceGroups[]: %#v", groups)
	}
}

func TestNormalizeLightingDesignImportPayloadConvertsExplicitNaturalGroups(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "客厅",
				"items": []any{
					map[string]any{"name": "吸顶灯", "quantity": float64(1), "category": "吸顶灯"},
					map[string]any{"name": "黑色格栅灯", "quantity": float64(2), "category": "格栅灯", "color": "黑色"},
					map[string]any{"name": "白色嵌入式射灯", "quantity": float64(2), "category": "筒射灯", "color": "白色", "installStyle": "嵌入式"},
				},
			},
		},
		"groups": []any{
			map[string]any{
				"name":     "客厅格栅灯组",
				"roomName": "客厅",
				"match":    map[string]any{"category": "格栅灯"},
			},
			map[string]any{
				"name":     "客厅嵌入式射灯组",
				"roomName": "客厅",
				"match":    map[string]any{"name": "白色嵌入式射灯"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	groups := payload["deviceGroups"].([]any)
	if len(groups) != 2 {
		t.Fatalf("deviceGroups=%#v", groups)
	}
	assertGroup := func(name string, wantCount int) {
		t.Helper()
		for _, raw := range groups {
			group := raw.(map[string]any)
			if group["localName"] != name {
				continue
			}
			deviceIDs := group["deviceIds"].([]any)
			if len(deviceIDs) != wantCount || group["roomId"] == nil {
				t.Fatalf("group %s = %#v", name, group)
			}
			return
		}
		t.Fatalf("group %s not found: %#v", name, groups)
	}
	assertGroup("客厅格栅灯组", 2)
	assertGroup("客厅嵌入式射灯组", 2)
	if _, ok := payload["groups"]; ok {
		t.Fatalf("natural groups alias must not be sent to cloud: %#v", payload)
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
	assertCandidateOnly("吸顶灯", "1-000000031")
	assertCandidateOnly("黑色格栅灯", "1-000002044")
	assertCandidateOnly("筒灯", "1-000003857")
	assertCandidateOnly("36°射灯", "1-000005105")
	assertCandidateOnly("爱思系列筒射灯", "1-000004861")
	assertCandidateOnly("夙夜版青空灯", "1-000003810")
}

func TestNormalizeLightingDesignImportPayloadPreservesCallerSelectedProducts(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "主卧",
				"items": []any{
					map[string]any{
						"name":         "36°射灯",
						"quantity":     float64(4),
						"materialCode": "1-000004714",
						"notes":        "Skill 按主卧重点照明选择 S 系列 75 开孔 36° 15w 深空灰候选",
					},
					map[string]any{
						"name":         "爱思系列筒射灯",
						"quantity":     float64(3),
						"materialCode": "1-000004861",
						"notes":        "Skill 按用户爱思系列描述选择 S 系列 75 开孔 36° 12w 深空灰候选",
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

func TestNormalizeLightingDesignImportPayloadIsIdempotent(t *testing.T) {
	normalized, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{
				"name": "客厅",
				"items": []any{
					map[string]any{"name": "黑色格栅灯", "quantity": float64(2), "category": "格栅灯"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	renormalized, err := NormalizeLightingDesignImportPayload("200191", normalized)
	if err != nil {
		t.Fatalf("Normalize normalized payload error: %v", err)
	}
	devices := renormalized["devices"].([]any)
	if len(devices) != 2 {
		t.Fatalf("renormalized devices=%#v", devices)
	}
	rooms := renormalized["rooms"].([]any)
	if len(rooms) != 1 || rooms[0].(map[string]any)["localName"] != "客厅" {
		t.Fatalf("renormalized rooms=%#v", rooms)
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
