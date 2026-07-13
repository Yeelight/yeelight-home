package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSceneExecuteClientRunsOpenControlScene(t *testing.T) {
	var gotAuthorization string
	var gotClientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotAuthorization = request.Header.Get("Authorization")
		gotClientID = request.Header.Get("Client-Id")
		writer.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost || request.URL.Path != "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID: "house-1",
		SceneID: "scene-1",
		Credentials: SceneExecuteCredentials{
			Authorization: "scene-secret-token",
			ClientID:      "client-scene-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotAuthorization != "Bearer scene-secret-token" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotClientID != "client-scene-1" {
		t.Fatalf("Client-Id = %q", gotClientID)
	}
	if result.Region != "dev" || result.HouseID != "house-1" || result.SceneID != "scene-1" || result.Source != "open_control_scene_endpoint" || result.APICalls != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSceneExecuteClientFallsBackWhenOpenControlReportsNoValidGateway(t *testing.T) {
	var gotCalls []string
	var gotFallbackHouseID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/controll/device/w/scene/scene-1":
			gotFallbackHouseID = request.Header.Get("houseId")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID: "house-1",
		SceneID: "scene-1",
		Credentials: SceneExecuteCredentials{
			Authorization: "scene-secret-token",
			ClientID:      "client-scene-1",
		},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if gotFallbackHouseID != "house-1" {
		t.Fatalf("fallback houseId header = %q", gotFallbackHouseID)
	}
	if result.Source != "control_device_scene_endpoint" || result.APICalls != 2 {
		t.Fatalf("result = %#v", result)
	}
	if len(gotCalls) != 2 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestSceneExecuteClientFallsBackToThingSceneEndpoint(t *testing.T) {
	var gotCalls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/controll/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":"40401","message":"not supported"}`))
		case "/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Source != "thing_device_scene_endpoint" || result.APICalls != 3 {
		t.Fatalf("result = %#v", result)
	}
	if len(gotCalls) != 3 {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
}

func TestSceneExecuteClientFallsBackToBatchSceneEndpointWithGatewayFromSceneDetail(t *testing.T) {
	var gotBatchBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","details":[{"typeId":2,"resId":"device-1","gatewayDeviceId":"gateway-1"}]}}`))
		case "/apis/iot/v1/thing/device/w/scenes":
			gotBatchBody = readRequestBody(t, request)
			_, _ = writer.Write([]byte(`{"success":true,"data":{"gateway-1":true}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Source != "thing_device_scenes_batch_endpoint" || result.APICalls != 5 {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(gotBatchBody, `"gateway-1":"scene-1"`) {
		t.Fatalf("batch body = %s", gotBatchBody)
	}
}

func TestSceneExecuteClientFallsBackToBatchSceneEndpointWithGatewayFromDeviceDetail(t *testing.T) {
	var gotDeviceDetail bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","details":[{"targetType":"device","targetId":"device-1"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info":
			gotDeviceDetail = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1","gatewayDeviceId":"gateway-1"}}`))
		case "/apis/iot/v1/thing/device/w/scenes":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"gateway-1":true}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !gotDeviceDetail {
		t.Fatal("device detail was not read")
	}
	if result.Source != "thing_device_scenes_batch_endpoint" || result.APICalls != 6 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSceneExecuteClientFallsBackToBatchSceneEndpointWithNestedSceneActions(t *testing.T) {
	var gotBatchBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"detail":{"actions":[{"targetType":"device","targetId":"device-1"}]}}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1","gatewayDeviceId":"gateway-1"}}`))
		case "/apis/iot/v1/thing/device/w/scenes":
			gotBatchBody = readRequestBody(t, request)
			_, _ = writer.Write([]byte(`{"success":true,"data":{"gateway-1":true}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Source != "thing_device_scenes_batch_endpoint" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(gotBatchBody, `"gateway-1":"scene-1"`) {
		t.Fatalf("batch body = %s", gotBatchBody)
	}
}

func TestSceneExecuteClientFallsBackToV1DeviceDetailForGateway(t *testing.T) {
	var gotV1Detail bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"details":[{"targetType":"device","targetId":"device-1"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1"}}`))
		case "/apis/iot/v1/device/device-1/r/detail":
			gotV1Detail = true
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1","gatewayDeviceId":"gateway-1"}}`))
		case "/apis/iot/v1/thing/device/w/scenes":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"gateway-1":true}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !gotV1Detail {
		t.Fatal("v1 device detail fallback was not read")
	}
	if result.Source != "thing_device_scenes_batch_endpoint" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSceneExecuteClientFallsBackToSceneActionsNodePropertySet(t *testing.T) {
	var gotPropertyPaths []string
	var gotPropertyBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"detail":{"actions":[{"targetType":"device","targetId":"device-1","set":{"power":true,"brightness":72}},{"targetType":"group","targetId":"group-1","set":{"ct":3000}}]}}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info",
			"/apis/iot/v1/device/device-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1"}}`))
		case "/apis/iot/v1/thing/device/w/scenes":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p",
			"/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/l",
			"/apis/iot/v1/open/control/house/house-1/control/4/group-1/w/properties/ct":
			gotPropertyPaths = append(gotPropertyPaths, request.URL.Path)
			gotPropertyBodies = append(gotPropertyBodies, readRequestBody(t, request))
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Source != "scene_actions_node_property_fallback" || result.APICalls != 7 {
		t.Fatalf("result = %#v", result)
	}
	if len(gotPropertyPaths) != 3 {
		t.Fatalf("gotPropertyPaths = %#v", gotPropertyPaths)
	}
	joinedBodies := strings.Join(gotPropertyBodies, "\n")
	if !strings.Contains(joinedBodies, `"value":true`) || !strings.Contains(joinedBodies, `"value":72`) || !strings.Contains(joinedBodies, `"value":3000`) {
		t.Fatalf("property bodies = %s", joinedBodies)
	}
}

func TestSceneExecuteClientFallsBackToSceneActionsFromParamsJSON(t *testing.T) {
	var gotPropertyPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"sceneId":"scene-1","details":[{"typeId":2,"resId":"device-1","params":"{\"set\":{\"power\":true,\"colorTemperature\":3000}}"}]}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info",
			"/apis/iot/v1/device/device-1/r/detail",
			"/apis/iot/v1/thing/device/w/scenes":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/p",
			"/apis/iot/v1/open/control/house/house-1/control/2/device-1/w/properties/ct":
			gotPropertyPaths = append(gotPropertyPaths, request.URL.Path)
			_, _ = writer.Write([]byte(`{"success":true,"data":{"result":"ok"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Source != "scene_actions_node_property_fallback" {
		t.Fatalf("result = %#v", result)
	}
	if len(gotPropertyPaths) != 2 {
		t.Fatalf("gotPropertyPaths = %#v", gotPropertyPaths)
	}
}

func TestSceneExecuteClientDoesNotFallbackWithoutSceneActionSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/open/control/house/house-1/control/w/scenes/scene-1",
			"/apis/iot/v1/controll/device/w/scene/scene-1",
			"/apis/iot/v1/thing/device/w/scene/scene-1":
			_, _ = writer.Write([]byte(`{"success":false,"code":1611,"message":"当前情景无有效网关"}`))
		case "/apis/iot/v1/scene/scene-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"detail":{"actions":[{"targetType":"device","targetId":"device-1"}]}}}`))
		case "/apis/iot/v2/thing/manage/house/house-1/device/device-1/r/info",
			"/apis/iot/v1/device/device-1/r/detail":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1"}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	_, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err == nil {
		t.Fatal("expected no valid gateway failure")
	}
	if !strings.Contains(err.Error(), "当前情景无有效网关") {
		t.Fatalf("err = %v", err)
	}
}

func TestSceneExecuteClientReportsBusinessFailureWithoutTokenLeak(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":false,"code":"40301","message":"no permission","data":null}`))
	}))
	defer server.Close()

	client := NewSceneExecuteClient(Endpoint{Region: "dev", BaseURL: server.URL}, server.Client())
	_, err := client.Run(context.Background(), SceneExecuteRequest{
		HouseID:     "house-1",
		SceneID:     "scene-1",
		Credentials: SceneExecuteCredentials{Authorization: "scene-secret-token"},
	})
	if err == nil {
		t.Fatal("expected business failure")
	}
	if !strings.Contains(err.Error(), "code=40301") || !strings.Contains(err.Error(), "message=no permission") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "scene-secret-token") {
		t.Fatalf("token leaked in error: %v", err)
	}
}

func readRequestBody(t *testing.T, request *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatalf("read body error: %v", err)
	}
	return string(data)
}
