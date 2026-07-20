package i18n

import (
	"fmt"
	"strings"
)

const (
	Chinese = "zh-CN"
	English = "en-US"
)

const (
	RuntimeUnsupportedIntent        = "runtime.unsupported_intent"
	RuntimeInvokeFailed             = "runtime.invoke_failed"
	RuntimeLegacySuccess            = "runtime.legacy_success"
	RuntimeLegacyPartial            = "runtime.legacy_partial"
	RuntimeLegacyClarification      = "runtime.legacy_clarification"
	RuntimeLegacyBlocked            = "runtime.legacy_blocked"
	RuntimeLegacyAuthRequired       = "runtime.legacy_auth_required"
	RuntimeLegacyNotSupported       = "runtime.legacy_not_supported"
	LightPowerSet                   = "light.power_set"
	LightBrightnessSet              = "light.brightness_set"
	LightBrightnessAdjusted         = "light.brightness_adjusted"
	LightColorTemperatureSet        = "light.color_temperature_set"
	LightColorTemperatureAdjusted   = "light.color_temperature_adjusted"
	LightColorSet                   = "light.color_set"
	LightWriteVerificationMismatch  = "light.write_verification_mismatch"
	LightAdjustVerificationMismatch = "light.adjust_verification_mismatch"
	LightNonNumericState            = "light.non_numeric_state"
	LightClarification              = "light.clarification"
	SceneExecuted                   = "scene.executed"
	SceneClarification              = "scene.clarification"
	SetupTitle                      = "setup.title"
	SetupInvalidLanguage            = "setup.invalid_language"
	SetupMissingClient              = "setup.missing_client"
	SetupMissingMode                = "setup.missing_mode"
	SetupUnsupportedCombination     = "setup.unsupported_combination"
	SetupPlanReady                  = "setup.plan_ready"
	SetupChooseLanguage             = "setup.choose_language"
	SetupChooseClient               = "setup.choose_client"
	SetupChooseMode                 = "setup.choose_mode"
	SetupChooseHome                 = "setup.choose_home"
	SetupConfirm                    = "setup.confirm"
	SetupCancelled                  = "setup.cancelled"
	SetupComplete                   = "setup.complete"
	SetupStepRuntime                = "setup.step.runtime"
	SetupStepSkill                  = "setup.step.skill"
	SetupStepMCPLocal               = "setup.step.mcp_local"
	SetupStepMCPCloud               = "setup.step.mcp_cloud"
	SetupStepMCPGateway             = "setup.step.mcp_gateway"
	SetupStepLAN                    = "setup.step.lan"
	SetupStepLogin                  = "setup.step.login"
	SetupStepVerify                 = "setup.step.verify"
	SetupModeSkill                  = "setup.mode.skill"
	SetupModeMCP                    = "setup.mode.mcp"
	SetupModeLAN                    = "setup.mode.lan"
)

