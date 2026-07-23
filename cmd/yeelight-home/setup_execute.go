package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/semantic"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type setupExecutionOptions struct {
	Profile                    string
	Region                     string
	BizType                    string
	Locale                     string
	HomeDir                    string
	Quiet                      bool
	Interactive                bool
	InteractiveSkillsInstaller bool
	Prompt                     *setupPrompt
	Stdin                      io.Reader
	Stdout                     io.Writer
	Stderr                     io.Writer
	Account                    *setupAccountSwitch
}

type setupHomeChoice struct {
	ID   string
	Name string
}

type setupAccountSwitch struct {
	Profile     string
	Token       credential.TokenRecord
	HadToken    bool
	Metadata    credential.ProfileMetadata
	HadMetadata bool
	Changed     bool
}

func (app *app) executeSetupPlan(plan setupdomain.Plan, options setupExecutionOptions) (setupdomain.Result, error) {
	if options.Locale == "" {
		options.Locale = plan.Locale
	}
	if options.Account == nil {
		options.Account = &setupAccountSwitch{}
	}
	result := setupdomain.Result{Locale: plan.Locale, Client: plan.Client.ID, Mode: plan.Mode}
	var deferredErrors []error
	reporter := newSetupStepReporter(options)
	for _, step := range plan.Steps {
		stepResult := setupdomain.StepResult{ID: step.ID, Status: "ok"}
		if !options.Quiet {
			reporter.start(step)
		}
		err := app.executeSetupStep(plan, step, options, &stepResult)
		result.Steps = append(result.Steps, stepResult)
		if err != nil {
			stepResult.Status = "failed"
			result.Steps[len(result.Steps)-1] = stepResult
			if !options.Quiet {
				reporter.finish(step, stepResult, err)
			}
			stepError := fmt.Errorf("step %s failed: %w", step.ID, err)
			if step.Method == setupdomain.MethodSkillsCLI {
				deferredErrors = append(deferredErrors, stepError)
				continue
			}
			deferredErrors = append(deferredErrors, stepError)
			return result, app.rollbackSetupAccountSwitch(options.Account, errors.Join(deferredErrors...))
		}
		if !options.Quiet {
			reporter.finish(step, stepResult, nil)
		}
	}
	if err := app.persistSetupLocale(options.Profile, options.Locale); err != nil {
		deferredErrors = append(deferredErrors, fmt.Errorf("save setup language: %w", err))
		return result, app.rollbackSetupAccountSwitch(options.Account, errors.Join(deferredErrors...))
	}
	if len(deferredErrors) > 0 {
		return result, app.rollbackSetupAccountSwitch(options.Account, errors.Join(deferredErrors...))
	}
	result.OK = true
	if plan.Locale == "zh-CN" {
		result.Example = "现在可以直接对 AI 说：帮我打开客厅的灯，并调到适合看电视的亮度。"
	} else {
		result.Example = "You can now tell your AI: Turn on the living room lights and set a comfortable brightness for watching TV."
	}
	return result, nil
}

func (app *app) persistSetupLocale(profile string, locale string) error {
	flags := cliFlags{values: map[string]string{}}
	if profile != "" {
		flags.values["profile"] = profile
	}
	resolvedProfile, err := app.resolveTargetProfile(flags)
	if err != nil {
		return err
	}
	metadata, _, err := app.metadataStore.Load(resolvedProfile)
	if err != nil {
		return err
	}
	metadata = mergeProfileMetadata(metadata, resolvedProfile, map[string]string{semantic.FieldLanguage: locale})
	return app.metadataStore.Save(metadata)
}

func (app *app) executeSetupStep(plan setupdomain.Plan, step setupdomain.Step, options setupExecutionOptions, result *setupdomain.StepResult) error {
	switch step.Method {
	case setupdomain.MethodRuntimeCheck:
		result.Message = "ready"
		return nil
	case setupdomain.MethodAuthQR:
		reuseCurrentAccount, err := app.reuseCurrentSetupAccount(options)
		if err != nil {
			return err
		}
		if reuseCurrentAccount {
			result.Status = "skipped"
			result.Message = "already authenticated"
			return nil
		}
		account, err := app.captureSetupAccount(options)
		if err != nil {
			return err
		}
		accountState := &account
		if options.Account != nil {
			*options.Account = account
			accountState = options.Account
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
		accountState.Changed = accountState.HadToken || accountState.HadMetadata
		if err := app.clearSetupHomeSelection(options); err != nil {
			return app.rollbackSetupAccountSwitch(accountState, err)
		}
		return nil
	case setupdomain.MethodSkillsCLI:
		report, err := app.runSkillInstallerWithReport(step, options)
		if len(report.Failed) > 0 && len(report.Installed) > 0 {
			result.Status = "warning"
			result.Message = formatSkillInstallSummary(options.Locale, report)
		}
		return err
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

func (app *app) captureSetupAccount(options setupExecutionOptions) (setupAccountSwitch, error) {
	flags := cliFlags{values: profileRegionFlags(options)}
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		return setupAccountSwitch{}, err
	}
	token, hadToken, err := app.tokenStore.Load(profile)
	if err != nil {
		return setupAccountSwitch{}, fmt.Errorf("load current credential: %w", err)
	}
	metadata, hadMetadata, err := app.metadataStore.Load(profile)
	if err != nil {
		return setupAccountSwitch{}, fmt.Errorf("load current profile metadata: %w", err)
	}
	return setupAccountSwitch{
		Profile: profile, Token: token, HadToken: hadToken,
		Metadata: metadata, HadMetadata: hadMetadata,
	}, nil
}

func (app *app) rollbackSetupAccountSwitch(account *setupAccountSwitch, cause error) error {
	if account == nil || !account.Changed {
		return cause
	}
	var rollbackErrors []error
	if account.HadToken {
		if err := app.tokenStore.Save(account.Token); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore previous credential: %w", err))
		}
	} else if err := app.tokenStore.Delete(account.Profile); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("remove replacement credential: %w", err))
	}
	if account.HadMetadata {
		if err := app.metadataStore.Save(account.Metadata); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore previous profile metadata: %w", err))
		}
	} else if err := app.metadataStore.Delete(account.Profile); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("remove replacement profile metadata: %w", err))
	}
	if len(rollbackErrors) == 0 {
		account.Changed = false
		return cause
	}
	return errors.Join(append([]error{cause}, rollbackErrors...)...)
}

func (app *app) reuseCurrentSetupAccount(options setupExecutionOptions) (bool, error) {
	flags := cliFlags{values: profileRegionFlags(options)}
	if app.authStatus(flags)["authenticated"] != true {
		return false, nil
	}
	if !options.Interactive || options.Prompt == nil {
		return true, nil
	}
	return options.Prompt.reuseCurrentAccount(options.Locale)
}

func (app *app) clearSetupHomeSelection(options setupExecutionOptions) error {
	flags := cliFlags{values: profileRegionFlags(options)}
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		return err
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		return err
	}
	metadata.Profile = profile
	metadata.HouseID = ""
	return app.metadataStore.Save(metadata)
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
