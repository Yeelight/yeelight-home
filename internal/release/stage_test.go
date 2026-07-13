package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageCopiesAllowlistedFilesAndScansOutput(t *testing.T) {
	root := t.TempDir()
	writeReleaseFixture(t, root)
	output := filepath.Join(t.TempDir(), "stage")
	result, err := Stage(root, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), output)
	if err != nil {
		t.Fatalf("Stage error: %v", err)
	}
	if !result.OK || result.FilesCopied != 11 {
		t.Fatalf("result = %#v", result)
	}
	for _, forbidden := range []string{"docs/internal.md", "specifications/api-registry.yaml", "runtime/bin/yeelight-home-test", "runtime/cmd/main.go", "scripts/install.sh", "scripts/install.ps1", "tools/build.sh"} {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(forbidden))); !os.IsNotExist(err) {
			t.Fatalf("forbidden file was staged: %s", forbidden)
		}
	}
}

func TestStageRejectsMissingIncludeAndExcludedInclude(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), `include:
  - missing/file.txt
exclude:
  - docs/**
`)
	if _, err := Stage(root, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), filepath.Join(t.TempDir(), "stage")); err == nil {
		t.Fatal("expected missing include to fail")
	}
	writeFile(t, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), `include:
  - docs/internal.md
exclude:
  - docs/**
`)
	writeFile(t, filepath.Join(root, "docs", "internal.md"), "internal")
	if _, err := Stage(root, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), filepath.Join(t.TempDir(), "stage")); err == nil {
		t.Fatal("expected excluded include to fail")
	}
}

func writeReleaseFixture(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "SKILL.md"), "public skill")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "agents", "openai.yaml"), "interface:\n  display_name: Yeelight Smart Home\n")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "assets", "intent-catalog.json"), "{}")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "assets", "catalog", "yeelight-domain.json"), "{}")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "assets", "schemas", "request.json"), "{}")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "references", "device-control.md"), "device control")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "references", "thing-model.md"), "thing model")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "scripts", "invoke"), "invoke wrapper")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "scripts", "invoke.sh"), "invoke")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "scripts", "invoke.ps1"), "invoke")
	writeFile(t, filepath.Join(root, "skill", "yeelight-smart-home", "scripts", "runtime-manifest.json"), "{}")
	writeFile(t, filepath.Join(root, "scripts", "install.sh"), "install")
	writeFile(t, filepath.Join(root, "scripts", "install.ps1"), "install")
	writeFile(t, filepath.Join(root, "runtime", "bin", "yeelight-home-test"), "binary")
	writeFile(t, filepath.Join(root, "specifications", "api-registry.yaml"), "version: test")
	writeFile(t, filepath.Join(root, "specifications", "runtime-capability-coverage.json"), "{}")
	writeFile(t, filepath.Join(root, "specifications", "environment-error-matrix.md"), "environment matrix")
	writeFile(t, filepath.Join(root, "specifications", "blocked-capability-ledger.md"), "blocked ledger")
	writeFile(t, filepath.Join(root, "specifications", "risk-policy.yaml"), "version: test")
	writeFile(t, filepath.Join(root, "docs", "internal.md"), "internal")
	writeFile(t, filepath.Join(root, "runtime", "cmd", "main.go"), "package main")
	writeFile(t, filepath.Join(root, "tools", "build.sh"), "echo build")
	writeFile(t, filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), `include:
  - skill/yeelight-smart-home/SKILL.md
  - skill/yeelight-smart-home/agents/openai.yaml
  - skill/yeelight-smart-home/assets/intent-catalog.json
  - skill/yeelight-smart-home/assets/catalog/*.json
  - skill/yeelight-smart-home/assets/schemas/*.json
  - skill/yeelight-smart-home/references/*.md
  - skill/yeelight-smart-home/scripts/invoke
  - skill/yeelight-smart-home/scripts/invoke.sh
  - skill/yeelight-smart-home/scripts/invoke.ps1
  - skill/yeelight-smart-home/scripts/runtime-manifest.json
exclude:
  - docs/**
  - specifications/**
  - runtime/bin/**
  - runtime/**/*.go
  - tools/**
`)
}
