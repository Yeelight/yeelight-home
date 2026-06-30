package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/config"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/storage"
)

var version = "dev"

const (
	exitOK            = 0
	exitInvalidInput  = 2
	exitInternalError = 6
)

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return newAppFromEnv().run(args, stdin, stdout, stderr)
}

type app struct {
	qrClient          auth.QRClient
	tokenStore        credential.Store
	metadataStore     credential.FileMetadataStore
	preparedOperation *operation.Prepared
	memoryStore       storage.JSONStore
	topologyCache     topologyCache
	sleep             func(context.Context, time.Duration) error
}

func newAppFromEnv() *app {
	paths := config.ResolveFromEnv()
	fileTokenStore := credential.NewFileTokenStore(filepath.Join(paths.ConfigDir, "tokens.json"))
	app := &app{
		tokenStore:    credential.NewFallbackStore(credential.NewSystemStore("yeelight-home"), fileTokenStore),
		metadataStore: credential.NewFileMetadataStore(filepath.Join(paths.ConfigDir, "profiles.json")),
		memoryStore:   storage.NewJSONStore(filepath.Join(paths.DataDir, "memory.json")),
		topologyCache: newTopologyCache(filepath.Join(paths.CacheDir, "topology.json")),
	}
	return app
}

func (app *app) run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		return printRootHelp(stdout)
	}
	if code, ok := printHelpForArgs(stdout, stderr, args); ok {
		return code
	}
	if isVersionArg(args[0]) {
		return printVersion(args[1:], stdout, stderr)
	}
	switch args[0] {
	case "explain":
		return runExplainAlias(args[1:], stdout, stderr)
	case "api":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "api")
		}
		return app.runAPI(args[1:], stdout, stderr)
	case "auth":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "auth")
		}
		return app.runAuth(args[1:], stdin, stdout, stderr)
	case "config":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "config")
		}
		return app.runConfig(args[1:], stdout, stderr)
	case "completion":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "completion")
		}
		return app.runCompletion(args[1:], stdout, stderr)
	case "doctor":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "doctor")
		}
		return app.runDoctor(args[1:], stdout, stderr)
	case "dev":
		return app.runDev(args[1:], stdout, stderr)
	case "invoke":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "invoke")
		}
		return app.runInvoke(args[1:], stdin, stdout, stderr)
	case "intent":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "intent")
		}
		return runIntent(args[1:], stdout, stderr)
	case "profile":
		if hasSubcommandHelp(args[1:]) {
			return printCommandHelp(stdout, stderr, "profile")
		}
		return app.runProfile(args[1:], stdout, stderr)
	default:
		if _, ok := moduleCommands[args[0]]; ok {
			if hasSubcommandHelp(args[1:]) {
				return printCommandHelp(stdout, stderr, args[0])
			}
			if args[0] == "home" && isNativeHomeCommand(args[1:]) {
				return app.runHome(args[1:], stdout, stderr)
			}
			return app.runModuleCommand(args[0], args[1:], stdout, stderr)
		}
		_, _ = fmt.Fprintf(stderr, "unsupported command %q\n", args[0])
		return exitInvalidInput
	}
}

func hasSubcommandHelp(args []string) bool {
	return len(args) > 0 && isHelpArg(args[0])
}
