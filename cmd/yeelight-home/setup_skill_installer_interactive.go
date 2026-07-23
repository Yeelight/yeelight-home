package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

var skillsInstallerAgentEnvironment = map[string]struct{}{
	"AI_AGENT":                   {},
	"ANTIGRAVITY_AGENT":          {},
	"AUGMENT_AGENT":              {},
	"CLAUDECODE":                 {},
	"CLAUDE_CODE":                {},
	"CLAUDE_CODE_IS_COWORK":      {},
	"CODEX_CI":                   {},
	"CODEX_SANDBOX":              {},
	"CODEX_THREAD_ID":            {},
	"COPILOT_ALLOW_ALL":          {},
	"COPILOT_GITHUB_TOKEN":       {},
	"COPILOT_MODEL":              {},
	"CURSOR_AGENT":               {},
	"CURSOR_EXTENSION_HOST_ROLE": {},
	"CURSOR_TRACE_ID":            {},
	"GEMINI_CLI":                 {},
	"OPENCODE_CLIENT":            {},
	"REPL_ID":                    {},
}

func (app *app) runInteractiveSkillsInstaller(command []string, sources []string, options setupExecutionOptions) (skillInstallReport, error) {
	candidate, err := app.resolveInteractiveSkillSource(command, sources)
	if err != nil {
		return skillInstallReport{}, err
	}
	processErr := app.runInteractiveSkillsInstallerCommand(candidate, options)
	agents, verifyErr := app.verifyInteractiveSkillsInstall(candidate, options)
	if verifyErr == nil {
		return skillInstallReport{Installed: agents}, nil
	}
	if processErr != nil {
		// skills@1.5.20 may exit non-zero after the user cancels its native
		// prompt. Verification is the reliable boundary: no installed Skill
		// means setup stopped without changing the result to a false success.
		return skillInstallReport{}, fmt.Errorf("%w: Vercel Skills installer stopped before installation completed", errSetupCancelled)
	}
	return skillInstallReport{}, fmt.Errorf("%w: %v", errSetupCancelled, verifyErr)
}

func (app *app) resolveInteractiveSkillSource(command []string, sources []string) ([]string, error) {
	if len(sources) == 0 {
		return append([]string(nil), command...), nil
	}
	var lastErr error
	for _, source := range sources {
		candidate := replaceSkillInstallerSource(command, sources[0], source)
		preflight := append(append([]string(nil), candidate...), "--list")
		if err := app.runSkillInstallerCommand(preflight, io.Discard, io.Discard); err != nil {
			lastErr = err
			continue
		}
		return candidate, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no Skill source is available")
	}
	return nil, lastErr
}

func replaceSkillInstallerSource(command []string, original string, source string) []string {
	candidate := append([]string(nil), command...)
	if source == "" || original == "" {
		return candidate
	}
	if index := slices.Index(candidate, original); index >= 0 {
		candidate[index] = source
	}
	return candidate
}

func (app *app) runInteractiveSkillsInstallerCommand(command []string, options setupExecutionOptions) error {
	stdin := options.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := options.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := options.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	if app.processInput != nil {
		return app.processInput(context.Background(), command, stdin, stdout, stderr)
	}
	if app.process != nil {
		return app.process(context.Background(), command, stdout, stderr)
	}
	if len(command) == 0 {
		return fmt.Errorf("command is empty")
	}
	process := exec.CommandContext(context.Background(), command[0], command[1:]...)
	process.Stdin = stdin
	process.Stdout = stdout
	process.Stderr = stderr
	process.Env = sanitizeSkillsInstallerEnvironment(os.Environ())
	return process.Run()
}

func sanitizeSkillsInstallerEnvironment(environment []string) []string {
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if found {
			if _, remove := skillsInstallerAgentEnvironment[name]; remove {
				continue
			}
		}
		result = append(result, entry)
	}
	return result
}

func (app *app) verifyInteractiveSkillsInstall(command []string, options setupExecutionOptions) ([]string, error) {
	if len(command) < 3 {
		return nil, fmt.Errorf("cannot derive Skills CLI package from installer command")
	}
	listCommand := []string{command[0], command[1], command[2], "list", "--global", "--json"}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := app.runSetupProcess(context.Background(), listCommand, &stdout, &stderr); err != nil {
		return nil, fmt.Errorf("verify Skill installation: %w", err)
	}
	var installed []struct {
		Name   string   `json:"name"`
		Agents []string `json:"agents"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &installed); err != nil {
		return nil, fmt.Errorf("parse Skills CLI verification: %w", err)
	}
	for _, skill := range installed {
		if skill.Name == "yeelight-smart-home" {
			return append([]string(nil), skill.Agents...), nil
		}
	}
	return nil, fmt.Errorf("Vercel Skills installer exited without installing yeelight-smart-home")
}
