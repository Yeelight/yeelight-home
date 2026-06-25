package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestDevSeedHouseRequiresAllowWriteDev(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"dev", "seed-house", "--json", "--region", "dev", "--name", "Codex Dev Test Home"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--allow-write-dev") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestDevSeedHouseRejectsNonDevRegion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"dev", "seed-house", "--json", "--region", "cn", "--name", "Codex Dev Test Home", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "only allowed for dev") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestDevSeedHouseCreatesHouseAndSavesMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	listCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			listCalls++
			if listCalls == 1 {
				_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
				return
			}
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-created","name":"Codex Dev Test Home"}]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"data":"house-created"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")

	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-dev-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-dev-1", QRDevice: "F8:24:41:1B:55:37"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"dev", "seed-house", "--json", "--region", "dev", "--name", "Codex Dev Test Home", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-dev-secret") || strings.Contains(stderr.String(), "token-dev-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if len(calls) != 4 {
		t.Fatalf("calls = %#v", calls)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.HouseID != "house-created" || metadata.ClientID != "client-dev-1" || metadata.QRDevice != "F8:24:41:1B:55:37" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["houseId"] != "house-created" || response["created"] != true || response["verified"] != true {
		t.Fatalf("response = %#v", response)
	}
}

func TestDevSeedHouseSavesMetadataWhenCreateIDIsEntityListVerified(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		case "/apis/iot/v2/thing/manage/house/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/w/create":
			_, _ = writer.Write([]byte(`{"success":true,"data":"house-created"}`))
		case "/apis/iot/v2/thing/manage/house/house-created/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-created/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-created/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")

	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-dev-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-dev-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"dev", "seed-house", "--json", "--region", "dev", "--name", "Codex Dev Test Home", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.HouseID != "house-created" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["verifiedBy"] != "entity_list" {
		t.Fatalf("response = %#v", response)
	}
}

func TestDevSeedHouseReusesStoredHouseIDWhenEntityListVerifies(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls = append(calls, request.Method+" "+request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/iot/v2/thing/manage/house/house-stored/room/r/info/1/100",
			"/apis/iot/v2/thing/manage/house/house-stored/area/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-stored/device/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-stored/group/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v2/thing/manage/house/house-stored/scene/r/info/1/100":
			_, _ = writer.Write([]byte(`{"success":true,"data":{"rows":[]}}`))
		case "/apis/iot/v1/automations/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")

	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-dev-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-dev-1", HouseID: "house-stored"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"dev", "seed-house", "--json", "--region", "dev", "--name", "Codex Dev Test Home", "--allow-write-dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, call := range calls {
		if strings.Contains(call, "/w/create") || strings.Contains(call, "/house/r/list") {
			t.Fatalf("unexpected create/list call: %#v", calls)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["houseId"] != "house-stored" || response["created"] != false || response["verifiedBy"] != "entity_list_candidate" {
		t.Fatalf("response = %#v", response)
	}
}
