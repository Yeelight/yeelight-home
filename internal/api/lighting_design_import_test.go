package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeLightingDesignImportPayloadAcceptsHouseMetaAndShortKeys(t *testing.T) {
	payload, err := NormalizeLightingDesignImportPayload("200191", compactHouseMetaFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if payload["houseId"] != float64(200191) || payload["name"] != "粒粒的美丽家庭" || payload["version"] != 2 {
		t.Fatalf("payload base fields = %#v", payload)
	}
	gateway := payload["gateway"].(map[string]any)
	rooms := gateway["roomList"].([]any)
	if len(rooms) != 1 {
		t.Fatalf("rooms=%#v", rooms)
	}
	room := rooms[0].(map[string]any)
	devices := room["deviceList"].([]any)
	if len(devices) != 2 {
		t.Fatalf("devices=%#v", devices)
	}
	firstDevice := devices[0].(map[string]any)
	pid, _ := lightingDesignIntFromAny(firstDevice["pid"])
	if firstDevice["roomTempId"] != "rm1" || pid != 198666 {
		t.Fatalf("first device=%#v", firstDevice)
	}
	extra := firstDevice["extraMeta"].(map[string]string)
	if extra["materialCode"] != "1-000002044" {
		t.Fatalf("extraMeta=%#v", extra)
	}
	groups := room["groupList"].([]any)
	componentID, _ := lightingDesignIntFromAny(groups[0].(map[string]any)["componentId"])
	if componentID != 4 {
		t.Fatalf("groups=%#v", groups)
	}
	scene := payload["sceneList"].([]any)[0].(map[string]any)
	detail := scene["details"].([]any)[0].(map[string]any)
	typeID, _ := lightingDesignIntFromAny(detail["typeId"])
	action, _ := lightingDesignIntFromAny(detail["action"])
	if typeID != 4 || detail["tempId"] != "gp1" || action != 0 {
		t.Fatalf("scene detail=%#v", detail)
	}
	if detail["params"] != `{"delay":0,"set":{"ct":3000,"l":60,"p":true}}` {
		t.Fatalf("scene detail params=%#v", detail["params"])
	}
	automation := payload["automationList"].([]any)[0].(map[string]any)
	if automation["version"] != 2 || automation["params"] != `{"conditions":[{"clock":"09:00:00","type":"alarm"}],"type":"and"}` {
		t.Fatalf("automation=%#v", automation)
	}
}

func TestNormalizeLightingDesignImportPayloadRejectsNaturalTopology(t *testing.T) {
	_, err := NormalizeLightingDesignImportPayload("200191", map[string]any{
		"rooms": []any{
			map[string]any{"name": "客厅", "items": []any{map[string]any{"name": "吸顶灯"}}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires HouseMeta payload") {
		t.Fatalf("expected HouseMeta guidance error, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadRejectsLegacyOverwriteFlags(t *testing.T) {
	payload := compactHouseMetaFixture()
	payload["clearAll"] = true
	_, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err == nil || !strings.Contains(err.Error(), "clearAll/overwrite is not supported") {
		t.Fatalf("expected clearAll rejection, got %v", err)
	}
}

func TestNormalizeLightingDesignImportPayloadIsIdempotent(t *testing.T) {
	normalized, err := NormalizeLightingDesignImportPayload("200191", compactHouseMetaFixture())
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	renormalized, err := NormalizeLightingDesignImportPayload("200191", normalized)
	if err != nil {
		t.Fatalf("Normalize normalized payload error: %v", err)
	}
	room := renormalized["gateway"].(map[string]any)["roomList"].([]any)[0].(map[string]any)
	group := room["groupList"].([]any)[0].(map[string]any)
	ids, ok := group["deviceTempIdList"].([]string)
	if !ok || len(ids) != 2 || ids[0] != "dv1" {
		t.Fatalf("renormalized group=%#v", group)
	}
}

func TestNormalizeLightingDesignImportPayloadValidatesReferences(t *testing.T) {
	payload := compactHouseMetaFixture()
	automation := payload["atl"].([]any)[0].(map[string]any)
	action := automation["as"].([]any)[0].(map[string]any)
	action["tid"] = "gp-missing"
	_, err := NormalizeLightingDesignImportPayload("200191", payload)
	if err == nil || !strings.Contains(err.Error(), "does not match an imported resource") {
		t.Fatalf("expected reference validation error, got %v", err)
	}
}

func TestLightingDesignImportClientUsesMetaImportAndVerifies(t *testing.T) {
	var importBody map[string]any
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/apis/iot/v1/meta/import":
			if err := json.NewDecoder(request.Body).Decode(&importBody); err != nil {
				t.Fatalf("decode import body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":"request-1"}`))
		case request.URL.Path == "/apis/iot/v1/meta/status":
			if request.URL.Query().Get("requestKey") != "request-1" {
				t.Fatalf("requestKey=%s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":{"status":"1","houseId":"200191"}}`))
		case strings.Contains(request.URL.Path, "/area/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case strings.Contains(request.URL.Path, "/room/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":4001,"name":"客厅"}]}}`))
		case strings.Contains(request.URL.Path, "/device/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":5001,"name":"黑色格栅灯1","roomId":4001},{"id":5002,"name":"黑色格栅灯2","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/group/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":6001,"name":"客厅格栅灯组","roomId":4001}]}}`))
		case strings.Contains(request.URL.Path, "/scene/r/info/"):
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"id":7001,"name":"客厅回家模式"}]}}`))
		case strings.Contains(request.URL.Path, "/automations/r/list"):
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":8001,"name":"客厅每天9点"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	result, err := NewLightingDesignImportClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client()).Run(context.Background(), LightingDesignImportRequest{
		HouseID:        "200191",
		Intent:         LightingDesignImportCapability,
		Payload:        compactHouseMetaFixture(),
		VerifyAttempts: 1,
		Credentials: LightingDesignImportCredentials{
			Authorization: "Bearer token-secret",
			ClientID:      "client-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v calls=%#v", err, calls)
	}
	if result.RequestKey != "request-1" || result.Mode != "house_meta_import" || result.Counts["devices"] != 2 || result.Counts["groups"] != 1 || result.Counts["automations"] != 1 {
		t.Fatalf("result=%#v", result)
	}
	if importBody["gateway"] == nil || importBody["sceneList"] == nil || importBody["automationList"] == nil {
		t.Fatalf("import body=%#v", importBody)
	}
	if result.APICalls != 8 {
		t.Fatalf("apiCalls=%d calls=%#v", result.APICalls, calls)
	}
}

