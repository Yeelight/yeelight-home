package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunDeviceDetailGetProjectsSemanticFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/device-1/r/detail" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"alias":"主灯","attr":{"p":1,"l":42,"ct":3000,"mac":"AA:BB:CC:DD","localToken":"secret"},"capability":"p,l,ct","connectType":1,"deviceId":"device-1","did":"raw-did","gatewayDeviceId":"gw-1","houseId":"house-1","isBind":1,"isVirtual":1,"name":"主灯","roomId":"room-1","shadow":{"properties":{"p":true,"l":58,"ct":3200,"o":true,"localToken":"secret"}},"typeName":"色温灯"}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceDetailGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-detail-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceDetailGet error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"brightness":58`, `"colorTemperature":3200`, `"online":true`, `"attributes"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected detail missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-detail-secret", "secret", "AA:BB:CC:DD", `"attr"`, `"did"`, `"isBind"`, `"typeName"`, `"p"`, `"l"`, `"ct"`, "raw-did"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected detail leaked %q: %s", forbidden, text)
		}
	}
}

func TestRunDeviceComplexGetProjectsDynamicControlPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/complex" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"device-1","name":"客厅主灯","pid":133122,"pcId":44,"did":"raw-did","mac":"AA:BB:CC:DD","category":"light","typeName":"色温灯","roomId":"room-1","online":true,"localToken":"not-allowed","configs":[{"propId":"ct","desc":"色温","access":6,"format":"int","valueRange":{"min":2700,"max":6500,"step":100},"value":3200,"bindKey":"not-allowed"}],"properties":[{"propId":"p","desc":"开关","access":6,"format":"bool","value":true},{"propId":"l","desc":"亮度","access":6,"format":"int","valueRange":{"min":1,"max":100,"step":1},"value":72},{"propId":"online","desc":"在线","access":4,"format":"bool","value":true,"operators":[]},{"propId":"localToken","value":"not-allowed"}],"supportActions":[{"actionName":"toggle","params":[{"propId":"p","format":"bool"}],"secret":"not-allowed"}],"subDevices":[{"id":"sub-1","name":"副灯","category":"light","index":1,"did":"sub-did","properties":[{"propId":"ct","value":3300,"access":6}]}],"shadow":{"properties":{"p":true,"l":72,"ct":3200,"localToken":"not-allowed"}}}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceComplexGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-complex-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceComplexGet error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"property":"power"`, `"property":"brightness"`, `"property":"colorTemperature"`, `"property":"online"`, `"writable":false`, `"supportActions"`, `"actionName":"toggle"`, `"subDevices"`, `"typeName":"色温灯"`, `"current"`, `"propertyCount":3`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected complex detail missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-complex-secret", "not-allowed", "AA:BB:CC:DD", "raw-did", "sub-did", `"did"`, `"mac"`, `"localToken"`, `"bindKey"`, `"secret"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected complex detail leaked %q: %s", forbidden, text)
		}
	}
}

func TestRunDeviceComplexGetMarksProtocolSwitchLightPowerReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/complex" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"device-1","name":"light-knx开关灯-17000024-01","pid":17000024,"category":"light","properties":[{"propId":"p","desc":"开关","access":7,"format":"boolean","operators":["set","toggle"],"value":false},{"propId":"online","desc":"在线","access":4,"format":"boolean","operators":[],"value":true}],"configs":[{"propId":"adr_sw_ctl","desc":"开关控制组地址","access":6,"format":"uint32"},{"propId":"adr_sw_sts","desc":"开关状态组地址","access":6,"format":"uint32"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceComplexGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-complex-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceComplexGet error: %v", err)
	}
	data := result.Data.(map[string]any)
	detail := data["detail"].(map[string]any)
	properties := detail["properties"].([]any)
	for _, row := range properties {
		property := row.(map[string]any)
		if property["property"] == "power" {
			if property["writable"] != false {
				t.Fatalf("protocol switch light power should be read-only in user control projection: %#v", property)
			}
			return
		}
	}
	t.Fatalf("projected complex detail missing power property: %#v", detail)
}

func TestRunDeviceComplexGetMarksDaliLightControlsReadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/complex" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"device-1","name":"light-dali色温灯-17000004-01","pid":17000004,"category":"light","properties":[{"propId":"p","desc":"开关","access":7,"format":"boolean","operators":["set","toggle"],"value":false},{"propId":"l","desc":"亮度","access":7,"format":"int","operators":["set"],"value":68},{"propId":"ct","desc":"色温","access":7,"format":"int","operators":["set"],"value":3500},{"propId":"online","desc":"在线","access":4,"format":"boolean","operators":[],"value":true}],"configs":[{"propId":"adr_lit_ctl","desc":"灯光控制组地址","access":6,"format":"uint32"},{"propId":"adr_ct_ctl","desc":"色温控制组地址","access":6,"format":"uint32"}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceComplexGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-complex-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceComplexGet error: %v", err)
	}
	data := result.Data.(map[string]any)
	detail := data["detail"].(map[string]any)
	properties := detail["properties"].([]any)
	seen := map[string]bool{}
	for _, row := range properties {
		property := row.(map[string]any)
		name := stringFromAny(property["property"])
		if name == "power" || name == "brightness" || name == "colorTemperature" {
			seen[name] = true
			if property["writable"] != false {
				t.Fatalf("dali light control property should be read-only in user projection: %#v", property)
			}
		}
	}
	for _, want := range []string{"power", "brightness", "colorTemperature"} {
		if !seen[want] {
			t.Fatalf("projected complex detail missing %s property: %#v", want, detail)
		}
	}
}

func TestRunDeviceComplexGetInheritsProtocolContextForSubDeviceControls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/complex" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"device-1","name":"light-dali彩光灯-17000015-01","capabilityPid":"17000015","category":"light","properties":[{"propId":"p","desc":"开关","access":7,"format":"boolean","operators":["set","toggle"],"value":true},{"propId":"l","desc":"亮度","access":7,"format":"int","operators":["set"],"value":80},{"propId":"ct","desc":"色温","access":7,"format":"int","operators":["set"],"value":4500},{"propId":"c","desc":"彩光","access":7,"format":"uint32","operators":["set"]}],"configs":[{"propId":"daliVersion","desc":"DALI版本","access":7,"format":"uint8"},{"propId":"daliSwitchType","desc":"dali开关类型","access":7,"format":"uint8"}],"subDevices":[{"category":"light","componentId":"1","configs":[{"propId":"name","desc":"名称","access":6,"format":"string"}],"properties":[{"propId":"p","desc":"开关","access":7,"format":"boolean","operators":["set","toggle"]},{"propId":"l","desc":"亮度","access":7,"format":"int","operators":["set"]},{"propId":"ct","desc":"色温","access":7,"format":"int","operators":["set"]},{"propId":"c","desc":"彩光","access":7,"format":"uint32","operators":["set"]}]}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceComplexGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-complex-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceComplexGet error: %v", err)
	}
	data := result.Data.(map[string]any)
	detail := data["detail"].(map[string]any)
	subDevices := detail["subDevices"].([]any)
	if len(subDevices) != 1 {
		t.Fatalf("subDevices = %#v", detail)
	}
	properties := subDevices[0].(map[string]any)["properties"].([]any)
	seen := map[string]bool{}
	for _, row := range properties {
		property := row.(map[string]any)
		name := stringFromAny(property["property"])
		if name == "power" || name == "brightness" || name == "colorTemperature" || name == "color" {
			seen[name] = true
			if property["writable"] != false {
				t.Fatalf("protocol sub-device light control property should inherit read-only projection: %#v", property)
			}
		}
	}
	for _, want := range []string{"power", "brightness", "colorTemperature", "color"} {
		if !seen[want] {
			t.Fatalf("projected sub-device missing %s property: %#v", want, subDevices[0])
		}
	}
}

func TestRunGroupComplexGetProjectsDynamicControlPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v2/thing/manage/house/house-1/group/group-1/r/complex" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"id":"group-1","name":"客厅筒灯组","cid":55,"roomId":"room-1","rank":2,"configs":[{"propId":"l","desc":"亮度","access":6,"format":"int","value":66}],"properties":[{"propId":"p","desc":"开关","access":6,"format":"bool","value":true},{"propId":"ct","desc":"色温","access":6,"format":"int","value":3000}],"supportActions":[{"actionName":"toggle"}],"devices":[{"id":"device-1","name":"筒灯 1","did":"raw-did","category":"light","properties":[{"propId":"l","value":66,"access":6}],"localToken":"not-allowed"}],"psk":"not-allowed"}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunGroupComplexGet(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		Parameters:  map[string]any{"groupId": "group-1"},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-group-complex-secret"},
	})
	if err != nil {
		t.Fatalf("RunGroupComplexGet error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"property":"power"`, `"property":"colorTemperature"`, `"devices"`, `"deviceCount":1`, `"supportActions"`, `"productComponentId":55`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected group complex detail missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-group-complex-secret", "not-allowed", "raw-did", `"did"`, `"localToken"`, `"psk"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected group complex detail leaked %q: %s", forbidden, text)
		}
	}
}

func TestRunDeviceAttrListProjectsSemanticFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/device/r/attrs" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"attributes":[{"id":"device-1","p":1,"l":42,"ct":3000,"o":true,"mac":"AA:BB:CC:DD","ssid":"private-wifi","localToken":"secret","did":"raw-did","isBind":1}]}}`))
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunDeviceAttrList(context.Background(), MetadataReadonlyRequest{
		HouseID:     "house-1",
		DeviceID:    "device-1",
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-attr-secret"},
	})
	if err != nil {
		t.Fatalf("RunDeviceAttrList error: %v", err)
	}
	data, _ := json.Marshal(result.Data)
	text := string(data)
	for _, want := range []string{`"power":1`, `"brightness":42`, `"colorTemperature":3000`, `"online":true`} {
		if !strings.Contains(text, want) {
			t.Fatalf("projected attrs missing %s: %s", want, text)
		}
	}
	for _, forbidden := range []string{"token-attr-secret", "secret", "AA:BB:CC:DD", "private-wifi", `"p"`, `"l"`, `"ct"`, `"attr"`, `"did"`, `"isBind"`, "raw-did"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("projected attrs leaked %q: %s", forbidden, text)
		}
	}
}
