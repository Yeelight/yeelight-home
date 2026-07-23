package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	MCPAdapterClaudeCode    = "claude-code-cli"
	MCPAdapterCodex         = "codex-cli"
	MCPAdapterOpenClaw      = "openclaw-cli"
	MCPAdapterHermes        = "hermes-cli"
	MCPAdapterFactoryDroid  = "factory-droid-cli"
	MCPAdapterClaudeDesktop = "claude-desktop-json"
	MCPAdapterStandardJSON  = "mcp-servers-json"
	MCPAdapterVSCodeJSON    = "vscode-servers-json"
	MCPAdapterGeminiJSON    = "gemini-mcp-json"
	MCPAdapterOpenCodeJSON  = "opencode-mcp-json"
	MCPAdapterZedJSON       = "zed-context-servers-json"
	MCPAdapterAmpJSON       = "amp-mcp-servers-json"
)

func MCPClients(homeDir string) []Client {
	paths := platformMCPPaths(homeDir)
	return []Client{
		mcpClient("claude-code", "Claude Code", MCPAdapterClaudeCode, filepath.Join(homeDir, ".claude.json"), "claude-code"),
		mcpClient("claude-desktop", "Claude Desktop", MCPAdapterClaudeDesktop, paths.claudeDesktop, ""),
		mcpClient("codex", "Codex", MCPAdapterCodex, "", "codex"),
		mcpClient("openclaw", "OpenClaw", MCPAdapterOpenClaw, "", "openclaw"),
		mcpClient("hermes-agent", "Hermes Agent", MCPAdapterHermes, "", "hermes-agent"),
		mcpClient("factory-droid", "Factory Droid", MCPAdapterFactoryDroid, filepath.Join(homeDir, ".factory", "mcp.json"), ""),
		mcpClient("cursor", "Cursor", MCPAdapterStandardJSON, filepath.Join(homeDir, ".cursor", "mcp.json"), "cursor"),
		mcpClient("vscode", "VS Code / GitHub Copilot", MCPAdapterVSCodeJSON, paths.vscode, "github-copilot"),
		mcpClient("gemini-cli", "Gemini CLI", MCPAdapterGeminiJSON, filepath.Join(homeDir, ".gemini", "settings.json"), "gemini-cli"),
		mcpClient("qwen-code", "Qwen Code", MCPAdapterGeminiJSON, filepath.Join(homeDir, ".qwen", "settings.json"), "qwen-code"),
		mcpClient("kiro-cli", "Kiro CLI", MCPAdapterStandardJSON, filepath.Join(homeDir, ".kiro", "settings", "mcp.json"), "kiro-cli"),
		mcpClient("kilo-code", "Kilo Code", MCPAdapterOpenCodeJSON, filepath.Join(homeDir, ".config", "kilo", "kilo.json"), "kilo-code"),
		mcpClient("zed", "Zed", MCPAdapterZedJSON, paths.zed, "zed"),
		mcpClient("amp", "Amp", MCPAdapterAmpJSON, filepath.Join(homeDir, ".config", "amp", "settings.json"), "amp"),
		mcpClient("windsurf", "Windsurf", MCPAdapterStandardJSON, filepath.Join(homeDir, ".codeium", "windsurf", "mcp_config.json"), "windsurf"),
		mcpClient("cline", "Cline", MCPAdapterStandardJSON, filepath.Join(paths.codeUser, "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), "cline"),
		mcpClient("roo", "Roo Code", MCPAdapterStandardJSON, filepath.Join(paths.codeUser, "globalStorage", "rooveterinaryinc.roo-cline", "settings", "mcp_settings.json"), "roo"),
		mcpClient("opencode", "OpenCode", MCPAdapterOpenCodeJSON, filepath.Join(paths.configHome, "opencode", "opencode.json"), "opencode"),
		{ID: "qclaw", Name: "QClaw", MCPAdapter: MCPAdapterStandardJSON, MCPConfigPath: filepath.Join(homeDir, ".qclaw", "mcp.json"), SupportsSkill: true, SupportsMCP: true, SkillPath: filepath.Join(homeDir, ".qclaw", "skills", "yeelight-smart-home")},
		mcpClient("codebuddy", "WorkBuddy / CodeBuddy", MCPAdapterStandardJSON, filepath.Join(homeDir, ".codebuddy", ".mcp.json"), "codebuddy"),
	}
}

func mcpClient(id string, name string, adapter string, configPath string, skillAgent string) Client {
	client := Client{ID: id, Name: name, MCPAdapter: adapter, MCPConfigPath: configPath, SupportsMCP: true}
	if skillAgent != "" {
		client.SupportsSkill = true
		client.SkillAgents = []string{skillAgent}
	}
	return client
}

type mcpPaths struct {
	configHome    string
	codeUser      string
	claudeDesktop string
	vscode        string
	zed           string
}

func platformMCPPaths(homeDir string) mcpPaths {
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}
	paths := mcpPaths{configHome: configHome}
	switch runtime.GOOS {
	case "darwin":
		applicationSupport := filepath.Join(homeDir, "Library", "Application Support")
		paths.codeUser = filepath.Join(applicationSupport, "Code", "User")
		paths.claudeDesktop = filepath.Join(applicationSupport, "Claude", "claude_desktop_config.json")
		paths.zed = filepath.Join(configHome, "zed", "settings.json")
	case "windows":
		appData := strings.TrimSpace(os.Getenv("APPDATA"))
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		paths.codeUser = filepath.Join(appData, "Code", "User")
		paths.claudeDesktop = filepath.Join(appData, "Claude", "claude_desktop_config.json")
		paths.zed = filepath.Join(appData, "Zed", "settings.json")
	default:
		paths.codeUser = filepath.Join(configHome, "Code", "User")
		paths.claudeDesktop = filepath.Join(configHome, "Claude", "claude_desktop_config.json")
		paths.zed = filepath.Join(configHome, "zed", "settings.json")
	}
	paths.vscode = filepath.Join(paths.codeUser, "mcp.json")
	return paths
}

func FindMCPClient(homeDir string, id string) (Client, bool) {
	normalized := normalizeAgentID(id)
	for _, client := range MCPClients(homeDir) {
		if client.ID == normalized {
			return client, true
		}
	}
	return Client{}, false
}

func ResolveClient(homeDir string, id string, mode Mode) (Client, error) {
	return resolveClient(homeDir, id, mode, exec.LookPath)
}

func resolveClient(homeDir string, id string, mode Mode, lookPath func(string) (string, error)) (Client, error) {
	if strings.Contains(id, ",") {
		agents := normalizeAgentIDs(strings.Split(id, ","))
		if len(agents) == 0 {
			return Client{}, fmt.Errorf("at least one Agent id is required")
		}
		if mode == ModeMCP {
			return resolveMCPClients(homeDir, agents)
		}
		for _, agent := range agents {
			if agent == "qclaw" || agent == "claude-desktop" {
				return Client{}, fmt.Errorf("client %q cannot be combined in a standard Skill install", agent)
			}
		}
		return Client{ID: strings.Join(agents, ","), Name: strings.Join(agents, ", "), SkillAgents: agents, SupportsSkill: true}, nil
	}
	normalized := normalizeAgentID(id)
	if normalized == "" {
		normalized = "auto"
	}
	if mode == ModeMCP && (normalized == "auto" || normalized == "all") {
		clients := MCPClients(homeDir)
		if normalized == "auto" {
			clients = DetectMCPClients(homeDir, lookPath)
			if len(clients) == 0 {
				return Client{}, fmt.Errorf("no supported MCP client was detected; choose one with --client")
			}
		}
		return combineMCPClients(normalized, clients), nil
	}
	if known, ok := FindMCPClient(homeDir, normalized); ok {
		if !known.Supports(mode) {
			return Client{}, fmt.Errorf("client %q does not support mode %q", known.ID, mode)
		}
		return known, nil
	}
	if mode == ModeSkill && normalized == "promptscript" {
		return Client{}, fmt.Errorf("PromptScript is project-only and does not support global Skill installation; run skills add from the target project without --global")
	}
	if mode == ModeMCP {
		return Client{}, fmt.Errorf("MCP auto-configuration is not verified for client %q", normalized)
	}
	if normalized == "claude-desktop" {
		return Client{}, fmt.Errorf("Claude Desktop does not load Agent Skills; choose MCP mode")
	}
	skillAgents := []string{normalized}
	name := normalized
	if normalized == "auto" {
		skillAgents = detectSkillAgents(homeDir, lookPath)
		if len(skillAgents) == 0 {
			return Client{}, fmt.Errorf("no supported Skill client was detected; choose one with --agent")
		}
		name = "Detected AI clients: " + strings.Join(skillAgents, ", ")
	} else if normalized == "all" {
		skillAgents = globalSkillAgents(homeDir)
		name = "All verified clients with global Skill support"
	}
	return Client{ID: normalized, Name: name, SkillAgents: skillAgents, SupportsSkill: true}, nil
}

func detectSkillAgents(homeDir string, lookPath func(string) (string, error)) []string {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	seen := map[string]bool{}
	result := []string{}
	for _, client := range MCPClients(homeDir) {
		if !client.SupportsSkill || len(client.SkillAgents) == 0 || !skillClientDetected(client, lookPath) {
			continue
		}
		for _, agent := range client.SkillAgents {
			if agent == "promptscript" || seen[agent] {
				continue
			}
			seen[agent] = true
			result = append(result, agent)
		}
	}
	return result
}

func DetectSkillClients(homeDir string) []Client {
	agents := detectSkillAgents(homeDir, exec.LookPath)
	clients := make([]Client, 0, len(agents))
	for _, agent := range agents {
		name := agent
		for _, client := range MCPClients(homeDir) {
			if slices.Contains(client.SkillAgents, agent) {
				name = client.Name
				break
			}
		}
		clients = append(clients, Client{ID: agent, Name: name, SkillAgents: []string{agent}, SupportsSkill: true})
	}
	return clients
}

func globalSkillAgents(homeDir string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, client := range MCPClients(homeDir) {
		if !client.SupportsSkill {
			continue
		}
		for _, agent := range client.SkillAgents {
			if agent == "promptscript" || seen[agent] {
				continue
			}
			seen[agent] = true
			result = append(result, agent)
		}
	}
	return result
}

func skillClientDetected(client Client, lookPath func(string) (string, error)) bool {
	// Cline and Roo are VS Code extensions. The generic `code` executable does
	// not prove either extension is installed, so rely on their own data paths.
	if client.ID != "cline" && client.ID != "roo" {
		for _, command := range clientDetectionCommands(client.ID) {
			if _, err := lookPath(command); err == nil {
				return true
			}
		}
	}
	for _, path := range clientDetectionPaths(client) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func normalizeAgentIDs(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeAgentID(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func normalizeAgentID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "claude", "claude-cli":
		return "claude-code"
	case "hermes":
		return "hermes-agent"
	case "workbuddy", "work-buddy":
		return "codebuddy"
	case "github-copilot", "copilot", "vs-code":
		return "vscode"
	case "roo-code":
		return "roo"
	case "gemini":
		return "gemini-cli"
	case "qwen":
		return "qwen-code"
	case "kiro":
		return "kiro-cli"
	case "kilo", "kilocode":
		return "kilo-code"
	case "droid", "factory", "factory-ai":
		return "factory-droid"
	case "*":
		return "all"
	default:
		return normalized
	}
}

func resolveMCPClients(homeDir string, ids []string) (Client, error) {
	targets := make([]Client, 0, len(ids))
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		client, ok := FindMCPClient(homeDir, id)
		if !ok || !client.SupportsMCP {
			return Client{}, fmt.Errorf("MCP auto-configuration is not verified for client %q", id)
		}
		targets = append(targets, client)
		names = append(names, client.Name)
	}
	return Client{ID: strings.Join(ids, ","), Name: strings.Join(names, ", "), MCPTargets: targets, SupportsMCP: true}, nil
}

func combineMCPClients(id string, clients []Client) Client {
	names := make([]string, 0, len(clients))
	for _, client := range clients {
		names = append(names, client.Name)
	}
	return Client{ID: id, Name: strings.Join(names, ", "), MCPTargets: append([]Client(nil), clients...), SupportsMCP: true}
}

func DetectMCPClients(homeDir string, lookPath func(string) (string, error)) []Client {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	result := []Client{}
	for _, client := range MCPClients(homeDir) {
		if clientDetected(client, lookPath) {
			result = append(result, client)
		}
	}
	return result
}

func clientDetected(client Client, lookPath func(string) (string, error)) bool {
	for _, command := range clientDetectionCommands(client.ID) {
		if _, err := lookPath(command); err == nil {
			return true
		}
	}
	for _, path := range clientDetectionPaths(client) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func clientDetectionCommands(id string) []string {
	switch id {
	case "claude-code":
		return []string{"claude"}
	case "codex":
		return []string{"codex"}
	case "openclaw":
		return []string{"openclaw"}
	case "hermes-agent":
		return []string{"hermes"}
	case "factory-droid":
		return []string{"droid"}
	case "cursor":
		return []string{"cursor"}
	case "vscode", "cline", "roo":
		return []string{"code"}
	case "gemini-cli":
		return []string{"gemini"}
	case "qwen-code":
		return []string{"qwen"}
	case "kiro-cli":
		return []string{"kiro-cli", "kiro"}
	case "kilo-code":
		return []string{"kilo"}
	case "zed":
		return []string{"zed"}
	case "amp":
		return []string{"amp"}
	case "windsurf":
		return []string{"windsurf"}
	case "opencode":
		return []string{"opencode"}
	case "qclaw":
		return []string{"qclaw"}
	case "codebuddy":
		return []string{"codebuddy"}
	default:
		return nil
	}
}

func clientDetectionPaths(client Client) []string {
	paths := []string{}
	if client.MCPConfigPath != "" {
		paths = append(paths, client.MCPConfigPath)
	}
	switch client.ID {
	case "claude-desktop", "cursor", "gemini-cli", "qwen-code", "kiro-cli", "kilo-code", "zed", "amp", "windsurf", "opencode", "qclaw", "codebuddy":
		if client.MCPConfigPath != "" {
			paths = append(paths, filepath.Dir(client.MCPConfigPath))
		}
	case "cline", "roo":
		if client.MCPConfigPath != "" {
			paths = append(paths, filepath.Dir(filepath.Dir(client.MCPConfigPath)))
		}
	}
	return paths
}

func MCPClientTargets(client Client) []Client {
	if len(client.MCPTargets) > 0 {
		return append([]Client(nil), client.MCPTargets...)
	}
	return []Client{client}
}

func RecommendedMode(client Client) Mode {
	if client.SupportsSkill {
		return ModeSkill
	}
	return ModeMCP
}
