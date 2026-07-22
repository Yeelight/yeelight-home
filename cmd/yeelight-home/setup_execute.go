package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/semantic"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type setupExecutionOptions struct {
	Profile     string
	Region      string
	BizType     string
	Locale      string
	HomeDir     string
	Quiet       bool
	Interactive bool
	Prompt      *setupPrompt
	Stdout      io.Writer
	Stderr      io.Writer
}

type setupHomeChoice struct {
	ID   string
	Name string
}

func (app *app) executeSetupPlan(plan setupdomain.Plan, options setupExecutionOptions) (setupdomain.Result, error) {
	if options.Locale == "" {
		options.Locale = plan.Locale
	}
	result := setupdomain.Result{Locale: plan.Locale, Client: plan.Client.ID, Mode: plan.Mode}
	for _, step := range plan.Steps {
		stepResult := setupdomain.StepResult{ID: step.ID, Status: "ok"}
		err := app.executeSetupStep(plan, step, options, &stepResult)
		result.Steps = append(result.Steps, stepResult)
		if err != nil {
			stepResult.Status = "failed"
			result.Steps[len(result.Steps)-1] = stepResult
			return result, fmt.Errorf("step %s failed: %w", step.ID, err)
		}
	}
	result.OK = true
	if plan.Locale == "zh-CN" {
		result.Example = "现在可以直接对 AI 说：帮我打开客厅的灯，并调到适合看电视的亮度。"
	} else {
		result.Example = "You can now tell your AI: Turn on the living room lights and set a comfortable brightness for watching TV."
	}
	return result, nil
}

func (app *app) executeSetupStep(plan setupdomain.Plan, step setupdomain.Step, options setupExecutionOptions, result *setupdomain.StepResult) error {
	switch step.Method {
	case setupdomain.MethodRuntimeCheck:
		result.Message = "running"
		return nil
	case setupdomain.MethodAuthQR:
		flags := cliFlags{values: profileRegionFlags(options)}
		if app.authStatus(flags)["authenticated"] == true {
			result.Status = "skipped"
			result.Message = "already authenticated"
			return nil
		}
		args := []string{"login", "--qr"}
		args = appendProfileRegionArgs(args, options)
		qrOutput := options.Stdout
		if options.Quiet {
			qrOutput = options.Stderr
		}
		if code := app.runAuth(args, nil, qrOutput, options.Stderr); code != exitOK {
			return fmt.Errorf("QR login returned exit code %d", code)
		}
		return nil
	case setupdomain.MethodSkillsCLI:
		return app.runSkillInstaller(step, options)
	case setupdomain.MethodDirectSkill:
		return app.installDirectSkill(step, options)
	case setupdomain.MethodNativeMCP:
		return app.configureMCPClient(plan, options)
	case setupdomain.MethodLANRuntime:
		args := append([]string(nil), step.Command[3:]...)
		args = appendProfileArg(args, options.Profile)
		if code := app.runConfig(args, io.Discard, options.Stderr); code != exitOK {
			return fmt.Errorf("LAN configuration returned exit code %d", code)
		}
		return nil
	case setupdomain.MethodVerify:
		if plan.Mode == setupdomain.ModeLAN {
			lanArgs := []string{"inspect", "--json"}
			lanArgs = appendProfileArg(lanArgs, options.Profile)
			if code := app.runLAN(lanArgs, io.Discard, options.Stderr); code != exitOK {
				return fmt.Errorf("LAN gateway verification returned exit code %d", code)
			}
			if plan.ControlMode == setupdomain.ControlModeLocalOnly {
				return nil
			}
		}
		args := appendProfileRegionArgs([]string{"--json"}, options)
		if code := app.runDoctor(args, io.Discard, options.Stderr); code != exitOK {
			return fmt.Errorf("doctor returned exit code %d", code)
		}
		homeArgs := appendProfileRegionArgs([]string{"list", "--json"}, options)
		var homeOutput bytes.Buffer
		if code := app.runHome(homeArgs, strings.NewReader(""), &homeOutput, options.Stderr); code != exitOK {
			return fmt.Errorf("home list returned exit code %d", code)
		}
		return app.selectDefaultSetupHome(homeOutput.Bytes(), options)
	default:
		return fmt.Errorf("unsupported setup method %q", step.Method)
	}
}

