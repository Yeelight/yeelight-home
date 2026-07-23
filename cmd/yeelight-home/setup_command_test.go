package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/credential"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func TestSetupPlanSupportsBilingualArbitrarySkillAgents(t *testing.T) {
	for _, locale := range []string{"zh-CN", "en-US"} {
		t.Run(locale, func(t *testing.T) {
			app := newTestApp(t)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := app.run([]string{"setup", "--lang", locale, "--mode", "skill", "--agent", "future-agent", "--plan", "--json", "--home-dir", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("code = %d, stderr = %s", code, stderr.String())
			}
			var plan setupdomain.Plan
			if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
				t.Fatalf("Unmarshal plan error: %v", err)
			}
			if plan.Locale != locale || plan.Client.ID != "future-agent" || plan.Steps[2].Method != setupdomain.MethodSkillsCLI {
				t.Fatalf("plan = %#v", plan)
			}
		})
	}
}

func TestSetupInteractiveDefaultsToSkillAndCanCancel(t *testing.T) {
	app := newTestApp(t)
	app.terminal = func(io.Reader) bool { return true }
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"setup", "--lang", "zh-CN", "--agent", "codex", "--home-dir", t.TempDir()}, strings.NewReader("\nn\n"), &stdout, &stderr)
	if code != exitOK || !strings.Contains(stdout.String(), "已取消安装") {
		t.Fatalf("code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
}

func TestSetupInteractiveSkillLetsUserChooseDetectedAgent(t *testing.T) {
	homeDir := t.TempDir()
	if err := os.MkdirAll(homeDir+"/.qclaw", 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	app := newTestApp(t)
	app.terminal = func(io.Reader) bool { return true }
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"setup", "--lang", "zh-CN", "--home-dir", homeDir}, strings.NewReader("\nqclaw\nn\n"), &stdout, &stderr)
	if code != exitOK || !strings.Contains(strings.ToLower(stdout.String()), "qclaw") || !strings.Contains(stdout.String(), "已取消安装") {
		t.Fatalf("code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
}

func TestSetupPromptSupportsDefaultAndMultipleSkillAgents(t *testing.T) {
	for name, input := range map[string]string{"default": "\n", "multiple": "1,2\n"} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader(input)), stdout: &stdout}
			clients := []setupdomain.Client{{ID: "codex", Name: "Codex"}, {ID: "openclaw", Name: "OpenClaw"}}
			selected, err := prompt.chooseSkillAgents("en-US", clients)
			if err != nil {
				t.Fatalf("chooseSkillAgents error: %v", err)
			}
			want := "auto"
			if name == "multiple" {
				want = "codex,openclaw"
			}
			if selected != want || !strings.Contains(strings.ToLower(stdout.String()), "codex") {
				t.Fatalf("selected=%q want=%q output=%s", selected, want, stdout.String())
			}
		})
	}
}

func TestSetupPromptAcceptsAgentIDWhenNoneWasDetected(t *testing.T) {
	var stdout bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("codex\n")), stdout: &stdout}
	selected, err := prompt.chooseSkillAgents("en-US", nil)
	if err != nil || selected != "codex" || !strings.Contains(stdout.String(), "No AI") {
		t.Fatalf("selected=%q err=%v output=%s", selected, err, stdout.String())
	}
}

func TestSetupPromptLetsUserReuseOrSwitchAccountsInBothLanguages(t *testing.T) {
	tests := []struct {
		name      string
		locale    string
		input     string
		wantReuse bool
		wantText  string
		wantErr   string
	}{
		{name: "zh default", locale: "zh-CN", input: "\n", wantReuse: true, wantText: "继续使用当前账号"},
		{name: "zh switch", locale: "zh-CN", input: "2\n", wantReuse: false, wantText: "重新扫码并切换账号"},
		{name: "en keep", locale: "en-US", input: "1\n", wantReuse: true, wantText: "Keep using the current account"},
		{name: "en invalid", locale: "en-US", input: "3\n", wantText: "Scan again and switch accounts", wantErr: "Choose 1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader(test.input)), stdout: &stdout}
			got, err := prompt.reuseCurrentAccount(test.locale)
			if got != test.wantReuse || !strings.Contains(stdout.String(), test.wantText) {
				t.Fatalf("reuse=%t output=%q", got, stdout.String())
			}
			if test.wantErr == "" && err != nil {
				t.Fatalf("reuseCurrentAccount error: %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("reuseCurrentAccount error=%v", err)
			}
		})
	}
}

