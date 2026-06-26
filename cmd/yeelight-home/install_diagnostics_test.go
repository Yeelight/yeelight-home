package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestInstallDiagnosticsReportsPackageManagerVersionSkew(t *testing.T) {
	tempDir := t.TempDir()
	executable := tempDir + "/npm-cache/v0.1.6/darwin-arm64/yeelight-home"
	pathLookup := tempDir + "/bin/yeelight-home"
	npmWrapper := tempDir + "/node_modules/yeelight-home/bin/yeelight-home.js"
	runner := func(command string, args ...string) (string, error) {
		key := command + " " + strings.Join(args, " ")
		switch key {
		case "npm root -g":
			return "/opt/homebrew/lib/node_modules", nil
		case "npm list -g yeelight-home --depth=0 --json":
			return `{"dependencies":{"yeelight-home":{"version":"0.1.4"}}}`, nil
		case "brew --prefix":
			return "/opt/homebrew", nil
		case "brew list --versions yeelight-home":
			return "yeelight-home 0.1.6", nil
		case "brew list --cask --versions yeelight-home":
			return "", os.ErrNotExist
		default:
			t.Fatalf("unexpected command: %s", key)
			return "", nil
		}
	}
	diagnostics := buildInstallDiagnostics(
		executable,
		pathLookup,
		"0.1.6",
		"darwin",
		"arm64",
		npmWrapper,
		runner,
		nil,
	)
	packageManagers := diagnostics["packageManagers"].(map[string]any)
	npm := packageManagers["npm"].(map[string]any)
	homebrew := packageManagers["homebrew"].(map[string]any)
	if npm["version"] != "0.1.4" || homebrew["version"] != "0.1.6" {
		t.Fatalf("packageManagers = %#v", packageManagers)
	}
	if homebrew["channel"] != "formula" || homebrew["formula"].(map[string]any)["installed"] != true {
		t.Fatalf("homebrew = %#v", homebrew)
	}
	warnings := diagnostics["warnings"].([]string)
	if !containsString(warnings, "npm_global_package_version_differs_from_runtime_version") {
		t.Fatalf("warnings = %#v", warnings)
	}
	if !containsString(warnings, "npm_wrapper_differs_from_path_lookup") {
		t.Fatalf("warnings = %#v", warnings)
	}
	if diagnostics["npmWrapper"] == "" || diagnostics["npmWrapperResolved"] == "" {
		t.Fatalf("npm wrapper diagnostics missing: %#v", diagnostics)
	}
	remediations := diagnostics["remediations"].([]string)
	if len(remediations) == 0 || !strings.Contains(strings.Join(remediations, "\n"), "npm install -g yeelight-home@latest") {
		t.Fatalf("remediations = %#v", remediations)
	}
}

func TestInstallDiagnosticsReportsUnavailablePackageManagers(t *testing.T) {
	runner := func(command string, args ...string) (string, error) {
		return "", os.ErrNotExist
	}
	diagnostics := buildInstallDiagnostics("/usr/local/bin/yeelight-home", "/usr/local/bin/yeelight-home", "0.1.6", "linux", "amd64", "", runner, nil)
	packageManagers := diagnostics["packageManagers"].(map[string]any)
	npm := packageManagers["npm"].(map[string]any)
	homebrew := packageManagers["homebrew"].(map[string]any)
	if npm["available"] != false || homebrew["available"] != false {
		t.Fatalf("packageManagers = %#v", packageManagers)
	}
}

func TestInstallDiagnosticsOnlyIncludesLatestWhenCheckerProvided(t *testing.T) {
	runner := func(command string, args ...string) (string, error) {
		switch command + " " + strings.Join(args, " ") {
		case "npm root -g":
			return "/opt/homebrew/lib/node_modules", nil
		case "npm list -g yeelight-home --depth=0 --json":
			return `{"dependencies":{"yeelight-home":{"version":"0.1.6"}}}`, nil
		case "brew --prefix":
			return "/opt/homebrew", nil
		case "brew list --versions yeelight-home", "brew list --cask --versions yeelight-home":
			return "", os.ErrNotExist
		default:
			return "", os.ErrNotExist
		}
	}
	offline := buildInstallDiagnostics("/tmp/yeelight-home", "/tmp/yeelight-home", "0.1.6", "darwin", "arm64", "", runner, nil)
	if _, ok := offline["latest"]; ok {
		t.Fatalf("offline diagnostics unexpectedly included latest: %#v", offline["latest"])
	}
	online := buildInstallDiagnostics("/tmp/yeelight-home", "/tmp/yeelight-home", "0.1.6", "darwin", "arm64", "", runner, func(_ context.Context) map[string]any {
		return map[string]any{"checked": true, "channels": map[string]any{"npm": map[string]any{"ok": true, "version": "0.1.6"}}}
	})
	if _, ok := online["latest"]; !ok {
		t.Fatalf("online diagnostics missing latest")
	}
}

