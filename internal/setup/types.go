package setup

import "fmt"

type Mode string

const (
	ModeSkill Mode = "skill"
	ModeMCP   Mode = "mcp"
	ModeLAN   Mode = "lan"
)

type MCPSource string

const (
	MCPSourceLocal   MCPSource = "local"
	MCPSourceCloud   MCPSource = "cloud"
	MCPSourceGateway MCPSource = "gateway"
)

const (
	ControlModeLocalPreferred = "local-preferred"
	ControlModeLocalOnly      = "local-only"
)

type InstallMethod string

const (
	MethodRuntimeCheck InstallMethod = "runtime-check"
	MethodSkillsCLI    InstallMethod = "skills-cli"
	MethodDirectSkill  InstallMethod = "direct-skill"
	MethodNativeMCP    InstallMethod = "native-mcp-config"
	MethodLANRuntime   InstallMethod = "lan-runtime"
	MethodAuthQR       InstallMethod = "auth-qr"
	MethodVerify       InstallMethod = "verify"
)

type Client struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	SkillAgents   []string `json:"skillAgents,omitempty"`
	SkillPath     string   `json:"skillPath,omitempty"`
	MCPAdapter    string   `json:"mcpAdapter,omitempty"`
	MCPConfigPath string   `json:"mcpConfigPath,omitempty"`
	MCPTargets    []Client `json:"mcpTargets,omitempty"`
	SupportsSkill bool     `json:"supportsSkill"`
	SupportsMCP   bool     `json:"supportsMcp"`
}

func (client Client) Supports(mode Mode) bool {
	switch mode {
	case ModeSkill:
		return client.SupportsSkill
	case ModeMCP:
		return client.SupportsMCP
	case ModeLAN:
		return client.SupportsSkill || client.SupportsMCP
	default:
		return false
	}
}

type Step struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Method      InstallMethod `json:"method"`
	Command     []string      `json:"command,omitempty"`
	Sources     []string      `json:"sources,omitempty"`
	Destination string        `json:"destination,omitempty"`
}

type StepResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Result struct {
	OK      bool         `json:"ok"`
	Locale  string       `json:"locale"`
	Client  string       `json:"client"`
	Mode    Mode         `json:"mode"`
	Steps   []StepResult `json:"steps"`
	Example string       `json:"example,omitempty"`
}

type Plan struct {
	Locale      string    `json:"locale"`
	Client      Client    `json:"client"`
	Mode        Mode      `json:"mode"`
	MCPSource   MCPSource `json:"mcpSource,omitempty"`
	GatewayIP   string    `json:"gatewayIp,omitempty"`
	ControlMode string    `json:"controlMode,omitempty"`
	BizType     string    `json:"bizType"`
	Steps       []Step    `json:"steps"`
}

type Options struct {
	Locale      string
	ClientID    string
	Mode        Mode
	MCPSource   string
	GatewayIP   string
	ControlMode string
	BizType     string
	HomeDir     string
	LookPath    func(string) (string, error)
}

func ParseControlMode(value string, mode Mode) (string, error) {
	if mode != ModeLAN {
		if value != "" {
			return "", fmt.Errorf("control mode requires setup mode lan")
		}
		return "", nil
	}
	if value == "" {
		return ControlModeLocalPreferred, nil
	}
	switch value {
	case ControlModeLocalPreferred, ControlModeLocalOnly:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported LAN control mode %q", value)
	}
}

func ParseMCPSource(value string, mode Mode) (MCPSource, error) {
	if value == "" {
		if mode == ModeMCP || mode == ModeLAN {
			return MCPSourceLocal, nil
		}
		return "", nil
	}
	source := MCPSource(value)
	switch source {
	case MCPSourceLocal:
		return source, nil
	case MCPSourceCloud:
		if mode != ModeMCP {
			return "", fmt.Errorf("MCP source cloud requires mode mcp")
		}
		return source, nil
	case MCPSourceGateway:
		if mode != ModeLAN {
			return "", fmt.Errorf("MCP source gateway requires mode lan")
		}
		return source, nil
	default:
		return "", fmt.Errorf("unsupported MCP source %q", value)
	}
}

func ParseMode(value string) (Mode, error) {
	mode := Mode(value)
	switch mode {
	case ModeSkill, ModeMCP, ModeLAN:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported setup mode %q", value)
	}
}