func TestSetupAuthenticatedAccountDecisionDefaultsToReuseOutsideInteractiveMode(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer existing-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	reuse, err := app.reuseCurrentSetupAccount(setupExecutionOptions{Profile: "default", Locale: "en-US"})
	if err != nil || !reuse {
		t.Fatalf("reuse=%t err=%v", reuse, err)
	}
}

func TestSetupInteractiveExistingAccountDefaultsToReuseWithoutStartingQR(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer existing-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	client := &testQRClient{}
	app.qrClient = client
	var promptOutput bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("\n")), stdout: &promptOutput}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Locale: "en-US", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err != nil || result.Status != "skipped" || result.Message != "already authenticated" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if len(client.createDevices) != 0 {
		t.Fatalf("QR client was called: %#v", client.createDevices)
	}
	if !strings.Contains(promptOutput.String(), "Keep using the current account") {
		t.Fatalf("prompt output=%q", promptOutput.String())
	}
}

func TestSetupUnauthenticatedAccountStartsQRWithoutReusePrompt(t *testing.T) {
	app := newTestApp(t)
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-first-login", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{QRCodeID: "qr-first-login", Status: "LOGIN", Token: auth.QRToken{AccessToken: "first-secret"}}},
	}
	var promptOutput bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("")), stdout: &promptOutput}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Region: "dev", Locale: "zh-CN", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err != nil {
		t.Fatalf("executeSetupStep error: %v", err)
	}
	if strings.Contains(promptOutput.String(), "继续使用当前账号") {
		t.Fatalf("unexpected account reuse prompt=%q", promptOutput.String())
	}
}

func TestSetupInteractiveReauthenticationClearsPreviousAccountHome(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "old-house", ClientID: "old-client"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-switch", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{QRCodeID: "qr-switch", Status: "LOGIN", Token: auth.QRToken{AccessToken: "new-secret"}}},
	}
	var promptOutput bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: &promptOutput}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Region: "dev", Locale: "zh-CN", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err != nil {
		t.Fatalf("executeSetupStep error: %v", err)
	}
	record, ok, err := app.tokenStore.Load("default")
	if err != nil || !ok || record.AccessToken != "Bearer new-secret" {
		t.Fatalf("token=%#v ok=%t err=%v", record, ok, err)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "" || metadata.ClientID != "" {
		t.Fatalf("metadata=%#v ok=%t err=%v", metadata, ok, err)
	}
	if !strings.Contains(promptOutput.String(), "重新扫码并切换账号") {
		t.Fatalf("prompt output=%q", promptOutput.String())
	}
}

