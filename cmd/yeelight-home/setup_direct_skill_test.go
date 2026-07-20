package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func TestInstallDirectSkillUsesDownloadedPackage(t *testing.T) {
	app := newTestApp(t)
	destination := filepath.Join(t.TempDir(), "skills", "yeelight-smart-home")
	var sources []string
	app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
		sources = append(sources, command[4])
		clonePath := command[len(command)-1]
		skillPath := filepath.Join(clonePath, "skills", "yeelight-smart-home")
		if err := os.MkdirAll(skillPath, 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("installed"), 0o600)
	}
	step := setupdomain.Step{
		Method: setupdomain.MethodDirectSkill, Destination: destination,
		Sources: []string{"https://example.com/skills.git"},
	}
	if err := app.installDirectSkill(step, setupExecutionOptions{Stderr: io.Discard}); err != nil {
		t.Fatalf("installDirectSkill error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil || string(data) != "installed" || len(sources) != 1 || sources[0] != step.Sources[0] {
		t.Fatalf("installed data = %q, sources = %#v, err = %v", data, sources, err)
	}
}

func TestReplaceDirectoryAtomicallyReplacesExistingSkill(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "skills", "yeelight-smart-home")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatalf("MkdirAll source error: %v", err)
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatalf("MkdirAll destination error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile source error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "SKILL.md"), []byte("old"), 0o600); err != nil {
		t.Fatalf("WriteFile destination error: %v", err)
	}
	if err := replaceDirectory(source, destination); err != nil {
		t.Fatalf("replaceDirectory error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil || string(data) != "new" {
		t.Fatalf("installed SKILL.md = %q, err = %v", data, err)
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(destination), ".yeelight-smart-home.backup-*"))
	if err != nil || len(backups) != 0 {
		t.Fatalf("backup paths = %#v, err = %v", backups, err)
	}
}

func TestCopyDirectoryRejectsSkillSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows hosts")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatalf("MkdirAll source error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if err := os.Symlink("SKILL.md", filepath.Join(source, "linked.md")); err != nil {
		t.Fatalf("Symlink error: %v", err)
	}
	err := copyDirectory(source, filepath.Join(root, "destination"))
	if err == nil || !strings.Contains(err.Error(), "unsupported symlink") {
		t.Fatalf("copyDirectory error = %v", err)
	}
}
