package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type skillInstallFailureKind string

const (
	skillInstallFailureAgent  skillInstallFailureKind = "agent"
	skillInstallFailureSource skillInstallFailureKind = "source"
)

type skillInstallError struct {
	kind  skillInstallFailureKind
	cause error
}

func (err *skillInstallError) Error() string { return err.cause.Error() }
func (err *skillInstallError) Unwrap() error { return err.cause }

type skillInstallReport struct {
	Installed []string
	Failed    []string
}

func (app *app) runSkillInstaller(step setupdomain.Step, options setupExecutionOptions) error {
	_, err := app.runSkillInstallerWithReport(step, options)
	return err
}

func (app *app) runSkillInstallerWithReport(step setupdomain.Step, options setupExecutionOptions) (skillInstallReport, error) {
	command := append([]string(nil), step.Command...)
	if len(command) == 0 {
		return skillInstallReport{}, fmt.Errorf("skill installer command is empty")
	}
	agents := skillInstallerAgents(command)
	if options.InteractiveSkillsInstaller && len(agents) == 0 {
		return app.runInteractiveSkillsInstaller(command, step.Sources, options)
	}
	if !slices.Contains(command, "--yes") {
		command = append(command, "--yes")
	}
	if err := app.runSkillInstallerAcrossSources(command, step.Sources, options); err == nil {
		return skillInstallReport{Installed: agents}, nil
	} else if !isSkillAgentFailure(err) || len(agents) < 2 {
		return skillInstallReport{Failed: agents}, err
	}

	report := skillInstallReport{}
	for _, agent := range agents {
		candidate := skillInstallerCommandForAgent(command, agent)
		if err := app.runSkillInstallerAcrossSources(candidate, step.Sources, options); err != nil {
			report.Failed = append(report.Failed, agent)
			continue
		}
		report.Installed = append(report.Installed, agent)
	}
	if len(report.Installed) == 0 {
		return report, fmt.Errorf("Skill installation failed for every selected Agent: %s", strings.Join(report.Failed, ", "))
	}
	_, _ = fmt.Fprintln(skillInstallerDiagnosticsWriter(options), formatSkillInstallSummary(options.Locale, report))
	return report, nil
}

func (app *app) runSkillInstallerAcrossSources(command []string, sources []string, options setupExecutionOptions) error {
	output := options.Stdout
	if options.Quiet {
		output = options.Stderr
	}
	if len(sources) == 0 {
		return app.runSkillInstallerCommand(command, output, options.Stderr)
	}
	var lastErr error
	for _, source := range sources {
		candidate := append([]string(nil), command...)
		if index := slices.Index(candidate, sources[0]); index >= 0 {
			candidate[index] = source
		}
		err := app.runSkillInstallerCommand(candidate, output, options.Stderr)
		if err == nil {
			return nil
		}
		if isSkillAgentFailure(err) {
			return err
		}
		lastErr = err
	}
	return lastErr
}

func (app *app) runSkillInstallerCommand(command []string, stdout io.Writer, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	var diagnostics bytes.Buffer
	processErr := app.runSetupProcess(
		context.Background(), command,
		io.MultiWriter(stdout, &diagnostics),
		io.MultiWriter(stderr, &diagnostics),
	)
	diagnosticText := strings.ToLower(diagnostics.String())
	switch {
	case strings.Contains(diagnosticText, "invalid agents:"):
		return &skillInstallError{kind: skillInstallFailureAgent, cause: fmt.Errorf("Skill installer rejected one or more Agent ids")}
	case strings.Contains(diagnosticText, "does not support global skill installation"), strings.Contains(diagnosticText, "failed to install"):
		return &skillInstallError{kind: skillInstallFailureAgent, cause: fmt.Errorf("partial Skill installation failure: at least one selected Agent does not support global installation")}
	case strings.Contains(diagnosticText, "no skills found"):
		return &skillInstallError{kind: skillInstallFailureSource, cause: fmt.Errorf("Skill source did not expose any installable skills")}
	case processErr != nil:
		return processErr
	default:
		return nil
	}
}

func isSkillAgentFailure(err error) bool {
	typed, ok := err.(*skillInstallError)
	return ok && typed.kind == skillInstallFailureAgent
}

func skillInstallerAgents(command []string) []string {
	index := slices.Index(command, "--agent")
	if index < 0 {
		return nil
	}
	agents := []string{}
	for _, value := range command[index+1:] {
		if strings.HasPrefix(value, "-") {
			break
		}
		agents = append(agents, value)
	}
	return agents
}

func skillInstallerCommandForAgent(command []string, agent string) []string {
	index := slices.Index(command, "--agent")
	if index < 0 {
		return append(append([]string(nil), command...), "--agent", agent)
	}
	end := index + 1
	for end < len(command) && !strings.HasPrefix(command[end], "-") {
		end++
	}
	result := append([]string(nil), command[:index+1]...)
	result = append(result, agent)
	return append(result, command[end:]...)
}

func formatSkillInstallSummary(locale string, report skillInstallReport) string {
	if locale == "zh-CN" {
		return fmt.Sprintf("Skill 已安装到 %s；未安装：%s。可稍后用 --agent 单独重试。", strings.Join(report.Installed, ", "), strings.Join(report.Failed, ", "))
	}
	return fmt.Sprintf("Skill installed for %s; not installed for %s. Retry those clients later with --agent.", strings.Join(report.Installed, ", "), strings.Join(report.Failed, ", "))
}

func skillInstallerDiagnosticsWriter(options setupExecutionOptions) io.Writer {
	if options.Stderr != nil {
		return options.Stderr
	}
	return io.Discard
}
