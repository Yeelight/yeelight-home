package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseManifestSignsAndVerifiesCleanRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skill"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skill", "SKILL.md"), []byte("public skill package"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 7)
	}
	t.Setenv("YEELIGHT_RELEASE_SIGNING_KEY", base64.StdEncoding.EncodeToString(seed))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"release", "manifest", root}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var manifest map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &manifest); err != nil {
		t.Fatalf("invalid manifest json: %v", err)
	}
	if manifest["rootSha256"] == "" || manifest["version"] != "yeelight-smart-home-release-v1" {
		t.Fatalf("manifest = %#v", manifest)
	}
	manifestPath := filepath.Join(t.TempDir(), "release-manifest.json")
	if err := os.WriteFile(manifestPath, stdout.Bytes(), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"release", "verify-manifest", manifestPath}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("verify exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestReleaseManifestRequiresSigningKey(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("public skill package"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"release", "manifest", root}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "YEELIGHT_RELEASE_SIGNING_KEY") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestReleaseScanRejectsRawDocs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "internal.md"), []byte("internal"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"release", "scan", root}, strings.NewReader(""), &stdout, &stderr)
	if code != exitInvalidInput {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "raw docs") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestReleaseStageCopiesAllowlistedFiles(t *testing.T) {
	root := t.TempDir()
	writeCommandReleaseFixture(t, root)
	output := filepath.Join(t.TempDir(), "stage")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	code := run([]string{"release", "stage", "docs/development-governance/release-allowlist.yaml", output}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid stage json: %v", err)
	}
	if response["ok"] != true || response["filesCopied"] != float64(1) {
		t.Fatalf("response = %#v", response)
	}
	if _, err := os.Stat(filepath.Join(output, "docs", "internal.md")); !os.IsNotExist(err) {
		t.Fatalf("raw docs should not be staged")
	}
	if _, err := os.Stat(filepath.Join(output, "specifications", "release-allowlist.yaml")); !os.IsNotExist(err) {
		t.Fatalf("development specifications should not be staged")
	}
}

func TestReleaseBuildCreatesCurrentPlatformBinary(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := filepath.Clean("../..")

	code := run([]string{"release", "build", root}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("invalid build json: %v", err)
	}
	output, _ := response["output"].(string)
	if response["ok"] != true || !strings.Contains(output, "bin/yeelight-home-"+runtime.GOOS+"-"+runtime.GOARCH) {
		t.Fatalf("response = %#v", response)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("runtime binary was not created: %v", err)
	}
}

func writeCommandReleaseFixture(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "skill", "yeelight-smart-home"), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "runtime", "bin"), 0o755); err != nil {
		t.Fatalf("mkdir runtime bin: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "development-governance"), 0o755); err != nil {
		t.Fatalf("mkdir governance: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "specifications"), 0o755); err != nil {
		t.Fatalf("mkdir specifications: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skill", "yeelight-smart-home", "SKILL.md"), []byte("public skill package"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime", "bin", "yeelight-home-test"), []byte("binary"), 0o755); err != nil {
		t.Fatalf("write runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "internal.md"), []byte("internal"), 0o644); err != nil {
		t.Fatalf("write docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "development-governance", "release-allowlist.yaml"), []byte(`include:
  - skill/yeelight-smart-home/SKILL.md
exclude:
  - docs/**
  - specifications/**
  - runtime/bin/**
  - runtime/**/*.go
  - tools/**
`), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}
}
