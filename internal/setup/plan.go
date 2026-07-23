package setup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

var skillRepositories = []string{
	"https://github.com/Yeelight/yeelight-smart-home-skills",
	"https://gitee.com/yeelight/yeelight-smart-home-skills.git",
	"https://gitcode.com/Yeelight/yeelight-smart-home-skills.git",
}

const skillsInstallerPackage = "skills@1.5.20"

func BuildPlan(options Options) (Plan, error) {
	if !i18n.IsSupported(options.Locale) {
		return Plan{}, fmt.Errorf("locale must be zh-CN or en-US")
	}
	mcpSource, err := ParseMCPSource(options.MCPSource, options.Mode)
	if err != nil {
		return Plan{}, err
	}
	controlMode, err := ParseControlMode(options.ControlMode, options.Mode)
	if err != nil {
		return Plan{}, err
	}
	clientMode := options.Mode
	if options.Mode == ModeLAN && mcpSource == MCPSourceGateway {
		clientMode = ModeMCP
	}
	var client Client
	if options.InteractiveSkillsInstaller && clientMode != ModeMCP {
		client = Client{ID: "auto", Name: "Vercel Skills installer", SupportsSkill: true}
	} else {
		client, err = resolveClient(options.HomeDir, options.ClientID, clientMode, options.LookPath)
		if err != nil {
			return Plan{}, err
		}
	}
	if options.Mode == ModeLAN && strings.TrimSpace(options.GatewayIP) == "" {
		return Plan{}, fmt.Errorf("gateway IP is required for LAN mode")
	}
	if options.Mode == ModeLAN {
		if _, err := lanmcp.EndpointForGateway(options.GatewayIP); err != nil {
			return Plan{}, err
		}
	}

	plan := Plan{Locale: options.Locale, Client: client, Mode: options.Mode, MCPSource: mcpSource, GatewayIP: options.GatewayIP, ControlMode: controlMode, BizType: options.BizType}
	plan.Steps = append(plan.Steps, Step{ID: "runtime", Title: i18n.Text(options.Locale, i18n.SetupStepRuntime), Method: MethodRuntimeCheck, Command: []string{"yeelight-home", "version", "--json"}})
	if options.Mode != ModeLAN || controlMode != ControlModeLocalOnly {
		plan.Steps = append(plan.Steps, Step{ID: "login", Title: i18n.Text(options.Locale, i18n.SetupStepLogin), Method: MethodAuthQR, Command: []string{"yeelight-home", "auth", "login", "--qr"}})
	}
	switch options.Mode {
	case ModeSkill:
		plan.Steps = append(plan.Steps, skillStep(client, options.Locale))
	case ModeMCP:
		plan.Steps = append(plan.Steps, mcpStep(client, options.Locale, mcpSource))
	case ModeLAN:
		plan.Steps = append(plan.Steps, Step{
			ID:      "lan",
			Title:   i18n.Text(options.Locale, i18n.SetupStepLAN),
			Method:  MethodLANRuntime,
			Command: []string{"yeelight-home", "config", "set", "--control-mode", controlMode, "--gateway-ip", options.GatewayIP},
		})
		if client.SupportsSkill && mcpSource != MCPSourceGateway {
			plan.Steps = append(plan.Steps, skillStep(client, options.Locale))
		} else {
			plan.Steps = append(plan.Steps, mcpStep(client, options.Locale, mcpSource))
		}
	}
	plan.Steps = append(plan.Steps, Step{ID: "verify", Title: i18n.Text(options.Locale, i18n.SetupStepVerify), Method: MethodVerify, Command: []string{"yeelight-home", "doctor", "--json"}})
	return plan, nil
}

func mcpStep(client Client, locale string, source MCPSource) Step {
	key := i18n.SetupStepMCPLocal
	switch source {
	case MCPSourceCloud:
		key = i18n.SetupStepMCPCloud
	case MCPSourceGateway:
		key = i18n.SetupStepMCPGateway
	}
	return Step{ID: "mcp", Title: i18n.Text(locale, key), Method: MethodNativeMCP, Destination: client.MCPConfigPath}
}

func skillStep(client Client, locale string) Step {
	if len(client.SkillAgents) > 0 {
		command := []string{
			"npx", "-y", skillsInstallerPackage, "add", skillRepositories[0],
			"--skill", "yeelight-smart-home", "--global",
		}
		command = append(command, "--agent")
		command = append(command, client.SkillAgents...)
		return Step{
			ID:      "skill",
			Title:   i18n.Text(locale, i18n.SetupStepSkill),
			Method:  MethodSkillsCLI,
			Command: command,
			Sources: append([]string(nil), skillRepositories...),
		}
	}
	if client.SkillPath == "" {
		return Step{
			ID:     "skill",
			Title:  i18n.Text(locale, i18n.SetupStepSkill),
			Method: MethodSkillsCLI,
			Command: []string{
				"npx", "-y", skillsInstallerPackage, "add", skillRepositories[0],
				"--skill", "yeelight-smart-home", "--global",
			},
			Sources: append([]string(nil), skillRepositories...),
		}
	}
	return Step{
		ID:          "skill",
		Title:       i18n.Text(locale, i18n.SetupStepSkill),
		Method:      MethodDirectSkill,
		Sources:     append([]string(nil), skillRepositories...),
		Destination: filepath.Clean(client.SkillPath),
	}
}
