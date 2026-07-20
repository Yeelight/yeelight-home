package main

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestNativeMCPAdaptersUseLocalRuntimeCommand(t *testing.T) {
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	tests := []struct {
		name      string
		configure func(*app, []setupMCPServer, setupExecutionOptions) error
		binary    string
	}{
		{name: "codex", configure: (*app).configureCodexMCP, binary: "codex"},
		{name: "openclaw", configure: (*app).configureOpenClawMCP, binary: "openclaw"},
		{name: "hermes", configure: (*app).configureHermesMCP, binary: "hermes"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			var commands [][]string
			app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
				commands = append(commands, append([]string(nil), command...))
				return nil
			}
			if err := test.configure(app, []setupMCPServer{server}, setupExecutionOptions{Stderr: io.Discard}); err != nil {
				t.Fatalf("configure error: %v", err)
			}
			if len(commands) != 1 || commands[0][0] != test.binary {
				t.Fatalf("commands = %#v", commands)
			}
			joined := strings.Join(commands[0], " ")
			if !strings.Contains(joined, "yeelight-home") || !strings.Contains(joined, "--stdio") || strings.Contains(joined, "Authorization") {
				t.Fatalf("add command = %#v", commands[0])
			}
		})
	}
}

func TestHermesMCPConfirmsOverwriteAndToolDiscovery(t *testing.T) {
	app := newTestApp(t)
	var input string
	app.processInput = func(_ context.Context, command []string, stdin io.Reader, _ io.Writer, _ io.Writer) error {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		input = string(data)
		if command[0] != "hermes" {
			t.Fatalf("command = %#v", command)
		}
		return nil
	}
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	if err := app.configureHermesMCP([]setupMCPServer{server}, setupExecutionOptions{Stderr: io.Discard}); err != nil {
		t.Fatalf("configureHermesMCP error: %v", err)
	}
	if input != "y\ny\n" {
		t.Fatalf("Hermes confirmation input = %q", input)
	}
}

func TestOpenClawMCPUsesVerifiedRepeatableArguments(t *testing.T) {
	app := newTestApp(t)
	var commands [][]string
	app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
		commands = append(commands, append([]string(nil), command...))
		return nil
	}
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	if err := app.configureOpenClawMCP([]setupMCPServer{server}, setupExecutionOptions{Stderr: io.Discard}); err != nil {
		t.Fatalf("configureOpenClawMCP error: %v", err)
	}
	want := []string{"openclaw", "mcp", "add", "yeelight-home", "--command", "yeelight-home", "--arg", "mcp", "--arg", "serve", "--arg", "--stdio", "--no-probe"}
	if len(commands) != 1 || !reflect.DeepEqual(commands[0], want) {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestNativeMCPAdapterFailureDoesNotRemoveExistingServer(t *testing.T) {
	app := newTestApp(t)
	var commands [][]string
	app.process = func(_ context.Context, command []string, _ io.Writer, _ io.Writer) error {
		commands = append(commands, append([]string(nil), command...))
		return errors.New("add failed")
	}
	server := setupMCPServer{Name: "yeelight-home", Command: "yeelight-home", Args: []string{"mcp", "serve", "--stdio"}}
	if err := app.configureCodexMCP([]setupMCPServer{server}, setupExecutionOptions{Stderr: io.Discard}); err == nil {
		t.Fatal("configureCodexMCP unexpectedly succeeded")
	}
	if len(commands) != 1 || strings.Contains(strings.Join(commands[0], " "), " remove ") {
		t.Fatalf("commands = %#v", commands)
	}
}
