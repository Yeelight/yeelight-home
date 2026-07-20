package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

func (app *app) installDirectSkill(step setupdomain.Step, options setupExecutionOptions) error {
	if step.Destination == "" || len(step.Sources) == 0 {
		return fmt.Errorf("direct Skill destination or sources are missing")
	}
	tempRoot, err := os.MkdirTemp("", "yeelight-home-skill-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempRoot)
	clonePath := filepath.Join(tempRoot, "repository")
	var cloneErr error
	for _, source := range step.Sources {
		_ = os.RemoveAll(clonePath)
		cloneErr = app.runSetupProcess(context.Background(), []string{"git", "clone", "--depth", "1", source, clonePath}, io.Discard, options.Stderr)
		if cloneErr == nil {
			break
		}
	}
	if cloneErr != nil {
		return cloneErr
	}
	sourcePath := filepath.Join(clonePath, "skills", "yeelight-smart-home")
	if _, err := os.Stat(filepath.Join(sourcePath, "SKILL.md")); err != nil {
		return fmt.Errorf("downloaded repository does not contain yeelight-smart-home: %w", err)
	}
	return replaceDirectory(sourcePath, step.Destination)
}

func replaceDirectory(source string, destination string) error {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	staged := filepath.Join(parent, ".yeelight-smart-home.setup")
	backup := filepath.Join(parent, fmt.Sprintf(".yeelight-smart-home.backup-%d", time.Now().UnixNano()))
	_ = os.RemoveAll(staged)
	if err := copyDirectory(source, staged); err != nil {
		return err
	}
	hadExisting := false
	if _, err := os.Lstat(destination); err == nil {
		hadExisting = true
		if err := os.Rename(destination, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(staged, destination); err != nil {
		if hadExisting {
			_ = os.Rename(backup, destination)
		}
		return err
	}
	if hadExisting {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func copyDirectory(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("Skill package contains unsupported symlink %s", relative)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info, infoErr := entry.Info(); infoErr == nil && info.Mode()&0o111 != 0 {
			mode = 0o700
		}
		return os.WriteFile(target, data, mode)
	})
}
