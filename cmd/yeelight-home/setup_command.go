package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/i18n"
	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
	"golang.org/x/term"
)

func (app *app) runSetup(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	if !setupFlagsAllowed(flags) {
		_, _ = fmt.Fprintln(stderr, "setup: unsupported flag")
		return exitInvalidInput
	}
	interactive := app.isTerminal(stdin)
	prompting := interactive && !flags.bool("json") && !flags.bool("yes")
	rich := prompting && isTerminalWriter(stdout) && !flags.bool("plan")
	prompt := newSetupPrompt(stdin, stdout, rich)
	locale, err := app.resolveSetupLocale(flags, prompting, prompt)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	modeValue := flags.string("mode", "")
	if modeValue == "" && prompting {
		modeValue, err = prompt.chooseMode(locale)
	}
	if err != nil || modeValue == "" {
		_, _ = fmt.Fprintln(stderr, "setup: "+i18n.Text(locale, i18n.SetupMissingMode))
		return exitInvalidInput
	}
	mode, err := setupdomain.ParseMode(modeValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	profile, profileErr := app.resolveTargetProfile(flags)
	storedBizType := ""
	if profileErr == nil {
		if metadata, ok, loadErr := app.metadataStore.Load(profile); loadErr == nil && ok {
			storedBizType = metadata.BizType
		}
	}
	bizTypeValue := flags.string("biz-type", flags.string("bizType", ""))
	bizType, err := api.NormalizeBizType(firstNonEmpty(bizTypeValue, storedBizType, api.BizTypeConsumer))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	homeDir := flags.string("home-dir", "")
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "setup: resolve home directory: %v\n", err)
			return exitInternalError
		}
	}
	agentID := flags.string("agent", flags.string("client", ""))
	interactiveSkillsInstaller := rich && mode != setupdomain.ModeMCP && (agentID == "" || strings.EqualFold(agentID, "auto"))
	if agentID == "" && mode == setupdomain.ModeMCP && prompting {
		agentID, err = prompt.chooseMCPClient(locale, preferredMCPClients(homeDir))
	}
	if err != nil {
		if errors.Is(err, errSetupCancelled) {
			_, _ = fmt.Fprintln(stdout, i18n.Text(locale, i18n.SetupCancelled))
			return exitOK
		}
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	if agentID == "" {
		if mode == setupdomain.ModeMCP {
			_, _ = fmt.Fprintln(stderr, "setup: "+i18n.Text(locale, i18n.SetupMissingClient))
			return exitInvalidInput
		}
		agentID = "auto"
	}
	plan, err := setupdomain.BuildPlan(setupdomain.Options{
		Locale: locale, ClientID: agentID, Mode: mode,
		BizType:                    bizType,
		MCPSource:                  flags.string("mcp-source", ""),
		GatewayIP:                  flags.string("gateway-ip", ""),
		ControlMode:                flags.string("control-mode", ""),
		HomeDir:                    homeDir,
		InteractiveSkillsInstaller: interactiveSkillsInstaller,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInvalidInput
	}
	if flags.bool("plan") || (!interactive && !flags.bool("yes")) {
		return writeSetupPlan(plan, flags.bool("json"), stdout, stderr)
	}
	if prompting {
		if prompt.rich {
			writeSetupPlanRich(plan, prompt)
		} else {
			writeSetupPlanText(plan, stdout)
		}
		confirmed, confirmErr := prompt.confirm(locale)
		if confirmErr != nil {
			_, _ = fmt.Fprintf(stderr, "setup: %v\n", confirmErr)
			return exitInvalidInput
		}
		if !confirmed {
			_, _ = fmt.Fprintln(stdout, i18n.Text(locale, i18n.SetupCancelled))
			return exitOK
		}
	}
	result, err := app.executeSetupPlan(plan, setupExecutionOptions{
		Profile: flags.string("profile", ""), Region: flags.string("region", ""), BizType: bizType,
		Locale: locale, HomeDir: homeDir, Quiet: flags.bool("json"), Interactive: prompting, Prompt: prompt,
		Stdin: stdin, Stdout: stdout, Stderr: stderr, InteractiveSkillsInstaller: interactiveSkillsInstaller,
	})
	if err != nil {
		if errors.Is(err, errSetupCancelled) {
			_, _ = fmt.Fprintln(stdout, i18n.Text(locale, i18n.SetupCancelled))
			return exitOK
		}
		if flags.bool("json") {
			_ = json.NewEncoder(stdout).Encode(result)
		}
		_, _ = fmt.Fprintf(stderr, "setup: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	completionKey := i18n.SetupComplete
	if setupResultHasWarnings(result) {
		completionKey = i18n.SetupCompleteWithWarnings
	}
	_, _ = fmt.Fprintln(stdout, i18n.Text(locale, completionKey))
	_, _ = fmt.Fprintln(stdout, result.Example)
	return exitOK
}

func setupResultHasWarnings(result setupdomain.Result) bool {
	for _, step := range result.Steps {
		if step.Status == "warning" {
			return true
		}
	}
	return false
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func preferredMCPClients(homeDir string) []setupdomain.Client {
	return orderMCPClients(setupdomain.MCPClients(homeDir), setupdomain.DetectMCPClients(homeDir, nil))
}

func orderMCPClients(all []setupdomain.Client, detected []setupdomain.Client) []setupdomain.Client {
	if len(detected) == 0 {
		return all
	}
	seen := make(map[string]bool, len(all))
	ordered := make([]setupdomain.Client, 0, len(all))
	for _, client := range detected {
		seen[client.ID] = true
		ordered = append(ordered, client)
	}
	for _, client := range all {
		if !seen[client.ID] {
			ordered = append(ordered, client)
		}
	}
	return ordered
}

func setupFlagsAllowed(flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "lang", "language", "mode", "agent", "client", "mcp-source", "gateway-ip", "control-mode", "profile", "region", "biz-type", "bizType", "home-dir", "plan", "yes", "json":
		default:
			return false
		}
	}
	return true
}

func (app *app) resolveSetupLocale(flags cliFlags, interactive bool, prompt *setupPrompt) (string, error) {
	if raw := flags.string("lang", flags.string("language", "")); raw != "" {
		locale, ok := i18n.Normalize(raw)
		if !ok {
			return "", fmt.Errorf("language must be zh-CN or en-US")
		}
		return locale, nil
	}
	profile, err := app.resolveTargetProfile(flags)
	if err == nil {
		if metadata, ok, loadErr := app.metadataStore.Load(profile); loadErr == nil && ok {
			if locale, supported := i18n.Normalize(metadata.Language); supported {
				return locale, nil
			}
		}
	}
	if locale, ok := i18n.Detect(os.LookupEnv); ok {
		return locale, nil
	}
	if interactive {
		return prompt.chooseLanguage()
	}
	return "", fmt.Errorf("language is required in a non-interactive environment; use --lang zh-CN or --lang en-US")
}

func writeSetupPlan(plan setupdomain.Plan, asJSON bool, stdout io.Writer, stderr io.Writer) int {
	if asJSON {
		return writeJSON(stdout, stderr, plan)
	}
	writeSetupPlanText(plan, stdout)
	return exitOK
}

func writeSetupPlanText(plan setupdomain.Plan, stdout io.Writer) {
	_, _ = fmt.Fprintln(stdout, i18n.Text(plan.Locale, i18n.SetupTitle))
	_, _ = fmt.Fprintf(stdout, "%s / %s\n", plan.Client.Name, plan.Mode)
	for index, step := range plan.Steps {
		_, _ = fmt.Fprintf(stdout, "%d. %s\n", index+1, strings.TrimSpace(step.Title))
	}
}
