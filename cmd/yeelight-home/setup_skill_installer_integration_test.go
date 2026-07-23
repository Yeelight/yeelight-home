package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

const publicSkillRepository = "https://github.com/Yeelight/yeelight-smart-home-skills"

func TestSkillInstallerExternalAllVerifiedAgents(t *testing.T) {
	requireExternalSkillInstallerSmoke(t)
	app, options := isolatedSkillInstallerApp(t)
	agents := []string{
		"claude-code", "codex", "hermes-agent", "github-copilot", "gemini-cli",
		"qwen-code", "kilo", "zed", "amp", "windsurf", "opencode",
	}
	step := externalSkillStep(agents...)
	report, err := app.runSkillInstallerWithReport(step, options)
	if err != nil {
		t.Fatalf("runSkillInstallerWithReport error: %v", err)
	}
	if len(report.Installed) != len(agents) || len(report.Failed) != 0 {
		t.Fatalf("report = %#v", report)
	}
	if _, err := os.Stat(filepath.Join(options.HomeDir, ".agents", "skills", "yeelight-smart-home")); err != nil {
		t.Fatalf("installed Skill missing: %v", err)
	}
}

func TestSkillInstallerExternalKeepsValidAgentWhenAnotherIsInvalid(t *testing.T) {
	requireExternalSkillInstallerSmoke(t)
	app, options := isolatedSkillInstallerApp(t)
	report, err := app.runSkillInstallerWithReport(externalSkillStep("codex", "future-invalid"), options)
	if err != nil {
		t.Fatalf("partial install should keep the valid Agent: %v", err)
	}
	if len(report.Installed) != 1 || report.Installed[0] != "codex" || len(report.Failed) != 1 || report.Failed[0] != "future-invalid" {
		t.Fatalf("report = %#v", report)
	}
}

func TestSkillInstallerExternalRejectsPromptScriptExitZeroFailure(t *testing.T) {
	requireExternalSkillInstallerSmoke(t)
	app, options := isolatedSkillInstallerApp(t)
	report, err := app.runSkillInstallerWithReport(externalSkillStep("promptscript"), options)
	if err == nil || len(report.Failed) != 1 || report.Failed[0] != "promptscript" {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}

func requireExternalSkillInstallerSmoke(t *testing.T) {
	t.Helper()
	if os.Getenv("YEELIGHT_RUN_SKILL_INSTALLER_SMOKE") != "1" {
		t.Skip("set YEELIGHT_RUN_SKILL_INSTALLER_SMOKE=1 to run the real npx skills installer")
	}
}

func isolatedSkillInstallerApp(t *testing.T) (*app, setupExecutionOptions) {
	t.Helper()
	homeDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("npm_config_cache", cacheDir)
	return newTestApp(t), setupExecutionOptions{
		Locale: "en-US", HomeDir: homeDir, Stdout: io.Discard, Stderr: io.Discard,
	}
}

func externalSkillStep(agents ...string) setupdomain.Step {
	command := []string{
		"npx", "-y", "skills", "add", publicSkillRepository,
		"--skill", "yeelight-smart-home", "--global", "--yes", "--agent",
	}
	command = append(command, agents...)
	return setupdomain.Step{Method: setupdomain.MethodSkillsCLI, Command: command, Sources: []string{publicSkillRepository}}
}
