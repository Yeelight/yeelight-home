package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/yeelight/yeelight-home/internal/i18n"
)

func (app *app) runMCP(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printMCPUsage(stderr)
		return exitInvalidInput
	}
	if args[0] == "proxy" {
		return app.runCloudMCPProxy(args[1:], stdin, stdout, stderr)
	}
	if args[0] != "serve" {
		printMCPUsage(stderr)
		return exitInvalidInput
	}
	flags, err := parseFlags(args[1:])
	if err != nil || !mcpServeFlagsAllowed(flags) || !flags.bool("stdio") {
		printMCPUsage(stderr)
		return exitInvalidInput
	}
	locale, err := app.resolveMCPServerLocale(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp serve: %v\n", err)
		return exitInvalidInput
	}
	if language := flags.string("lang", ""); language != "" {
		flags.values["language"] = language
		delete(flags.values, "lang")
	}
	server := &localMCPServer{app: app, flags: flags, locale: locale, stderr: stderr}
	if err := server.serveStdio(context.Background(), stdin, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp serve: %v\n", err)
		return exitInternalError
	}
	return exitOK
}

func printMCPUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "usage: yeelight-home mcp serve --stdio [--profile <name>] [--region <region>] [--house-id <id>] [--lang <zh-CN|en-US>]")
	_, _ = fmt.Fprintln(writer, "       yeelight-home mcp proxy --stdio --target <metadata|iot> [--profile <name>] [--region <region>] [--house-id <id>]")
}

func (app *app) resolveMCPServerLocale(flags cliFlags) (string, error) {
	if value := flags.string("lang", flags.string("language", "")); value != "" {
		if locale, ok := i18n.Normalize(value); ok {
			return locale, nil
		}
		return "", fmt.Errorf("language must be zh-CN or en-US")
	}
	if contextInfo, err := app.resolveRuntimeContext(flags); err == nil && contextInfo.Language != "" {
		return contextInfo.Language, nil
	}
	if locale, ok := i18n.Detect(os.LookupEnv); ok {
		return locale, nil
	}
	return i18n.Chinese, nil
}

func mcpServeFlagsAllowed(flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "stdio", "profile", "region", "house-id", "lang", "language":
		default:
			return false
		}
	}
	return true
}
