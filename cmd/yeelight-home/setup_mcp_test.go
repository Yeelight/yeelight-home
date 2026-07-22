package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeelight/yeelight-home/internal/credential"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func TestClientSpecificMCPJSONFormats(t *testing.T) {
	servers := []setupMCPServer{{
		Name: "yeelight-iot", URL: "https://api.example.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer test-token"},
	}}
	tests := []struct {
		name      string
		write     func(string, []setupMCPServer) error
		topLevel  string
		entryKeys []string
	}{
		{name: "zed", write: writeZedMCPJSON, topLevel: "context_servers", entryKeys: []string{"url", "headers"}},
		{name: "amp", write: writeAmpMCPJSON, topLevel: "amp.mcpServers", entryKeys: []string{"url", "headers"}},
		{name: "kilo", write: writeOpenCodeMCPJSON, topLevel: "mcp", entryKeys: []string{"type", "url", "enabled", "headers"}},
		{name: "claude-code", write: writeClaudeCodeMCPJSON, topLevel: "mcpServers", entryKeys: []string{"type", "url", "headers"}},
		{name: "factory-droid", write: writeFactoryDroidMCPJSON, topLevel: "mcpServers", entryKeys: []string{"type", "url", "headers", "disabled"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(`{"theme":"existing"}`), 0o600); err != nil {
				t.Fatalf("WriteFile error: %v", err)
			}
			if err := test.write(path, servers); err != nil {
				t.Fatalf("write error: %v", err)
			}
			document := readJSONObject(t, path)
			if document["theme"] != "existing" {
				t.Fatalf("existing config was not preserved: %#v", document)
			}
			entries, ok := document[test.topLevel].(map[string]any)
			if !ok {
				t.Fatalf("%s = %#v", test.topLevel, document[test.topLevel])
			}
			entry, ok := entries["yeelight-iot"].(map[string]any)
			if !ok {
				t.Fatalf("entry = %#v", entries["yeelight-iot"])
			}
			for _, key := range test.entryKeys {
				if _, ok := entry[key]; !ok {
					t.Fatalf("entry missing %s: %#v", key, entry)
				}
			}
		})
	}
}

func TestConfigureOneMCPClientRoutesNewAdapters(t *testing.T) {
	app := newTestApp(t)
	servers := []setupMCPServer{{Name: "yeelight-lan", URL: "http://192.168.1.2:18080/mcp"}}
	tests := []struct {
		adapter string
		field   string
	}{
		{adapter: setupdomain.MCPAdapterZedJSON, field: "context_servers"},
		{adapter: setupdomain.MCPAdapterAmpJSON, field: "amp.mcpServers"},
		{adapter: setupdomain.MCPAdapterOpenCodeJSON, field: "mcp"},
		{adapter: setupdomain.MCPAdapterClaudeCode, field: "mcpServers"},
		{adapter: setupdomain.MCPAdapterFactoryDroid, field: "mcpServers"},
	}
	for _, test := range tests {
		path := filepath.Join(t.TempDir(), "config.json")
		client := setupdomain.Client{ID: test.adapter, Name: test.adapter, MCPAdapter: test.adapter, MCPConfigPath: path}
		if err := app.configureOneMCPClient(client, servers, setupExecutionOptions{Stderr: io.Discard}); err != nil {
			t.Fatalf("configureOneMCPClient(%s) error: %v", test.adapter, err)
		}
		if _, ok := readJSONObject(t, path)[test.field]; !ok {
			t.Fatalf("%s missing from %s", test.field, path)
		}
	}
}

func TestLocalMCPServerUsesRuntimeWithoutCredentials(t *testing.T) {
	servers, err := setupMCPServers(setupdomain.MCPSourceLocal, runtimeContext{
		Profile: "family", Region: "cn", HouseID: "house-1", Language: "zh-CN",
		AccessToken: "Bearer must-not-be-used",
	})
	if err != nil || len(servers) != 1 {
		t.Fatalf("servers = %#v, err = %v", servers, err)
	}
	server := servers[0]
	if server.Name != "yeelight-home" || server.Command != "yeelight-home" || server.URL != "" || len(server.Headers) != 0 {
		t.Fatalf("server = %#v", server)
	}
	joined := strings.Join(server.Args, " ")
	if !strings.Contains(joined, "mcp serve --stdio") || !strings.Contains(joined, "--profile family") || strings.Contains(joined, "must-not-be-used") {
		t.Fatalf("args = %#v", server.Args)
	}
}

