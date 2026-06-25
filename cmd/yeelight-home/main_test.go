package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/plan"
	"github.com/yeelight/yeelight-home/internal/storage"
)

func TestRootHelpAndVersionFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput string
	}{
		{name: "empty args show root help", args: []string{}, wantOutput: "Usage:\n  yeelight-home <command> [flags]"},
		{name: "long help", args: []string{"--help"}, wantOutput: "Commands:\n  auth"},
		{name: "short help", args: []string{"-h"}, wantOutput: "Global flags:"},
		{name: "help command", args: []string{"help", "home"}, wantOutput: "yeelight-home home list"},
		{name: "subcommand help", args: []string{"home", "--help"}, wantOutput: "home list is account-scoped"},
		{name: "nested help command", args: []string{"help", "auth", "token", "set"}, wantOutput: "Omit houseId for token-only account-scoped use"},
		{name: "nested trailing help", args: []string{"auth", "token", "set", "--help"}, wantOutput: "yeelight-home auth token set --token"},
		{name: "nested dev help", args: []string{"dev", "seed-room", "--help"}, wantOutput: "yeelight-home dev seed-room --json --region dev"},
		{name: "long version", args: []string{"--version"}, wantOutput: "yeelight-home dev"},
		{name: "short version", args: []string{"-v"}, wantOutput: "yeelight-home dev"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := run(test.args, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("stdout = %s, want substring %q", stdout.String(), test.wantOutput)
			}
		})
	}
}

func TestUnknownHelpTopicReturnsInvalidInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"help", "missing-command"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown help topic "missing-command"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:\n  yeelight-home <command> [flags]") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestInvokeRejectsUnknownIntent(t *testing.T) {
	input := `{"contractVersion":"1.0","requestId":"req-1","locale":"zh-CN","utterance":"测试","intent":"raw.api.call"}`
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke", "--stdin"}, strings.NewReader(input), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unsupported intent") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestInvokeRequiresStdinFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"invoke"}, strings.NewReader("{}"), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: yeelight-home invoke --stdin") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestAuthStatusJSONDoesNotExposeToken(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_AUTHENTICATED", "1")
	t.Setenv("YEELIGHT_HOME_PROFILE", "family-main")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "secret-token-value")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"auth", "status", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "secret-token-value") {
		t.Fatalf("stdout leaked token: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["authenticated"] != true {
		t.Fatalf("authenticated = %v", response["authenticated"])
	}
	if response["profile"] != "family-main" {
		t.Fatalf("profile = %v", response["profile"])
	}
}

func TestDoctorJSONReportsConfigAndAuthWarning(t *testing.T) {
	t.Setenv("YEELIGHT_HOME_DIR", "/tmp/yeelight-home-test")
	t.Setenv("YEELIGHT_HOME_PROFILE", "doctor-empty-profile")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := newTestApp(t).run([]string{"doctor", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["status"] != "warning" {
		t.Fatalf("status = %v", response["status"])
	}
	if response["configDir"] != "/tmp/yeelight-home-test/config" {
		t.Fatalf("configDir = %v", response["configDir"])
	}
	migrations, ok := response["memoryMigrations"].(map[string]any)
	if !ok {
		t.Fatalf("memoryMigrations = %#v", response["memoryMigrations"])
	}
	if migrations["status"] != "available" {
		t.Fatalf("memory migration status = %v", migrations["status"])
	}
}

func TestAuthLoginQRNoWaitPrintsPayloadWithoutToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev", "--device", "f82441000001"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["payload"] != "cli&F8:24:41:00:00:01&qr-nowait-1" {
		t.Fatalf("payload = %v", response["payload"])
	}
	if strings.Contains(stdout.String(), "token") {
		t.Fatalf("stdout leaked token-like data: %s", stdout.String())
	}
}

