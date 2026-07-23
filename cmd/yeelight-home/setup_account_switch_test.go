package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/credential"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func TestSetupClearHomeFailureRollsBackPreviousAccount(t *testing.T) {
	app := newTestApp(t)
	oldMetadata := credential.ProfileMetadata{
		Profile: "default", Region: "dev", HouseID: "old-house", ClientID: "old-client", QRDevice: "F8:24:41:00:00:01",
	}
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(oldMetadata); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	app.metadataStore = &failNthMetadataStore{
		FileMetadataStore: credential.NewFileMetadataStore(app.metadataStore.Path()),
		FailAt:            2,
		Err:               errors.New("clear home storage unavailable"),
	}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-clear-failure", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-clear-failure", Status: "LOGIN",
			Token: auth.QRToken{AccessToken: "new-secret", ClientID: "new-client"},
		}},
	}
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: io.Discard}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Region: "dev", Locale: "zh-CN", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err == nil || !strings.Contains(err.Error(), "clear home storage unavailable") {
		t.Fatalf("error=%v", err)
	}
	assertSetupAccountState(t, app, "Bearer old-secret", oldMetadata)
}

func TestSetupLocaleSaveFailureRollsBackPreviousAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"new-house","name":"New home"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")

	app := newTestApp(t)
	oldMetadata := credential.ProfileMetadata{
		Profile: "default", Region: "dev", BizType: "0", HouseID: "old-house", ClientID: "old-client", Language: "en-US", QRDevice: "F8:24:41:00:00:02",
	}
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(oldMetadata); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	app.metadataStore = &failNthMetadataStore{
		FileMetadataStore: credential.NewFileMetadataStore(app.metadataStore.Path()),
		FailAt:            4,
		Err:               errors.New("language storage unavailable"),
	}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-locale-failure", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-locale-failure", Status: "LOGIN",
			Token: auth.QRToken{AccessToken: "new-secret", ClientID: "new-client"},
		}},
	}
	app.process = func(context.Context, []string, io.Writer, io.Writer) error { return nil }
	plan, err := setupdomain.BuildPlan(setupdomain.Options{
		Locale: "zh-CN", ClientID: "codex", Mode: setupdomain.ModeSkill, BizType: "0", HomeDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: io.Discard}
	_, err = app.executeSetupPlan(plan, setupExecutionOptions{
		Profile: "default", Region: "dev", BizType: "0", Locale: "zh-CN", HomeDir: t.TempDir(),
		Interactive: true, Prompt: prompt, Stdout: io.Discard, Stderr: io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "save setup language") {
		t.Fatalf("error=%v", err)
	}
	assertSetupAccountState(t, app, "Bearer old-secret", oldMetadata)
}

type failNthMetadataStore struct {
	credential.FileMetadataStore
	FailAt int
	Err    error
	Saves  int
}

func (store *failNthMetadataStore) Save(metadata credential.ProfileMetadata) error {
	store.Saves++
	if store.Saves == store.FailAt {
		return store.Err
	}
	return store.FileMetadataStore.Save(metadata)
}

func assertSetupAccountState(t *testing.T, app *app, wantToken string, wantMetadata credential.ProfileMetadata) {
	t.Helper()
	record, ok, err := app.tokenStore.Load(wantMetadata.Profile)
	if err != nil || !ok || record.AccessToken != wantToken {
		t.Fatalf("token=%#v ok=%t err=%v", record, ok, err)
	}
	metadata, ok, err := app.metadataStore.Load(wantMetadata.Profile)
	if err != nil || !ok || metadata != wantMetadata {
		t.Fatalf("metadata=%#v want=%#v ok=%t err=%v", metadata, wantMetadata, ok, err)
	}
}