func TestParsePackageManagerVersions(t *testing.T) {
	if got := parseNPMGlobalVersion(`{"dependencies":{"yeelight-home":{"version":"0.1.7"}}}`); got != "0.1.7" {
		t.Fatalf("parseNPMGlobalVersion = %q", got)
	}
	if got := parseHomebrewVersion("yeelight-home 0.1.7"); got != "0.1.7" {
		t.Fatalf("parseHomebrewVersion = %q", got)
	}
	if !versionMismatch("v0.1.7", "0.1.6") || versionMismatch("0.1.7", "v0.1.7") || versionMismatch("dev", "0.1.7") {
		t.Fatal("versionMismatch returned unexpected result")
	}
	if !versionNewerThan("0.1.7", "0.1.6") || versionNewerThan("0.1.6", "0.1.7") || versionNewerThan("dev", "0.1.7") {
		t.Fatal("versionNewerThan returned unexpected result")
	}
}

func TestInstallDiagnosticsReportsOnlineLatestSkew(t *testing.T) {
	runner := func(command string, args ...string) (string, error) {
		key := command + " " + strings.Join(args, " ")
		switch key {
		case "npm root -g":
			return "/opt/homebrew/lib/node_modules", nil
		case "npm list -g yeelight-home --depth=0 --json":
			return `{"dependencies":{"yeelight-home":{"version":"0.1.4"}}}`, nil
		case "brew --prefix":
			return "/opt/homebrew", nil
		case "brew list --versions yeelight-home":
			return "yeelight-home 0.1.5", nil
		case "brew list --cask --versions yeelight-home":
			return "yeelight-home 0.1.4", nil
		default:
			t.Fatalf("unexpected command: %s", key)
			return "", nil
		}
	}
	latest := func(_ context.Context) map[string]any {
		return map[string]any{
			"checked": true,
			"channels": map[string]any{
				"npm":          map[string]any{"ok": true, "version": "0.1.6"},
				"homebrew":     map[string]any{"ok": true, "version": "0.1.6"},
				"homebrewCask": map[string]any{"ok": true, "version": "0.1.6"},
			},
		}
	}
	diagnostics := buildInstallDiagnostics("/tmp/yeelight-home", "/tmp/yeelight-home", "0.1.6", "darwin", "arm64", "", runner, latest)
	warnings := diagnostics["warnings"].([]string)
	for _, expected := range []string{"npm_global_package_behind_latest", "homebrew_formula_behind_latest", "homebrew_cask_behind_latest"} {
		if !containsString(warnings, expected) {
			t.Fatalf("warnings missing %s: %#v", expected, warnings)
		}
	}
	if !containsString(warnings, "homebrew_formula_version_differs_from_runtime_version") || !containsString(warnings, "homebrew_cask_version_differs_from_runtime_version") {
		t.Fatalf("warnings = %#v", warnings)
	}
	remediations := strings.Join(diagnostics["remediations"].([]string), "\n")
	if !strings.Contains(remediations, "npm install -g yeelight-home@latest") || !strings.Contains(remediations, "brew upgrade yeelight-home") || !strings.Contains(remediations, "brew upgrade --cask yeelight-home") {
		t.Fatalf("remediations = %s", remediations)
	}
}

func TestParseHomebrewFormulaVersion(t *testing.T) {
	body := `class YeelightHome < Formula
  version "0.1.6"
end`
	if got := parseHomebrewFormulaVersion(body); got != "0.1.6" {
		t.Fatalf("parseHomebrewFormulaVersion = %q", got)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
