package main

import (
	"fmt"
	"strings"

	setupdomain "github.com/yeelight/yeelight-home/internal/setup"
)

type setupMCPServer struct {
	Name    string
	URL     string
	Headers map[string]string
	Command string
	Args    []string
	Env     map[string]string
}

func (server setupMCPServer) isLocal() bool {
	return strings.TrimSpace(server.Command) != ""
}

func (app *app) configureMCPClient(plan setupdomain.Plan, options setupExecutionOptions) error {
	flags := cliFlags{values: profileRegionFlags(options)}
	contextInfo, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return err
	}
	source := plan.MCPSource
	if source == "" {
		source, err = setupdomain.ParseMCPSource("", plan.Mode)
		if err != nil {
			return err
		}
	}
	servers, err := setupMCPServers(source, contextInfo)
	if err != nil {
		return err
	}
	for _, client := range setupdomain.MCPClientTargets(plan.Client) {
		if err := app.configureOneMCPClient(client, servers, options); err != nil {
			return fmt.Errorf("configure %s: %w", client.Name, err)
		}
	}
	return nil
}

func (app *app) configureOneMCPClient(client setupdomain.Client, servers []setupMCPServer, options setupExecutionOptions) error {
	switch client.MCPAdapter {
	case setupdomain.MCPAdapterClaudeCode:
		return writeClaudeCodeMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterCodex:
		return app.configureCodexMCP(servers, options)
	case setupdomain.MCPAdapterOpenClaw:
		return app.configureOpenClawMCP(servers, options)
	case setupdomain.MCPAdapterHermes:
		return app.configureHermesMCP(servers, options)
	case setupdomain.MCPAdapterFactoryDroid:
		return writeFactoryDroidMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterClaudeDesktop:
		return writeClaudeDesktopMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterStandardJSON:
		return writeMCPServersJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterVSCodeJSON:
		return writeVSCodeMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterGeminiJSON:
		return writeGeminiMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterOpenCodeJSON:
		return writeOpenCodeMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterZedJSON:
		return writeZedMCPJSON(client.MCPConfigPath, servers)
	case setupdomain.MCPAdapterAmpJSON:
		return writeAmpMCPJSON(client.MCPConfigPath, servers)
	default:
		return fmt.Errorf("MCP auto-configuration is not implemented for %s", client.ID)
	}
}

func setupMCPServers(source setupdomain.MCPSource, contextInfo runtimeContext) ([]setupMCPServer, error) {
	switch source {
	case setupdomain.MCPSourceLocal:
		args := []string{"mcp", "serve", "--stdio", "--profile", contextInfo.Profile, "--region", contextInfo.Region}
		if contextInfo.HouseID != "" {
			args = append(args, "--house-id", contextInfo.HouseID)
		}
		if contextInfo.Language != "" {
			args = append(args, "--lang", contextInfo.Language)
		}
		return []setupMCPServer{{Name: "yeelight-home", Command: "yeelight-home", Args: args}}, nil
	case setupdomain.MCPSourceGateway:
		if contextInfo.LANEndpoint == "" {
			return nil, fmt.Errorf("LAN endpoint is not configured")
		}
		return []setupMCPServer{{Name: "yeelight-lan", URL: contextInfo.LANEndpoint}}, nil
	case setupdomain.MCPSourceCloud:
		return setupCloudMCPServers(contextInfo)
	default:
		return nil, fmt.Errorf("unsupported MCP source %q", source)
	}
}

func setupCloudMCPServers(contextInfo runtimeContext) ([]setupMCPServer, error) {
	args := []string{"mcp", "proxy", "--stdio", "--profile", contextInfo.Profile, "--region", contextInfo.Region}
	if contextInfo.HouseID != "" {
		args = append(args, "--house-id", contextInfo.HouseID)
	}
	return []setupMCPServer{
		{Name: "yeelight-metadata", Command: "yeelight-home", Args: append(append([]string{}, args...), "--target", "metadata")},
		{Name: "yeelight-iot", Command: "yeelight-home", Args: append(append([]string{}, args...), "--target", "iot")},
	}, nil
}
