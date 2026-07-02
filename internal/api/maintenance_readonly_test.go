package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaintenanceReadonlyAdaptersReturnRedactedProjection(t *testing.T) {
	var gotCalls []string
	var gotQueries []string
	var gotBodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotCalls = append(gotCalls, request.Method+" "+request.URL.Path)
		gotQueries = append(gotQueries, request.URL.RawQuery)
		body := bytes.Buffer{}
		_, _ = body.ReadFrom(request.Body)
		gotBodies = append(gotBodies, body.Bytes())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/upgrade/r/listfile":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[{"pid":1001,"version":42,"path":"/firmware.bin","md5":"abc","localToken":"not-allowed"}]}}`))
		case "/apis/iot/v1/upgrade/r/progress":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"deviceId":"device-1","state":1,"oldVersion":"1.0","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/upgrade/r/batchlistfile":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"deviceId":"device-1","version":43,"secret":"not-allowed"}]}`))
		case "/apis/iot/v1/progress/r/job-1":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"status":1,"progress":"80%","token":"not-allowed"}}`))
		case "/apis/iot/v1/appupgrade/r/latestfile":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"type":"1","osType":"1","version":8,"digitalVersion":"8.0.0","path":"/app.apk","accessToken":"not-allowed"}}`))
		case "/apis/iot/v1/ota/upgrade/r/batchListFilesByVersion":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"firmwareType":"main","version":44,"secret":"not-allowed"}]}`))
		case "/apis/iot/v1/nodeConfig/r/node_property":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"property":"power","range":"0/1","localToken":"not-allowed"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	credentials := MetadataReadonlyCredentials{Authorization: "Bearer token-maintenance-secret", ClientID: "client-1"}
	baseRequest := MetadataReadonlyRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Parameters: map[string]any{
			"capabilityPid":  1001,
			"currentVersion": "1.0",
			"deviceIds":      []any{"device-1"},
			"key":            "job-1",
			"languageCode":   "zh",
			"appType":        "yeelight",
			"osType":         "android",
			"firmwareType":   "main",
			"version":        44,
			"script":         "Hans",
			"region":         "CN",
			"nodeId":         "device-1",
			"nodeType":       "device",
		},
		Credentials: credentials,
	}

	results := []MetadataReadonlyResult{}
	for _, run := range []func() (MetadataReadonlyResult, error){
		func() (MetadataReadonlyResult, error) {
			return client.RunUpgradeFileList(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunUpgradeProgressGet(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunUpgradeFileBatchList(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunProgressGet(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunAppUpgradeLatestGet(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunOTAVersionFileBatchList(context.Background(), baseRequest)
		},
		func() (MetadataReadonlyResult, error) {
			return client.RunNodePropertyConfigGet(context.Background(), baseRequest)
		},
	} {
		result, err := run()
		if err != nil {
			t.Fatalf("run err = %v", err)
		}
		results = append(results, result)
	}

	expectedCalls := []string{
		"POST /apis/iot/v1/upgrade/r/listfile",
		"POST /apis/iot/v1/upgrade/r/progress",
		"POST /apis/iot/v1/upgrade/r/batchlistfile",
		"POST /apis/iot/v1/progress/r/job-1",
		"POST /apis/iot/v1/appupgrade/r/latestfile",
		"POST /apis/iot/v1/ota/upgrade/r/batchListFilesByVersion",
		"POST /apis/iot/v1/nodeConfig/r/node_property",
	}
	if strings.Join(gotCalls, "\n") != strings.Join(expectedCalls, "\n") {
		t.Fatalf("gotCalls = %#v", gotCalls)
	}
	if !bytes.Contains(gotBodies[0], []byte(`"pid":1001`)) || bytes.Contains(gotBodies[0], []byte("capabilityPid")) {
		t.Fatalf("unexpected listfile body: %s", string(gotBodies[0]))
	}
	if !bytes.Contains(gotBodies[4], []byte(`"type":"1"`)) || !bytes.Contains(gotBodies[4], []byte(`"osType":"1"`)) {
		t.Fatalf("unexpected app upgrade body: %s", string(gotBodies[4]))
	}
	if !bytes.Contains(gotBodies[5], []byte(`"queryList"`)) || !strings.Contains(gotCalls[5], "/batchListFilesByVersion") {
		t.Fatalf("unexpected ota version body/call: call=%s body=%s", gotCalls[5], string(gotBodies[5]))
	}
	if !strings.Contains(gotQueries[5], "language=zh") || !strings.Contains(gotQueries[5], "region=CN") || !strings.Contains(gotQueries[5], "script=Hans") {
		t.Fatalf("unexpected ota version query: %s", gotQueries[5])
	}
	if body := strings.TrimSpace(string(gotBodies[6])); body != "" && body != "null" {
		t.Fatalf("node property config should use query params only, body: %s", string(gotBodies[6]))
	}
	if !strings.Contains(gotQueries[6], "nodeId=device-1") || !strings.Contains(gotQueries[6], "nodeType=device") {
		t.Fatalf("unexpected node property config query: %s", gotQueries[6])
	}
	for _, result := range results {
		if result.Partial || result.APICalls != 1 {
			t.Fatalf("result = %#v", result)
		}
		data, err := json.Marshal(result.Data)
		if err != nil {
			t.Fatalf("marshal data: %v", err)
		}
		for _, forbidden := range []string{"not-allowed", "token-maintenance-secret"} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("result leaked %q: %s", forbidden, string(data))
			}
		}
	}
}

func TestMaintenanceReadonlyMissingContextDoesNotCallCloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatalf("unexpected HTTP call: %s %s", request.Method, request.URL.Path)
	}))
	defer server.Close()
	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	credentials := MetadataReadonlyCredentials{Authorization: "Bearer token-maintenance-secret", ClientID: "client-1"}

	tests := []struct {
		name    string
		run     func() (MetadataReadonlyResult, error)
		warning string
	}{
		{
			name: "upgrade file query",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunUpgradeFileList(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "upgrade_file_query_context_missing",
		},
		{
			name: "upgrade progress device",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunUpgradeProgressGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "device_context_missing",
		},
		{
			name: "batch file query",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunUpgradeFileBatchList(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "upgrade_batch_query_context_missing",
		},
		{
			name: "progress key",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunProgressGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "progress_key_missing",
		},
		{
			name: "app upgrade query",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunAppUpgradeLatestGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "app_upgrade_query_context_missing",
		},
		{
			name: "ota version query",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunOTAVersionFileBatchList(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "ota_version_file_query_context_missing",
		},
		{
			name: "node property config query",
			run: func() (MetadataReadonlyResult, error) {
				return client.RunNodePropertyConfigGet(context.Background(), MetadataReadonlyRequest{Parameters: map[string]any{}, Credentials: credentials})
			},
			warning: "node_property_config_context_missing",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.run()
			if err != nil {
				t.Fatalf("run err = %v", err)
			}
			if !result.Partial || result.APICalls != 0 || len(result.Warnings) != 1 || result.Warnings[0] != test.warning {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestNodePropertyConfigHTTPBadRequestReturnsPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apis/iot/v1/nodeConfig/r/node_property" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		http.Error(writer, `{"code":400,"message":"参数格式错误"}`, http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewMetadataReadonlyClient(Endpoint{Region: "dev", BaseURL: server.URL + "/apis/iot"}, server.Client())
	result, err := client.RunNodePropertyConfigGet(context.Background(), MetadataReadonlyRequest{
		HouseID:  "house-1",
		DeviceID: "device-1",
		Parameters: map[string]any{
			"nodeId":   "device-1",
			"nodeType": "device",
		},
		Credentials: MetadataReadonlyCredentials{Authorization: "Bearer token-maintenance-secret"},
	})
	if err != nil {
		t.Fatalf("RunNodePropertyConfigGet error = %v", err)
	}
	if !result.Partial || result.APICalls != 1 || result.RawShape != "<http_400>" {
		t.Fatalf("result = %#v", result)
	}
	if result.HouseID != "house-1" || result.DeviceID != "device-1" || result.Capability != "node.property_config.get" {
		t.Fatalf("context = %#v", result)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "cloud_read_endpoint_unavailable" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}
