package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
	code := app.run([]string{"setup", "--lang", "zh-CN", "--home-dir", t.TempDir()}, strings.NewReader("\nn\n"), &stdout, &stderr)
	if code != exitOK || !strings.Contains(stdout.String(), "已取消安装") {
		t.Fatalf("code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
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
	if err != nil || !ok || metadata.HouseID != "house-1" {
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