var catalogs = map[string]map[string]string{
	Chinese: {
		RuntimeUnsupportedIntent:        "当前 yeelight-home Runtime 不支持这个 intent。请改用 Skill 随附 intent-catalog.json 中的已支持意图，或先用 intent.explain 查询目标意图的公开契约。",
		RuntimeInvokeFailed:             "Runtime 执行失败，已返回可解析错误；调用方可以根据 error.code、error.message 和原始语义请求继续修正或重试。",
		RuntimeLegacySuccess:            "Yeelight Home 操作已成功完成。",
		RuntimeLegacyPartial:            "操作已部分完成，请查看 warnings 和结构化结果。",
		RuntimeLegacyClarification:      "Yeelight Home 继续执行前还需要补充信息。",
		RuntimeLegacyBlocked:            "Yeelight Home 已根据安全策略阻止此操作。",
		RuntimeLegacyAuthRequired:       "请先使用 Yeelight Pro APP 扫码登录。",
		RuntimeLegacyNotSupported:       "当前 Yeelight Home Runtime 不支持此操作。",
		LightPowerSet:                   "已设置 %s 的开关状态。",
		LightBrightnessSet:              "已设置 %s 的亮度。",
		LightBrightnessAdjusted:         "已调整 %s 的亮度。",
		LightColorTemperatureSet:        "已设置 %s 的色温。",
		LightColorTemperatureAdjusted:   "已调整 %s 的色温。",
		LightColorSet:                   "已设置 %s 的颜色。",
		LightWriteVerificationMismatch:  "%s 的控制指令已发送，但写后验证未匹配。",
		LightAdjustVerificationMismatch: "%s 的调节指令已发送，但写后验证未匹配。",
		LightNonNumericState:            "%s 当前属性值不是可验证的数值，已取消调节。",
		LightClarification:              "请明确要控制的灯光设备和目标状态。",
		SceneExecuted:                   "已执行情景：%s。",
		SceneClarification:              "请明确要执行的情景。",
		SetupTitle:                      "Yeelight AI 一键安装向导",
		SetupInvalidLanguage:            "不支持的语言。请选择 zh-CN 或 en-US。",
		SetupMissingClient:              "请选择要使用的 AI 客户端。",
		SetupMissingMode:                "请选择完整智能、轻量连接或局域网优先模式。",
		SetupUnsupportedCombination:     "%s 不支持所选安装模式。",
		SetupPlanReady:                  "安装计划已准备好，共 %d 步。",
		SetupChooseLanguage:             "请选择语言：1. 中文  2. English：",
		SetupChooseClient:               "请选择 AI 客户端：",
		SetupChooseMode:                 "请选择使用方式：1. 完整智能  2. 轻量连接  3. 局域网优先：",
		SetupChooseHome:                 "请选择默认家庭（直接回车使用第一个）：",
		SetupConfirm:                    "确认执行以上安装计划吗？[Y/n]：",
		SetupCancelled:                  "已取消安装，没有修改任何配置。",
		SetupComplete:                   "Yeelight AI 已完成安装和基础验证。",
		SetupStepRuntime:                "检查 yeelight-home 运行环境",
		SetupStepSkill:                  "安装易来全屋智能 Skill",
		SetupStepMCPLocal:               "连接本机 Yeelight Home Runtime",
		SetupStepMCPCloud:               "连接易来云端轻量服务",
		SetupStepMCPGateway:             "让 AI 客户端直接连接家庭网关",
		SetupStepLAN:                    "连接家庭网关的局域网能力",
		SetupStepLogin:                  "使用 Yeelight Pro APP 扫码登录",
		SetupStepVerify:                 "检查登录并读取家庭信息",
		SetupModeSkill:                  "完整智能（推荐，AI 更懂易来家庭）",
		SetupModeMCP:                    "轻量连接（由 AI 自己组织工具）",
		SetupModeLAN:                    "局域网优先（家庭网关就近控制）",
	},
	English: {
		RuntimeUnsupportedIntent:        "This yeelight-home Runtime does not support that intent. Use a supported intent from the Skill intent catalog, or inspect its public contract with intent.explain.",
		RuntimeInvokeFailed:             "The Runtime could not complete the request. A structured error was returned so the caller can correct or retry the original request.",
		RuntimeLegacySuccess:            "The Yeelight Home operation completed successfully.",
		RuntimeLegacyPartial:            "The operation completed partially. Review the warnings and structured result for details.",
		RuntimeLegacyClarification:      "More information is needed before Yeelight Home can continue.",
		RuntimeLegacyBlocked:            "Yeelight Home blocked this operation under its safety policy.",
		RuntimeLegacyAuthRequired:       "Sign in with the Yeelight Pro APP before continuing.",
		RuntimeLegacyNotSupported:       "This operation is not supported by the current Yeelight Home Runtime.",
		LightPowerSet:                   "Set the power state for %s.",
		LightBrightnessSet:              "Set the brightness for %s.",
		LightBrightnessAdjusted:         "Adjusted the brightness for %s.",
		LightColorTemperatureSet:        "Set the color temperature for %s.",
		LightColorTemperatureAdjusted:   "Adjusted the color temperature for %s.",
		LightColorSet:                   "Set the color for %s.",
		LightWriteVerificationMismatch:  "The control command was sent to %s, but the read-back value did not match.",
		LightAdjustVerificationMismatch: "The adjustment command was sent to %s, but the read-back value did not match.",
		LightNonNumericState:            "The current value for %s is not numeric, so the adjustment was cancelled.",
		LightClarification:              "Choose the light, room, area, or group to control and provide the desired value.",
		SceneExecuted:                   "Ran the scene: %s.",
		SceneClarification:              "Choose the scene to run.",
		SetupTitle:                      "Yeelight AI guided setup",
		SetupInvalidLanguage:            "Unsupported language. Choose zh-CN or en-US.",
		SetupMissingClient:              "Choose the AI client you want to use.",
		SetupMissingMode:                "Choose the full Skill, lightweight MCP, or LAN-preferred mode.",
		SetupUnsupportedCombination:     "%s does not support the selected setup mode.",
		SetupPlanReady:                  "The setup plan is ready with %d steps.",
		SetupChooseLanguage:             "Choose a language: 1. 中文  2. English: ",
		SetupChooseClient:               "Choose an AI client: ",
		SetupChooseMode:                 "Choose how to connect: 1. Full Skill  2. Lightweight MCP  3. LAN preferred: ",
		SetupChooseHome:                 "Choose the default home (press Enter to use the first):",
		SetupConfirm:                    "Run this setup plan? [Y/n]: ",
		SetupCancelled:                  "Setup was cancelled. No configuration was changed.",
		SetupComplete:                   "Yeelight AI setup and basic verification are complete.",
		SetupStepRuntime:                "Check the yeelight-home Runtime",
		SetupStepSkill:                  "Install the Yeelight whole-home Skill",
		SetupStepMCPLocal:               "Connect the local Yeelight Home Runtime",
		SetupStepMCPCloud:               "Connect the lightweight Yeelight cloud services",
		SetupStepMCPGateway:             "Connect the AI client directly to the home gateway",
		SetupStepLAN:                    "Connect the home gateway over the local network",
		SetupStepLogin:                  "Sign in with the Yeelight Pro APP",
		SetupStepVerify:                 "Check sign-in and read the home list",
		SetupModeSkill:                  "Full intelligence (recommended, with Yeelight home guidance)",
		SetupModeMCP:                    "Lightweight connection (your AI organizes the tools)",
		SetupModeLAN:                    "LAN preferred (control through the home gateway)",
	},
}

func Normalize(value string) (string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	if base, _, found := strings.Cut(normalized, "."); found {
		normalized = base
	}
	switch {
	case normalized == "zh", normalized == "zh-cn", normalized == "zh-hans", strings.HasPrefix(normalized, "zh-hans-"):
		return Chinese, true
	case normalized == "en", normalized == "en-us", strings.HasPrefix(normalized, "en-"):
		return English, true
	default:
		return "", false
	}
}

func IsSupported(value string) bool {
	return value == Chinese || value == English
}

func Detect(lookup func(string) (string, bool)) (string, bool) {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		if value, ok := lookup(name); ok {
			if locale, supported := Normalize(value); supported {
				return locale, true
			}
		}
	}
	return "", false
}

func Template(locale string, key string) string {
	if !IsSupported(locale) {
		locale = English
	}
	if value := catalogs[locale][key]; value != "" {
		return value
	}
	if value := catalogs[English][key]; value != "" {
		return value
	}
	return key
}

func Text(locale string, key string, args ...any) string {
	return fmt.Sprintf(Template(locale, key), args...)
}