func compactHouseMetaFixture() map[string]any {
	return map[string]any{
		"tid": "hm1",
		"n":   "粒粒的美丽家庭",
		"gateway": map[string]any{
			"tid": "gw1",
			"n":   "默认网关",
			"rl": []any{
				map[string]any{
					"tid": "rm1",
					"n":   "客厅",
					"dl": []any{
						map[string]any{"tid": "dv1", "n": "黑色格栅灯1", "pid": 198666, "mc": "1-000002044", "productName": "P20 明装磁吸格栅灯"},
						map[string]any{"tid": "dv2", "n": "黑色格栅灯2", "pid": 198666, "mc": "1-000002044", "productName": "P20 明装磁吸格栅灯"},
					},
					"gl": []any{
						map[string]any{"tid": "gp1", "n": "客厅格栅灯组", "cid": 4, "dtids": []any{"dv1", "dv2"}},
					},
				},
			},
		},
		"sl": []any{
			map[string]any{
				"tid": "sc1",
				"n":   "客厅回家模式",
				"ds": []any{
					map[string]any{"tpid": 4, "tid": "gp1", "rn": "客厅格栅灯组", "rk": 0, "ap": map[string]any{"dl": 0, "s": map[string]any{"p": true, "l": 60, "ct": 3000}}},
				},
			},
		},
		"atl": []any{
			map[string]any{
				"tid": "at1",
				"n":   "客厅每天9点",
				"st":  "00:00:00",
				"et":  "23:59:59",
				"rt":  2,
				"rv":  "0x7f",
				"ps":  map[string]any{"tp": "and", "cs": []any{map[string]any{"tp": "alarm", "c": "09:00:00"}}},
				"as": []any{
					map[string]any{"tpid": 4, "tid": "gp1", "rn": "客厅格栅灯组", "rk": 0, "ap": map[string]any{"dl": 0, "s": map[string]any{"p": true}}},
				},
			},
		},
	}
}