func TestCloudMCPServersUseStableOrderAndCredentialProxy(t *testing.T) {
	servers, err := setupMCPServers(setupdomain.MCPSourceCloud, runtimeContext{
		Profile: "family", Region: "cn", HouseID: "house-1", AccessToken: "Bearer test-token",
	})
	if err != nil || len(servers) != 2 {
		t.Fatalf("servers = %#v, err = %v", servers, err)
	}
	if servers[0].Name != "yeelight-metadata" || servers[1].Name != "yeelight-iot" {
		t.Fatalf("server order = %#v", servers)
	}
	for _, server := range servers {
		joined := strings.Join(server.Args, " ")
		if server.Command != "yeelight-home" || server.URL != "" || len(server.Headers) != 0 ||
			!strings.Contains(joined, "mcp proxy --stdio") || !strings.Contains(joined, "--profile family") ||
			!strings.Contains(joined, "--house-id house-1") || strings.Contains(joined, "test-token") {
			t.Fatalf("server %s = %#v", server.Name, server)
		}
	}
	if strings.Join(servers[0].Args, " ") == strings.Join(servers[1].Args, " ") {
		t.Fatalf("cloud targets must use distinct proxy arguments: %#v", servers)
	}
}

func TestConfigureMCPClientWritesLocalRuntimeInIsolatedHome(t *testing.T) {
	homeDir := t.TempDir()
	app := newTestApp(t)
	plan, err := setupdomain.BuildPlan(setupdomain.Options{
		Locale: "en-US", ClientID: "qclaw", Mode: setupdomain.ModeMCP, HomeDir: homeDir,
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if err := app.configureMCPClient(plan, setupExecutionOptions{Profile: "default", Region: "cn", HomeDir: homeDir, Stderr: io.Discard}); err != nil {
		t.Fatalf("configureMCPClient error: %v", err)
	}
	data, err := os.ReadFile(plan.Client.MCPConfigPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !bytes.Contains(data, []byte("yeelight-home")) || !bytes.Contains(data, []byte("--stdio")) || bytes.Contains(data, []byte("Authorization")) {
		t.Fatalf("config = %s", data)
	}
}

func TestConfigureMCPClientWritesCloudServicesInIsolatedHome(t *testing.T) {
	homeDir := t.TempDir()
	app := newTestApp(t)
	if err := app.tokenStore.Save(credential.TokenRecord{Profile: "default", AccessToken: "Bearer cloud-token"}); err != nil {
		t.Fatalf("Save token error: %v", err)
	}
	if err := app.metadataStore.Save(credential.ProfileMetadata{Profile: "default", Region: "cn", HouseID: "house-1"}); err != nil {
		t.Fatalf("Save metadata error: %v", err)
	}
	plan, err := setupdomain.BuildPlan(setupdomain.Options{
		Locale: "en-US", ClientID: "qclaw", Mode: setupdomain.ModeMCP, MCPSource: "cloud", HomeDir: homeDir,
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if err := app.configureMCPClient(plan, setupExecutionOptions{Profile: "default", Region: "cn", HomeDir: homeDir, Stderr: io.Discard}); err != nil {
		t.Fatalf("configureMCPClient error: %v", err)
	}
	data, err := os.ReadFile(plan.Client.MCPConfigPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	metadataIndex := bytes.Index(data, []byte("yeelight-metadata"))
	iotIndex := bytes.Index(data, []byte("yeelight-iot"))
	if metadataIndex < 0 || iotIndex < 0 || !bytes.Contains(data, []byte("mcp")) || !bytes.Contains(data, []byte("proxy")) ||
		!bytes.Contains(data, []byte("house-1")) || bytes.Contains(data, []byte("Bearer cloud-token")) || bytes.Contains(data, []byte("Authorization")) {
		t.Fatalf("config = %s", data)
	}
}

func TestLocalMCPJSONAdaptersUseStdioAndDoNotWriteAuthorization(t *testing.T) {
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	tests := []struct {
		name  string
		write func(string, []setupMCPServer) error
		field string
	}{
		{name: "standard", write: writeMCPServersJSON, field: "mcpServers"},
		{name: "claude-desktop", write: writeClaudeDesktopMCPJSON, field: "mcpServers"},
		{name: "vscode", write: writeVSCodeMCPJSON, field: "servers"},
		{name: "gemini", write: writeGeminiMCPJSON, field: "mcpServers"},
		{name: "opencode", write: writeOpenCodeMCPJSON, field: "mcp"},
		{name: "zed", write: writeZedMCPJSON, field: "context_servers"},
		{name: "amp", write: writeAmpMCPJSON, field: "amp.mcpServers"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := test.write(path, []setupMCPServer{server}); err != nil {
				t.Fatalf("write error: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile error: %v", err)
			}
			if bytes.Contains(data, []byte("Authorization")) || !bytes.Contains(data, []byte("yeelight-home")) || !bytes.Contains(data, []byte("--stdio")) {
				t.Fatalf("config = %s", data)
			}
			if readJSONObject(t, path)[test.field] == nil {
				t.Fatalf("missing %s in %s", test.field, data)
			}
		})
	}
}

func TestWindowsMCPConfigReplacementRestoresDestinationOnFailure(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source.json")
	destination := filepath.Join(directory, "destination.json")
	if err := os.WriteFile(source, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile source error: %v", err)
	}
	if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile destination error: %v", err)
	}
	calls := 0
	rename := func(from, to string) error {
		calls++
		if calls == 2 {
			return errors.New("replace failed")
		}
		return os.Rename(from, to)
	}
	if err := replaceMCPConfigFileWindows(source, destination, rename); err == nil {
		t.Fatal("replaceMCPConfigFileWindows unexpectedly succeeded")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "old" {
		t.Fatalf("restored destination = %q, err = %v", data, err)
	}
}

func TestMCPJSONWriterRejectsWrongSectionTypeWithoutChangingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := []byte(`{"theme":"existing","mcpServers":[]}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	if err := writeMCPServersJSON(path, []setupMCPServer{server}); err == nil || !strings.Contains(err.Error(), `field "mcpServers" must be a JSON object`) {
		t.Fatalf("writeMCPServersJSON error = %v", err)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !bytes.Equal(current, original) {
		t.Fatalf("config changed: got %s, want %s", current, original)
	}
}

func TestMCPJSONWriterInitializesEmptyExistingConfig(t *testing.T) {
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	for _, content := range []string{"", " \n\t"} {
		path := filepath.Join(t.TempDir(), "mcp.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}
		if err := writeMCPServersJSON(path, []setupMCPServer{server}); err != nil {
			t.Fatalf("writeMCPServersJSON(%q) error: %v", content, err)
		}
		if readJSONObject(t, path)["mcpServers"] == nil {
			t.Fatalf("mcpServers missing from initialized config: %s", path)
		}
	}
}

func TestMCPJSONWriterRejectsNonEmptyInvalidJSONWithoutChangingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	original := []byte(`{"mcpServers":`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	err := writeMCPServersJSON(path, []setupMCPServer{server})
	if err == nil || !strings.Contains(err.Error(), "parse existing MCP config") {
		t.Fatalf("writeMCPServersJSON error = %v", err)
	}
	current, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(current, original) {
		t.Fatalf("config changed: got %s, err = %v", current, readErr)
	}
}

func TestVerifyMCPConfigUnchangedDetectsConcurrentEdit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := []byte(`{"theme":"before"}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"after"}`), 0o600); err != nil {
		t.Fatalf("WriteFile concurrent edit error: %v", err)
	}
	if err := verifyMCPConfigUnchanged(path, original, true); err == nil || !strings.Contains(err.Error(), "changed during setup") {
		t.Fatalf("verifyMCPConfigUnchanged error = %v", err)
	}
}

func TestWindowsMCPConfigReplacementReportsRestoreFailure(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source.json")
	destination := filepath.Join(directory, "destination.json")
	if err := os.WriteFile(source, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile source error: %v", err)
	}
	if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile destination error: %v", err)
	}
	calls := 0
	rename := func(from, to string) error {
		calls++
		if calls == 2 {
			return errors.New("replace failed")
		}
		if calls == 3 {
			return errors.New("restore failed")
		}
		return os.Rename(from, to)
	}
	err := replaceMCPConfigFileWindows(source, destination, rename)
	if err == nil || !strings.Contains(err.Error(), "replace failed") || !strings.Contains(err.Error(), "restore previous MCP config") {
		t.Fatalf("replaceMCPConfigFileWindows error = %v", err)
	}
}

func readJSONObject(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	return document
}

func stringSlicesToBytes(values []string) [][]byte {
	result := make([][]byte, 0, len(values))
	for _, value := range values {
		result = append(result, []byte(value))
	}
	return result
}
