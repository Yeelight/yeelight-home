package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPClientsCoverCommonNativeAndJSONFamilies(t *testing.T) {
	clients := MCPClients("/tmp/home")
	if len(clients) < 20 {
		t.Fatalf("MCP client count = %d", len(clients))
	}
	want := map[string]bool{
		"claude-code": false, "claude-desktop": false, "codex": false,
		"openclaw": false, "hermes-agent": false, "cursor": false,
		"vscode": false, "gemini-cli": false, "qwen-code": false,
		"kiro-cli": false, "windsurf": false, "cline": false,
		"roo": false, "opencode": false, "qclaw": false, "codebuddy": false,
		"factory-droid": false, "kilo-code": false, "zed": false, "amp": false,
	}
	for _, client := range clients {
		if _, ok := want[client.ID]; ok {
			want[client.ID] = true
		}
		if client.MCPAdapter == "" || (!client.SupportsMCP) {
			t.Fatalf("invalid MCP client = %#v", client)
		}
	}
	for id, found := range want {
		if !found {
			t.Fatalf("missing MCP client %s", id)
		}
	}
}

func TestCommonClientAliases(t *testing.T) {
	tests := map[string]string{
		"workbuddy": "codebuddy", "hermes": "hermes-agent",
		"github-copilot": "vscode", "roo-code": "roo",
		"gemini": "gemini-cli", "qwen": "qwen-code", "kiro": "kiro-cli",
		"kilo": "kilo-code", "droid": "factory-droid", "factory-ai": "factory-droid",
	}
	for input, want := range tests {
		client, ok := FindMCPClient("/tmp/home", input)
		if !ok || client.ID != want {
			t.Fatalf("FindMCPClient(%q) = %#v, %v", input, client, ok)
		}
	}
}

func TestResolveMultipleMCPClientsKeepsEachAdapter(t *testing.T) {
	client, err := ResolveClient("/tmp/home", "cursor,zed,amp,kilo", ModeMCP)
	if err != nil {
		t.Fatalf("ResolveClient error: %v", err)
	}
	if client.ID != "cursor,zed,amp,kilo-code" || len(client.MCPTargets) != 4 {
		t.Fatalf("client = %#v", client)
	}
	wantAdapters := []string{MCPAdapterStandardJSON, MCPAdapterZedJSON, MCPAdapterAmpJSON, MCPAdapterOpenCodeJSON}
	for index, target := range client.MCPTargets {
		if target.MCPAdapter != wantAdapters[index] {
			t.Fatalf("target %d = %#v", index, target)
		}
	}
}

func TestResolveMultipleMCPClientsRejectsUnknownTarget(t *testing.T) {
	_, err := ResolveClient("/tmp/home", "cursor,unknown-client", ModeMCP)
	if err == nil {
		t.Fatal("expected unknown MCP client to be rejected")
	}
}

func TestResolveMultipleStandardSkillAgents(t *testing.T) {
	client, err := ResolveClient("/tmp/home", "codex,claude-code,opencode", ModeSkill)
	if err != nil {
		t.Fatalf("ResolveClient error: %v", err)
	}
	if len(client.SkillAgents) != 3 || client.SkillAgents[0] != "codex" || client.SkillAgents[2] != "opencode" {
		t.Fatalf("client = %#v", client)
	}
}

func TestResolveMCPAutoUsesOnlyDetectedClients(t *testing.T) {
	homeDir := t.TempDir()
	cursorDir := filepath.Join(homeDir, ".cursor")
	if err := os.MkdirAll(cursorDir, 0o700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	lookup := func(command string) (string, error) {
		if command == "codex" {
			return "/usr/local/bin/codex", nil
		}
		return "", fmt.Errorf("not found")
	}
	client, err := resolveClient(homeDir, "auto", ModeMCP, lookup)
	if err != nil {
		t.Fatalf("resolveClient error: %v", err)
	}
	if len(client.MCPTargets) != 2 || client.MCPTargets[0].ID != "codex" || client.MCPTargets[1].ID != "cursor" {
		t.Fatalf("client = %#v", client)
	}
}

func TestResolveMCPAllIncludesEveryVerifiedAdapter(t *testing.T) {
	client, err := resolveClient(t.TempDir(), "all", ModeMCP, nil)
	if err != nil {
		t.Fatalf("resolveClient error: %v", err)
	}
	if len(client.MCPTargets) != len(MCPClients(t.TempDir())) || len(client.MCPTargets) < 20 {
		t.Fatalf("client count = %d", len(client.MCPTargets))
	}
}

func TestResolveMCPAutoRequiresDetectedClient(t *testing.T) {
	_, err := resolveClient(t.TempDir(), "auto", ModeMCP, func(string) (string, error) {
		return "", fmt.Errorf("not found")
	})
	if err == nil {
		t.Fatal("expected auto detection failure")
	}
}