func TestSetupInteractiveAccountSwitchSelectsHomeFromNewAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"new-house-1","name":"新家"},{"id":"new-house-2","name":"父母家"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", BizType: "0", HouseID: "old-house", ClientID: "old-client"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-new-account", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{QRCodeID: "qr-new-account", Status: "LOGIN", Token: auth.QRToken{AccessToken: "new-secret", ClientID: "new-client"}}},
	}
	app.process = func(context.Context, []string, io.Writer, io.Writer) error { return nil }
	plan, err := setupdomain.BuildPlan(setupdomain.Options{Locale: "zh-CN", ClientID: "codex", Mode: setupdomain.ModeSkill, BizType: "0", HomeDir: t.TempDir()})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	var promptOutput bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n2\n")), stdout: &promptOutput}
	result, err := app.executeSetupPlan(plan, setupExecutionOptions{
		Profile: "default", Region: "dev", BizType: "0", Locale: "zh-CN", HomeDir: t.TempDir(),
		Interactive: true, Prompt: prompt, Stdout: io.Discard, Stderr: io.Discard,
	})
	if err != nil || !result.OK {
		t.Fatalf("result=%#v err=%v output=%q", result, err, promptOutput.String())
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "new-house-2" || metadata.ClientID != "new-client" {
		t.Fatalf("metadata=%#v ok=%t err=%v", metadata, ok, err)
	}
	if !strings.Contains(promptOutput.String(), "父母家") {
		t.Fatalf("prompt output=%q", promptOutput.String())
	}
}

func TestSetupFailedReauthenticationKeepsPreviousCredentialsAndHome(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "old-house", ClientID: "old-client"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-expired", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{QRCodeID: "qr-expired", Status: "EXPIRED"}},
	}
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: io.Discard}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Region: "dev", Locale: "en-US", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err == nil {
		t.Fatal("expected QR login failure")
	}
	record, ok, loadErr := app.tokenStore.Load("default")
	if loadErr != nil || !ok || record.AccessToken != "Bearer old-secret" {
		t.Fatalf("token=%#v ok=%t err=%v", record, ok, loadErr)
	}
	metadata, ok, loadErr := app.metadataStore.Load("default")
	if loadErr != nil || !ok || metadata.HouseID != "old-house" || metadata.ClientID != "old-client" {
		t.Fatalf("metadata=%#v ok=%t err=%v", metadata, ok, loadErr)
	}
}

func TestSetupMetadataSaveFailureRollsBackPreviousCredentialAndHome(t *testing.T) {
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer old-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	oldMetadata := credential.ProfileMetadata{
		Profile: "default", Region: "dev", HouseID: "old-house", ClientID: "old-client", QRDevice: "F8:24:41:00:00:01",
	}
	if err := app.metadataStore.Save(oldMetadata); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	baseStore := credential.NewFileMetadataStore(app.metadataStore.Path())
	app.metadataStore = &failOnceMetadataStore{FileMetadataStore: baseStore, err: errors.New("metadata storage unavailable")}
	app.qrClient = &testQRClient{
		created: auth.QRInfo{QRCodeID: "qr-storage-failure", Status: "CREATED", ExpireAt: time.Now().Add(time.Minute).UnixMilli()},
		checked: []auth.QRInfo{{
			QRCodeID: "qr-storage-failure", Status: "LOGIN",
			Token: auth.QRToken{AccessToken: "new-secret", ClientID: "new-client"},
		}},
	}
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: io.Discard}
	step := setupdomain.Step{ID: "login", Method: setupdomain.MethodAuthQR}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(setupdomain.Plan{}, step, setupExecutionOptions{
		Profile: "default", Region: "dev", Locale: "en-US", Interactive: true, Prompt: prompt,
		Stdout: io.Discard, Stderr: io.Discard,
	}, &result)
	if err == nil || !strings.Contains(err.Error(), "QR login returned exit code") {
		t.Fatalf("error=%v", err)
	}
	record, ok, loadErr := app.tokenStore.Load("default")
	if loadErr != nil || !ok || record.AccessToken != "Bearer old-secret" {
		t.Fatalf("token=%#v ok=%t err=%v", record, ok, loadErr)
	}
	metadata, ok, loadErr := app.metadataStore.Load("default")
	if loadErr != nil || !ok || metadata.HouseID != "old-house" || metadata.ClientID != "old-client" {
		t.Fatalf("metadata=%#v ok=%t err=%v", metadata, ok, loadErr)
	}
}

type failOnceMetadataStore struct {
	credential.FileMetadataStore
	err error
}

