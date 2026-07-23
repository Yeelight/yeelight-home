package setup

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

func TestBuildPlanUsesSkillsCLIForCodex(t *testing.T) {
	plan, err := BuildPlan(Options{Locale: "zh-CN", ClientID: "codex", Mode: ModeSkill, HomeDir: "/tmp/home"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if len(plan.Steps) != 4 || plan.Steps[2].Method != MethodSkillsCLI {
		t.Fatalf("steps = %#v", plan.Steps)
	}
	if plan.Steps[1].Title != "检查 Yeelight Pro 登录账号；可继续使用或扫码切换" {
		t.Fatalf("login step should describe account reuse and switching: %#v", plan.Steps[1])
	}
	if plan.Steps[2].Command[len(plan.Steps[2].Command)-1] != "codex" {
		t.Fatalf("skill command = %#v", plan.Steps[2].Command)
	}
}

func TestBuildPlanUsesUpstreamSkillsCLIIdentifierForKiloCode(t *testing.T) {
	plan, err := BuildPlan(Options{Locale: "zh-CN", ClientID: "kilo-code", Mode: ModeSkill, HomeDir: "/tmp/home"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	command := plan.Steps[2].Command
	if command[len(command)-1] != "kilo" || slices.Contains(command, "kilo-code") {
		t.Fatalf("skill command = %#v", command)
	}
}

func TestBuildPlanUsesFirstPartySkillPathForQClaw(t *testing.T) {
	plan, err := BuildPlan(Options{Locale: "zh-CN", ClientID: "qclaw", Mode: ModeSkill, HomeDir: "/tmp/home"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	step := plan.Steps[2]
	if step.Method != MethodDirectSkill || step.Destination != filepath.Join("/tmp/home", ".qclaw", "skills", "yeelight-smart-home") {
		t.Fatalf("step = %#v", step)
	}
}

func TestBuildPlanRequiresGatewayForLAN(t *testing.T) {
	_, err := BuildPlan(Options{Locale: "en-US", ClientID: "openclaw", Mode: ModeLAN, HomeDir: "/tmp/home"})
	if err == nil || err.Error() != "gateway IP is required for LAN mode" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildPlanLocalOnlySkipsCloudLogin(t *testing.T) {
	plan, err := BuildPlan(Options{
		Locale: "en-US", ClientID: "codex", Mode: ModeLAN, GatewayIP: "192.168.1.2",
		ControlMode: "local-only", HomeDir: "/tmp/home",
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if plan.ControlMode != ControlModeLocalOnly {
		t.Fatalf("control mode = %q", plan.ControlMode)
	}
	for _, step := range plan.Steps {
		if step.ID == "login" || step.Method == MethodAuthQR {
			t.Fatalf("local-only plan includes cloud login: %#v", plan.Steps)
		}
	}
}

func TestClaudeDesktopRecommendsMCP(t *testing.T) {
	client, ok := FindMCPClient("/tmp/home", "claude-desktop")
	if !ok || RecommendedMode(client) != ModeMCP {
		t.Fatalf("client = %#v", client)
	}
}

func TestBuildPlanDelegatesUnknownAgentToSkillsCLI(t *testing.T) {
	plan, err := BuildPlan(Options{Locale: "en-US", ClientID: "future-agent", Mode: ModeSkill, HomeDir: "/tmp/home"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	step := plan.Steps[2]
	if step.Method != MethodSkillsCLI || step.Command[len(step.Command)-1] != "future-agent" || len(step.Sources) != 3 {
		t.Fatalf("step = %#v", step)
	}
}

func TestBuildPlanAutoDetectsAndPinsInstalledSkillAgents(t *testing.T) {
	plan, err := BuildPlan(Options{
		Locale: "zh-CN", ClientID: "auto", Mode: ModeSkill, HomeDir: "/tmp/home",
		LookPath: func(command string) (string, error) {
			if command == "codex" {
				return "/usr/local/bin/codex", nil
			}
			return "", fmt.Errorf("not found")
		},
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	command := plan.Steps[2].Command
	wantSuffix := []string{"--agent", "codex"}
	if len(command) < len(wantSuffix) || !slices.Equal(command[len(command)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("auto command should pin detected agents: %#v", command)
	}
}

func TestBuildPlanDelegatesInteractiveSkillSelectionToVercel(t *testing.T) {
	plan, err := BuildPlan(Options{
		Locale: "zh-CN", ClientID: "auto", Mode: ModeSkill, HomeDir: "/tmp/home",
		InteractiveSkillsInstaller: true,
	})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	command := plan.Steps[2].Command
	if plan.Client.Name != "Vercel Skills installer" || slices.Contains(command, "--agent") || slices.Contains(command, "--yes") {
		t.Fatalf("interactive Skill command must delegate selection to Vercel: client=%#v command=%#v", plan.Client, command)
	}
	if len(command) < 3 || command[2] != "skills@1.5.20" {
		t.Fatalf("unexpected Vercel Skills package: %#v", command)
	}
}

func TestBuildPlanDefaultsMCPToLocalStdio(t *testing.T) {
	plan, err := BuildPlan(Options{Locale: "zh-CN", ClientID: "cursor", Mode: ModeMCP, HomeDir: "/tmp/home"})
	if err != nil {
		t.Fatalf("BuildPlan error: %v", err)
	}
	if plan.MCPSource != MCPSourceLocal || plan.Steps[2].Method != MethodNativeMCP || plan.Steps[2].Title != "连接本机 Yeelight Home Runtime" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestBuildPlanAcceptsExplicitCloudAndGatewayMCP(t *testing.T) {
	cloud, err := BuildPlan(Options{Locale: "en-US", ClientID: "codex", Mode: ModeMCP, MCPSource: "cloud", HomeDir: "/tmp/home"})
	if err != nil || cloud.MCPSource != MCPSourceCloud || cloud.Steps[2].Title != "Connect the lightweight Yeelight cloud services" {
		t.Fatalf("cloud plan = %#v, err = %v", cloud, err)
	}
	gateway, err := BuildPlan(Options{Locale: "en-US", ClientID: "codex", Mode: ModeLAN, MCPSource: "gateway", GatewayIP: "192.168.1.2", HomeDir: "/tmp/home"})
	if err != nil || gateway.MCPSource != MCPSourceGateway || gateway.Steps[3].Method != MethodNativeMCP || gateway.Steps[3].Title != "Connect the AI client directly to the home gateway" {
		t.Fatalf("gateway plan = %#v, err = %v", gateway, err)
	}
}

func TestParseMCPSourceRejectsInvalidModeCombination(t *testing.T) {
	if _, err := ParseMCPSource("cloud", ModeLAN); err == nil {
		t.Fatal("expected cloud source with LAN mode to fail")
	}
	if _, err := ParseMCPSource("gateway", ModeMCP); err == nil {
		t.Fatal("expected gateway source with MCP mode to fail")
	}
}