func TestAuthLoginQRUsesStableProfileDevice(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	client := &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}
	app.qrClient = client

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("first login exit code = %d, stderr = %s", code, stderr.String())
	}
	firstDevice := client.createDevices[0]
	if firstDevice == "" || firstDevice == "F8:24:41:00:00:01" {
		t.Fatalf("first device = %q", firstDevice)
	}

	stdout.Reset()
	stderr.Reset()
	client.created = auth.QRInfo{QRCodeID: "qr-nowait-2", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()}
	code = app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("second login exit code = %d, stderr = %s", code, stderr.String())
	}
	if client.createDevices[1] != firstDevice {
		t.Fatalf("second device = %q, want stable %q", client.createDevices[1], firstDevice)
	}
}

func TestAuthLoginQRExplicitDeviceOverridesStableProfileDevice(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	client := &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}
	app.qrClient = client

	code := app.run([]string{"auth", "login", "--qr", "--json", "--no-wait", "--region", "dev", "--device", "f82441010203"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("login exit code = %d, stderr = %s", code, stderr.String())
	}
	if client.createDevices[0] != "F8:24:41:01:02:03" {
		t.Fatalf("create device = %q", client.createDevices[0])
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.QRDevice != "F8:24:41:01:02:03" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
}

func TestAuthLoginQRNoWaitPlainTextPrintsTerminalQRCode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-nowait-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
	}

	code := app.run([]string{"auth", "login", "--qr", "--no-wait", "--region", "dev", "--device", "f82441000001"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "██") {
		t.Fatalf("expected terminal QR output, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Payload: cli&F8:24:41:00:00:01&qr-nowait-1") {
		t.Fatalf("expected payload fallback, got %s", stdout.String())
	}
}

func TestAuthLoginQRPlainTextPrintsQRCodeBeforePollingCompletes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-login-1",
			Status:   "LOGIN",
			Token:    auth.QRToken{AccessToken: "token-qr-secret-123456"},
		}},
	}
	app.sleep = func(context.Context, time.Duration) error {
		if !strings.Contains(stdout.String(), "Payload: cli&") || !strings.Contains(stdout.String(), "&qr-login-1") {
			t.Fatalf("expected QR prompt before polling, got %s", stdout.String())
		}
		return nil
	}

	code := app.run([]string{"auth", "login", "--qr", "--region", "dev", "--poll-interval-ms", "1", "--timeout-ms", "1000"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestAuthLoginQRSavesTokenAndMetadataWithoutLeakingToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-login-1", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-login-1",
			Status:   "LOGIN",
			Token:    auth.QRToken{AccessToken: "token-qr-secret-123456", ClientID: "client-qr-123456"},
			Source:   `dali:{"houseId":"house-qr-123456"}`,
		}},
	}

	code := app.run([]string{"auth", "login", "--qr", "--json", "--region", "dev", "--poll-interval-ms", "1", "--timeout-ms", "1000"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-qr-secret-123456") || strings.Contains(stderr.String(), "token-qr-secret-123456") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	record, ok, err := app.tokenStore.Load("default")
	if err != nil {
		t.Fatalf("Load token error: %v", err)
	}
	if !ok || record.AccessToken != "Bearer token-qr-secret-123456" {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "dev" || metadata.ClientID != "client-qr-123456" || metadata.HouseID != "house-qr-123456" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	credentials, ok := response["credentials"].(map[string]any)
	if !ok {
		t.Fatalf("credentials = %#v", response["credentials"])
	}
	if credentials["accessTokenPresent"] != true || credentials["houseId"] != "house-qr-123456" {
		t.Fatalf("credentials = %#v", credentials)
	}
	if _, ok := credentials["clientId"]; ok {
		t.Fatalf("credentials exposed clientId: %#v", credentials)
	}
}

func TestAPISmokeUsesEnvCredentialsAndDoesNotExposeToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var requestBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Body != nil {
			body, _ := io.ReadAll(request.Body)
			requestBodies = append(requestBodies, string(body))
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/apis/account/user/info":
			_, _ = writer.Write([]byte(`{"code":"200","data":{"nickname":"测试用户"}}`))
		case "/apis/iot/v1/house/r/list":
			_, _ = writer.Write([]byte(`{"success":true,"data":[{"id":"house-1","name":"默认家庭"}]}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-smoke-secret-123456")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-smoke-123456"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"api", "smoke", "--json", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-smoke-secret-123456") || strings.Contains(stderr.String(), "token-smoke-secret-123456") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Contains(strings.Join(requestBodies, "\n"), "houseId") {
		t.Fatalf("api smoke should not require or send houseId when unset: %#v", requestBodies)
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["accountOk"] != true || response["houseListOk"] != true || response["houseCount"] != float64(1) {
		t.Fatalf("response = %#v", response)
	}
}

func TestHomeListJSONAccountScopedEmptyListIncludesDiagnostics(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"home", "list", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-home-list-secret") || strings.Contains(stderr.String(), "token-home-list-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["houseCount"] != float64(0) || response["rawShape"] != "map[string]interface {}" || response["apiCalls"] != float64(1) {
		t.Fatalf("response = %#v", response)
	}
	warnings, ok := response["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "empty_account_home_list" {
		t.Fatalf("warnings = %#v", response["warnings"])
	}
}

func TestHomeListUnauthorizedReturnsActionableAuthError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "token-home-list-secret")
	app := newTestApp(t)

	code := app.run([]string{"home", "list", "--json", "--region", "dev"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "authorization failed") || !strings.Contains(stderr.String(), "auth login --qr --region dev") {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "token-home-list-secret") {
		t.Fatalf("stderr leaked token: %s", stderr.String())
	}
}

func TestAuthStatusReadsStoredCredentialMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", ClientID: "client-1", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}

	code := app.run([]string{"auth", "status", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-secret") {
		t.Fatalf("status leaked token: %s", stdout.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["authenticated"] != true || response["profile"] != "default" || response["houseId"] != "house-1" {
		t.Fatalf("response = %#v", response)
	}
	if _, ok := response["clientId"]; ok {
		t.Fatalf("status exposed clientId: %#v", response)
	}
}

func TestTokenOnlyProfileSupportsAuthStatusDoctorAndRuntimeContext(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--token", "Bearer token-only-secret", "--profile", "token-only", "--region", "cn", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth token set exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-only-secret") || strings.Contains(stderr.String(), "token-only-secret") {
		t.Fatalf("token leaked: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var tokenSet map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &tokenSet); err != nil {
		t.Fatalf("invalid token set json: %v", err)
	}
	if tokenSet["tokenPresent"] != true || tokenSet["region"] != "cn" || tokenSet["houseId"] != "" {
		t.Fatalf("token set response = %#v", tokenSet)
	}
	metadata, ok, err := app.metadataStore.Load("token-only")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "cn" || metadata.HouseID != "" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{"profile": "token-only"}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if !context.TokenPresent || context.AccessToken != "Bearer token-only-secret" || context.HouseID != "" {
		t.Fatalf("context = %#v", context)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"auth", "status", "--profile", "token-only", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("auth status exit code = %d, stderr = %s", code, stderr.String())
	}
	var status map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("invalid auth status json: %v", err)
	}
	if status["authenticated"] != true || status["houseId"] != "" {
		t.Fatalf("auth status = %#v", status)
	}

	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"doctor", "--profile", "token-only", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("doctor exit code = %d, stderr = %s", code, stderr.String())
	}
	var doctor map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doctor); err != nil {
		t.Fatalf("invalid doctor json: %v", err)
	}
	if doctor["status"] != "ok" || doctor["authenticated"] != true || doctor["houseId"] != "" {
		t.Fatalf("doctor = %#v", doctor)
	}
}

func TestRuntimeContextPrecedenceFlagsEnvProfileDefaults(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer token-store"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", ClientID: "client-profile", HouseID: "house-profile"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	t.Setenv("YEELIGHT_CLOUD_REGION", "sg")
	t.Setenv("YEELIGHT_HOME_HOUSE_ID", "house-env")
	t.Setenv("YEELIGHT_HOME_ACCESS_TOKEN", "Bearer token-env")

	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{
		"region":   "eu",
		"house-id": "house-flag",
	}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Region != "eu" || context.ClientID != "client-profile" || context.HouseID != "house-flag" || context.AccessToken != "Bearer token-env" {
		t.Fatalf("context = %#v", context)
	}
}

func TestRuntimeContextDefaultsToCNRegion(t *testing.T) {
	app := newTestApp(t)
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Region != "cn" {
		t.Fatalf("Region = %q", context.Region)
	}
}

func TestAuthTokenSetDoesNotWriteTokenToProfileMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"auth", "token", "set", "--token", "Bearer token-manual-secret", "--profile", "manual", "--region", "cn", "--house-id", "house-1", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "token-manual-secret") {
		t.Fatalf("stdout leaked token: %s", stdout.String())
	}
	data, err := os.ReadFile(app.metadataStore.Path())
	if err != nil {
		t.Fatalf("ReadFile metadata error: %v", err)
	}
	if strings.Contains(string(data), "token-manual-secret") || strings.Contains(string(data), "accessToken") {
		t.Fatalf("metadata leaked token: %s", string(data))
	}
	record, ok, err := app.tokenStore.Load("manual")
	if err != nil {
		t.Fatalf("Load token error: %v", err)
	}
	if !ok || record.AccessToken != "Bearer token-manual-secret" {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
}

func TestConfigSetAndHomeSelectUpdateProfileMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"config", "set", "--profile", "family", "--region", "sg", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("config set exit code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = app.run([]string{"home", "select", "--profile", "family", "--house-id", "house-selected", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("home select exit code = %d, stderr = %s", code, stderr.String())
	}
	metadata, ok, err := app.metadataStore.Load("family")
	if err != nil {
		t.Fatalf("Load metadata error: %v", err)
	}
	if !ok || metadata.Region != "sg" || metadata.ClientID != "" || metadata.HouseID != "house-selected" {
		t.Fatalf("metadata = %#v ok=%v", metadata, ok)
	}
}

func TestProfileUseSetsActiveProfile(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newTestApp(t)

	code := app.run([]string{"profile", "use", "--profile", "family", "--region", "cn", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	context, err := app.resolveRuntimeContext(cliFlags{values: map[string]string{}})
	if err != nil {
		t.Fatalf("resolveRuntimeContext error: %v", err)
	}
	if context.Profile != "family" || context.Region != "cn" {
		t.Fatalf("context = %#v", context)
	}
}

func newTestApp(t *testing.T) *app {
	t.Helper()
	tokenStore := credential.NewMemoryStore()
	return &app{
		tokenStore:    tokenStore,
		metadataStore: credential.NewFileMetadataStore(t.TempDir() + "/profiles.json"),
		planStore:     plan.NewStore(t.TempDir() + "/pending_plans.json"),
		memoryStore:   storage.NewJSONStore(t.TempDir() + "/memory.json"),
		sleep:         func(context.Context, time.Duration) error { return nil },
	}
}

type testQRClient struct {
	created       auth.QRInfo
	checked       []auth.QRInfo
	checkCalls    int
	createDevices []string
}

func (client *testQRClient) Create(_ context.Context, device string) (auth.QRInfo, error) {
	client.createDevices = append(client.createDevices, device)
	return client.created, nil
}

func (client *testQRClient) Check(context.Context, string) (auth.QRInfo, error) {
	index := client.checkCalls
	client.checkCalls++
	if index >= len(client.checked) {
		return client.checked[len(client.checked)-1], nil
	}
	return client.checked[index], nil
}