func (store *failOnceMetadataStore) Save(metadata credential.ProfileMetadata) error {
	if store.err != nil {
		err := store.err
		store.err = nil
		return err
	}
	return store.FileMetadataStore.Save(metadata)
}

func TestSetupRejectsUnknownFlag(t *testing.T) {
	app := newTestApp(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"setup", "--lang", "en-US", "--mode", "skill", "--unknown", "value", "--plan"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput || stdout.Len() != 0 || !strings.Contains(stderr.String(), "unsupported flag") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestSetupYesRunsSkillAndReadOnlyVerificationWithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path != "/apis/iot/v1/house/r/all" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"list":[{"id":"house-1","name":"Home"}]}}`))
	}))
	defer server.Close()
	t.Setenv("YEELIGHT_API_BASE_URL", server.URL+"/apis/iot")
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer setup-secret"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var commands [][]string
	app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
		commands = append(commands, append([]string(nil), command...))
		return nil
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.run([]string{"setup", "--lang", "en-US", "--mode", "skill", "--agent", "codex", "--yes", "--json", "--home-dir", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if len(commands) != 1 || commands[0][0] != "npx" {
		t.Fatalf("commands = %#v", commands)
	}
	if strings.Contains(stdout.String(), "setup-secret") || strings.Contains(stderr.String(), "setup-secret") {
		t.Fatalf("setup leaked token: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	var result setupdomain.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.OK {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "house-1" || metadata.Language != "en-US" {
		t.Fatalf("selected metadata = %#v, ok = %v, err = %v", metadata, ok, err)
	}
}

func TestSetupDefaultHomeSelectionPreservesExistingChoice(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", HouseID: "house-existing"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	if err := app.selectDefaultSetupHome([]byte(`{"houses":[{"id":"house-first"}]}`), setupExecutionOptions{Profile: "default", Region: "dev"}); err != nil {
		t.Fatalf("selectDefaultSetupHome error: %v", err)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "house-existing" {
		t.Fatalf("selected metadata = %#v, ok = %v, err = %v", metadata, ok, err)
	}
}

func TestSetupDefaultHomeSelectionLetsInteractiveUserChoose(t *testing.T) {
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "dev", Language: "zh-CN"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	var promptOutput bytes.Buffer
	prompt := &setupPrompt{reader: bufio.NewReader(strings.NewReader("2\n")), stdout: &promptOutput}
	data := []byte(`{"houses":[{"id":"house-first","name":"家"},{"id":"house-second","name":"父母家"}]}`)
	if err := app.selectDefaultSetupHome(data, setupExecutionOptions{Profile: "default", Region: "dev", Interactive: true, Prompt: prompt}); err != nil {
		t.Fatalf("selectDefaultSetupHome error: %v", err)
	}
	metadata, ok, err := app.metadataStore.Load("default")
	if err != nil || !ok || metadata.HouseID != "house-second" {
		t.Fatalf("selected metadata = %#v, ok = %v, err = %v", metadata, ok, err)
	}
	if !strings.Contains(promptOutput.String(), "父母家") {
		t.Fatalf("prompt output = %s", promptOutput.String())
	}
}

func TestSetupLANVerificationFailureBlocksCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "gateway unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	app := newTestApp(t)
	if err := app.metadataStore.Save(credential.ProfileMetadata{
		Profile: "default", Region: "dev", ControlMode: controlModeLocalOnly, LANEndpoint: server.URL + "/mcp",
	}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	plan := setupdomain.Plan{Locale: "en-US", Mode: setupdomain.ModeLAN, ControlMode: setupdomain.ControlModeLocalOnly}
	step := setupdomain.Step{ID: "verify", Method: setupdomain.MethodVerify}
	result := setupdomain.StepResult{ID: step.ID}
	err := app.executeSetupStep(plan, step, setupExecutionOptions{Profile: "default", Region: "dev", Stdout: io.Discard, Stderr: io.Discard}, &result)
	if err == nil || !strings.Contains(err.Error(), "LAN gateway verification") {
		t.Fatalf("verification error = %v", err)
	}
}

func TestMCPJSONMergePreservesExistingConfigAndUsesProtectedPermissions(t *testing.T) {
	path := t.TempDir() + "/mcp.json"
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"existing":{"url":"http://127.0.0.1:9000/mcp"}},"theme":"dark"}`), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	servers := []setupMCPServer{{Name: "yeelight-iot", URL: "https://api.yeelight.com/apis/mcp_server/v1/mcp", Headers: map[string]string{"Authorization": "Bearer secret"}}}
	if err := writeMCPServersJSON(path, servers); err != nil {
		t.Fatalf("writeMCPServersJSON error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !bytes.Contains(data, []byte("existing")) || !bytes.Contains(data, []byte("yeelight-iot")) || !bytes.Contains(data, []byte("dark")) {
		t.Fatalf("config = %s", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
}

func TestSkillInstallerDoesNotTreatPartialAgentFailureAsSuccess(t *testing.T) {
	app := newTestApp(t)
	app.process = func(_ context.Context, _ []string, _ io.Writer, stderr io.Writer) error {
		_, _ = fmt.Fprintln(stderr, "PromptScript: PromptScript does not support global skill installation")
		return nil
	}
	step := setupdomain.Step{
		Method:  setupdomain.MethodSkillsCLI,
		Command: []string{"npx", "-y", "skills", "add", "https://example.com/skills", "--global", "--yes"},
	}
	err := app.runSkillInstaller(step, setupExecutionOptions{Stdout: io.Discard, Stderr: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "partial Skill installation failure") {
		t.Fatalf("runSkillInstaller error = %v", err)
	}
}

func TestSkillInstallerFallsBackAcrossOfficialMirrors(t *testing.T) {
	app := newTestApp(t)
	var commands [][]string
	app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
		commands = append(commands, append([]string(nil), command...))
		if len(commands) < 3 {
			return errors.New("source unavailable")
		}
		return nil
	}
	step := setupdomain.Step{
		Method:  setupdomain.MethodSkillsCLI,
		Command: []string{"npx", "-y", "skills", "add", "https://github.com/Yeelight/yeelight-smart-home-skills", "--global", "--yes", "--agent", "codex"},
		Sources: []string{
			"https://github.com/Yeelight/yeelight-smart-home-skills",
			"https://gitee.com/yeelight/yeelight-smart-home-skills.git",
			"https://gitcode.com/Yeelight/yeelight-smart-home-skills.git",
		},
	}
	if err := app.runSkillInstaller(step, setupExecutionOptions{Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatalf("runSkillInstaller error: %v", err)
	}
	if len(commands) != 3 || !slices.Contains(commands[1], step.Sources[1]) || !slices.Contains(commands[2], step.Sources[2]) {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestSkillInstallerFallsBackWhenSkillsCLIReportsNoSkillsWithExitZero(t *testing.T) {
	app := newTestApp(t)
	var commands [][]string
	app.process = func(_ context.Context, command []string, stdout io.Writer, _ io.Writer) error {
		commands = append(commands, append([]string(nil), command...))
		if len(commands) == 1 {
			_, _ = fmt.Fprintln(stdout, "No skills found at this URL.")
		}
		return nil
	}
	step := setupdomain.Step{
		Method:  setupdomain.MethodSkillsCLI,
		Command: []string{"npx", "-y", "skills", "add", "https://example.com/first", "--global", "--yes", "--agent", "codex"},
		Sources: []string{"https://example.com/first", "https://example.com/second.git"},
	}
	if err := app.runSkillInstaller(step, setupExecutionOptions{Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatalf("runSkillInstaller error: %v", err)
	}
	if len(commands) != 2 || !slices.Contains(commands[1], step.Sources[1]) {
		t.Fatalf("commands = %#v", commands)
	}
}