func (app *app) selectDefaultSetupHome(data []byte, options setupExecutionOptions) error {
	contextInfo, err := app.resolveRuntimeContext(cliFlags{values: profileRegionFlags(options)})
	if err != nil {
		return err
	}
	selectedBizType, err := api.NormalizeBizType(options.BizType)
	if err != nil {
		return err
	}
	storedBizType, err := api.NormalizeBizType(contextInfo.Metadata.BizType)
	if err != nil {
		return err
	}
	typeChanged := selectedBizType != storedBizType
	if strings.TrimSpace(contextInfo.HouseID) != "" && !typeChanged {
		return nil
	}
	var result struct {
		Houses []setupHomeChoice `json:"houses"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parse home list verification: %w", err)
	}
	if len(result.Houses) == 0 || strings.TrimSpace(result.Houses[0].ID) == "" {
		return fmt.Errorf("no Yeelight Pro home is available for setup")
	}
	houseID := result.Houses[0].ID
	if len(result.Houses) > 1 && options.Interactive && options.Prompt != nil {
		houseID, err = options.Prompt.chooseHome(firstNonEmpty(options.Locale, contextInfo.Language), result.Houses)
		if err != nil {
			return err
		}
	}
	metadata := mergeProfileMetadata(contextInfo.Metadata, contextInfo.Profile, map[string]string{
		semantic.FieldHouseID: houseID,
		semantic.FieldBizType: selectedBizType,
	})
	return app.metadataStore.Save(metadata)
}

func (app *app) runSkillInstaller(step setupdomain.Step, options setupExecutionOptions) error {
	command := append([]string(nil), step.Command...)
	if len(command) == 0 {
		return fmt.Errorf("skill installer command is empty")
	}
	output := options.Stdout
	if options.Quiet {
		output = options.Stderr
	}
	if len(step.Sources) == 0 {
		return app.runSkillInstallerCommand(command, output, options.Stderr)
	}
	var lastErr error
	for _, source := range step.Sources {
		candidate := append([]string(nil), command...)
		if index := slices.Index(candidate, step.Sources[0]); index >= 0 {
			candidate[index] = source
		}
		if err := app.runSkillInstallerCommand(candidate, output, options.Stderr); err == nil {
			return nil
		} else if strings.Contains(err.Error(), "partial Skill installation failure") {
			return err
		} else {
			lastErr = err
		}
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
	err := app.runSetupProcess(
		context.Background(),
		command,
		io.MultiWriter(stdout, &diagnostics),
		io.MultiWriter(stderr, &diagnostics),
	)
	if err != nil {
		return err
	}
	diagnosticText := strings.ToLower(diagnostics.String())
	if strings.Contains(diagnosticText, "does not support global skill installation") || strings.Contains(diagnosticText, "failed to install") {
		return fmt.Errorf("partial Skill installation failure: at least one selected Agent does not support global installation")
	}
	if strings.Contains(diagnosticText, "no skills found") {
		return fmt.Errorf("Skill source did not expose any installable skills")
	}
	return nil
}

func (app *app) runSetupProcess(ctx context.Context, command []string, stdout io.Writer, stderr io.Writer) error {
	if app.process != nil {
		return app.process(ctx, command, stdout, stderr)
	}
	if len(command) == 0 {
		return fmt.Errorf("command is empty")
	}
	process := exec.CommandContext(ctx, command[0], command[1:]...)
	process.Stdout = stdout
	process.Stderr = stderr
	return process.Run()
}

func (app *app) runSetupProcessWithInput(ctx context.Context, command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if app.processInput != nil {
		return app.processInput(ctx, command, stdin, stdout, stderr)
	}
	if app.process != nil {
		return app.process(ctx, command, stdout, stderr)
	}
	if len(command) == 0 {
		return fmt.Errorf("command is empty")
	}
	process := exec.CommandContext(ctx, command[0], command[1:]...)
	process.Stdin = stdin
	process.Stdout = stdout
	process.Stderr = stderr
	return process.Run()
}

func profileRegionFlags(options setupExecutionOptions) map[string]string {
	values := map[string]string{}
	if options.Profile != "" {
		values["profile"] = options.Profile
	}
	if options.Region != "" {
		values["region"] = options.Region
	}
	if options.BizType != "" {
		values["biz-type"] = options.BizType
	}
	return values
}

func appendProfileRegionArgs(args []string, options setupExecutionOptions) []string {
	args = appendProfileArg(args, options.Profile)
	if options.Region != "" {
		args = append(args, "--region", options.Region)
	}
	if options.BizType != "" {
		args = append(args, "--biz-type", options.BizType)
	}
	return args
}

func appendProfileArg(args []string, profile string) []string {
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	return args
}
