package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

func (app *app) runLAN(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return printCommandHelp(stdout, stderr, "lan")
	}
	action := args[0]
	toolName := ""
	flagArgs := args[1:]
	if action == "call" && len(flagArgs) > 0 && !strings.HasPrefix(flagArgs[0], "--") {
		toolName = flagArgs[0]
		flagArgs = flagArgs[1:]
	}
	flags, err := parseFlags(flagArgs)
	if err != nil || !lanFlagsAllowed(action, flags) {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home lan <inspect|tools|call> [tool] [--args-json <json>] [--yes] [--json] [--profile <name>] [--gateway-ip <ip>|--lan-endpoint <url>]")
		return exitInvalidInput
	}
	contextInfo, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "lan %s: %v\n", action, err)
		return exitInvalidInput
	}
	if contextInfo.LANEndpoint == "" {
		_, _ = fmt.Fprintln(stderr, "lan: configure --gateway-ip or --lan-endpoint first")
		return exitInvalidInput
	}
	client, err := lanmcp.NewClient(contextInfo.LANEndpoint, lanmcp.Options{})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "lan %s: %v\n", action, err)
		return exitInvalidInput
	}
	tools, err := client.ListAllTools(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "lan %s: %v\n", action, err)
		return exitInternalError
	}
	switch action {
	case "inspect", "tools":
		return writeLANInspection(stdout, stderr, flags.bool("json"), contextInfo, tools)
	case "call":
		return runLANCall(client, tools, toolName, flags, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported lan action %q\n", action)
		return exitInvalidInput
	}
}

func runLANCall(client *lanmcp.Client, tools lanmcp.ListToolsResult, toolName string, flags cliFlags, stdout io.Writer, stderr io.Writer) int {
	if toolName == "" {
		_, _ = fmt.Fprintln(stderr, "lan call: tool name is required")
		return exitInvalidInput
	}
	var selected *lanmcp.Tool
	for index := range tools.Tools {
		if tools.Tools[index].Name == toolName {
			selected = &tools.Tools[index]
			break
		}
	}
	if selected == nil {
		_, _ = fmt.Fprintf(stderr, "lan call: gateway does not expose tool %q\n", toolName)
		return exitInvalidInput
	}
	arguments := map[string]any{}
	if raw := flags.string("args-json", flags.string("args", "")); raw != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		if err := decoder.Decode(&arguments); err != nil {
			_, _ = fmt.Fprintf(stderr, "lan call: invalid arguments JSON: %v\n", err)
			return exitInvalidInput
		}
	}
	if !flags.bool("yes") {
		preview := map[string]any{
			"ok": true, "executed": false, "tool": selected,
			"arguments": arguments, "nextStep": "Review the tool schema and rerun with --yes to execute.",
		}
		return writeJSON(stdout, stderr, preview)
	}
	result, err := client.CallTool(context.Background(), tools.Session, toolName, arguments)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "lan call: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}

func writeLANInspection(stdout io.Writer, stderr io.Writer, asJSON bool, contextInfo runtimeContext, tools lanmcp.ListToolsResult) int {
	result := map[string]any{
		"ok": true, "endpoint": contextInfo.LANEndpoint,
		"protocolVersion": tools.Session.ProtocolVersion,
		"stateless":       tools.Session.Stateless,
		"serverInfo":      tools.Session.ServerInfo,
		"toolCount":       len(tools.Tools), "tools": tools.Tools,
	}
	if asJSON {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "LAN MCP: %s\nProtocol: %s\nTools: %d\n", contextInfo.LANEndpoint, tools.Session.ProtocolVersion, len(tools.Tools))
	for _, tool := range tools.Tools {
		_, _ = fmt.Fprintf(stdout, "- %s: %s\n", tool.Name, strings.TrimSpace(tool.Description))
	}
	return exitOK
}

func lanFlagsAllowed(action string, flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "json", "profile", "region", "house-id", "gateway-ip", "lan-endpoint", "control-mode":
		case "args", "args-json", "yes":
			if action != "call" {
				return false
			}
		default:
			return false
		}
	}
	return true
}
