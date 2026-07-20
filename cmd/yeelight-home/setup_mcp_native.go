package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
)

func (app *app) configureCodexMCP(servers []setupMCPServer, options setupExecutionOptions) error {
	for _, server := range servers {
		command := []string{"codex", "mcp", "add", server.Name}
		if server.isLocal() {
			command = append(command, "--", server.Command)
			command = append(command, server.Args...)
		} else if len(server.Headers) == 0 {
			command = append(command, "--url", server.URL)
		} else {
			remoteArgs := []string{"--", "npx", "-y", "mcp-remote", server.URL}
			for index, name := range sortedStringKeys(server.Headers) {
				envName := fmt.Sprintf("YEELIGHT_MCP_HEADER_%d", index+1)
				command = append(command, "--env", envName+"="+server.Headers[name])
				remoteArgs = append(remoteArgs, "--header", name+":${"+envName+"}")
			}
			command = append(command, remoteArgs...)
		}
		if err := app.runSetupProcess(context.Background(), command, io.Discard, options.Stderr); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) configureOpenClawMCP(servers []setupMCPServer, options setupExecutionOptions) error {
	for _, server := range servers {
		command := []string{"openclaw", "mcp", "add", server.Name}
		if server.isLocal() {
			command = append(command, "--command", server.Command)
			for _, argument := range server.Args {
				command = append(command, "--arg", argument)
			}
		} else {
			command = append(command, "--url", server.URL, "--transport", "streamable-http")
			for _, name := range sortedStringKeys(server.Headers) {
				command = append(command, "--header", name+"="+server.Headers[name])
			}
		}
		command = append(command, "--no-probe")
		if err := app.runSetupProcess(context.Background(), command, io.Discard, options.Stderr); err != nil {
			return err
		}
	}
	return nil
}

func (app *app) configureHermesMCP(servers []setupMCPServer, options setupExecutionOptions) error {
	for _, server := range servers {
		command := []string{"hermes", "mcp", "add", server.Name}
		if server.isLocal() {
			command = append(command, "--command", server.Command, "--args")
			command = append(command, server.Args...)
		} else if len(server.Headers) == 0 {
			command = append(command, "--url", server.URL)
		} else {
			command = append(command, "--command", "npx", "--env")
			remoteArgs := []string{"--args", "-y", "mcp-remote", server.URL}
			for index, name := range sortedStringKeys(server.Headers) {
				envName := fmt.Sprintf("YEELIGHT_MCP_HEADER_%d", index+1)
				command = append(command, envName+"="+server.Headers[name])
				remoteArgs = append(remoteArgs, "--header", name+":${"+envName+"}")
			}
			command = append(command, remoteArgs...)
		}
		if err := app.runSetupProcessWithInput(context.Background(), command, strings.NewReader("y\ny\n"), io.Discard, options.Stderr); err != nil {
			return err
		}
	}
	return nil
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
