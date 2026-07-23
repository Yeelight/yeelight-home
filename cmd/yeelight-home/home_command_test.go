package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func TestHomeSelectLetsInteractiveUserChooseByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"house-1","name":"我的家"},{"id":"house-2","name":"父母家"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	app.terminal = func(io.Reader) bool { return true }
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer test-token"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", Language: "zh-CN"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"home", "select", "--region", "dev"}, strings.NewReader("父母家\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("home select code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "house-2" {
		t.Fatalf("metadata = %#v, ok = %v, err = %v", metadata, ok, err)
	}
	if !strings.Contains(stdout.String(), "父母家") {
		t.Fatalf("prompt did not list homes: %s", stdout.String())
	}
}

func TestHomeSelectUsesAndPersistsExplicitLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"house-1","name":"Home"},{"id":"house-2","name":"Parents"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	app.terminal = func(io.Reader) bool { return true }
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer test-token"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", Language: "zh-CN"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"home", "select", "--region", "dev", "--lang", "en-US"}, strings.NewReader("2\n"), &stdout, &stderr)
	metadata, ok, err := app.metadataStore.Load("default")
	if code != exitOK || err != nil || !ok || metadata.HouseID != "house-2" || metadata.Language != "en-US" {
		t.Fatalf("code=%d metadata=%#v ok=%v err=%v stdout=%s stderr=%s", code, metadata, ok, err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Choose the default home") {
		t.Fatalf("prompt was not localized: %s", stdout.String())
	}
}

func TestHomeSelectWithoutIDExplainsHowToRecoverInNonInteractiveUse(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"home", "select"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput || !strings.Contains(stderr.String(), "interactive terminal") || !strings.Contains(stderr.String(), "--house-id") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}
