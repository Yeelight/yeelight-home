package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRejectsRawDocsDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "internal.md"), "internal reference")

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if result.OK {
		t.Fatal("expected scan to reject raw docs")
	}
	if len(result.Violations) != 1 || result.Violations[0].Rule != "raw-docs" {
		t.Fatalf("violations = %#v", result.Violations)
	}
}

func TestScanRejectsForbiddenSensitiveText(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "skill", "README.md"), "Authorization: Bearer secret")

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if result.OK {
		t.Fatal("expected scan to reject forbidden text")
	}
	if len(result.Violations) != 2 {
		t.Fatalf("violations = %#v", result.Violations)
	}
}

func TestScanAcceptsCleanPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "skill", "README.md"), "public skill package")

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if !result.OK {
		t.Fatalf("expected clean package, violations = %#v", result.Violations)
	}
	if result.FilesScanned != 1 {
		t.Fatalf("FilesScanned = %d", result.FilesScanned)
	}
}

func TestScanSkipsRuntimeBinaryContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "runtime", "bin", "yeelight-home-test"), "binary with Authorization bytes")
	writeFile(t, filepath.Join(root, "skill", "SKILL.md"), "public skill")

	result, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if !result.OK {
		t.Fatalf("runtime binary content should be skipped: %#v", result.Violations)
	}
	if result.FilesScanned != 2 {
		t.Fatalf("FilesScanned = %d", result.FilesScanned)
	}
}

func TestScanAllowlistRequiresDomainCatalogAndRawDocsExclude(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "release-allowlist.yaml")
	writeFile(t, path, `include:
  - skill/yeelight-smart-home/SKILL.md
  - skill/yeelight-smart-home/assets/intent-catalog.json
  - skill/yeelight-smart-home/assets/schemas/*.json
  - skill/yeelight-smart-home/references/*.md
  - runtime/bin/yeelight-home-*
exclude:
  - runtime/**/*.go
  - tools/**
`)

	result, err := ScanAllowlist(path)
	if err != nil {
		t.Fatalf("ScanAllowlist error: %v", err)
	}
	if result.OK {
		t.Fatal("expected allowlist scan to fail")
	}
	if len(result.Violations) != 10 {
		t.Fatalf("violations = %#v", result.Violations)
	}
}

func TestScanAllowlistAcceptsReleasePolicy(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "release-allowlist.yaml")
	writeFile(t, path, `include:
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
releaseScans:
  forbiddenPatterns:
    - Bearer
    - accessToken
    - refreshToken
    - Authorization
    - client_secret
    - internal-api
    - confluence.yeedev
    - "192.168."
`)

	result, err := ScanAllowlist(path)
	if err != nil {
		t.Fatalf("ScanAllowlist error: %v", err)
	}
	if !result.OK {
		t.Fatalf("expected clean allowlist, violations = %#v", result.Violations)
	}
}

func TestScanAllowlistRejectsDevelopmentSpecificationsInclude(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "release-allowlist.yaml")
	writeFile(t, path, `include:
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
  - specifications/api-registry.yaml
exclude:
  - docs/**
  - specifications/**
  - runtime/bin/**
  - runtime/**/*.go
  - tools/**
`)

	result, err := ScanAllowlist(path)
	if err != nil {
		t.Fatalf("ScanAllowlist error: %v", err)
	}
	if result.OK {
		t.Fatal("expected allowlist scan to reject development specifications include")
	}
}

func TestScanAllowlistRejectsSensitiveTextOutsidePolicyPatterns(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "release-allowlist.yaml")
	writeFile(t, path, `include:
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
  - scripts/install.sh
  - scripts/install.ps1
  - runtime/bin/yeelight-home-*
exclude:
  - docs/**
  - specifications/**
  - runtime/**/*.go
  - tools/**
notes:
  example: Authorization must not be here
`)

	result, err := ScanAllowlist(path)
	if err != nil {
		t.Fatalf("ScanAllowlist error: %v", err)
	}
	if result.OK {
		t.Fatal("expected allowlist scan to reject sensitive note")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
