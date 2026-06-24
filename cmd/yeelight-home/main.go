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
	"github.com/yeelight/yeelight-home/internal/plan"
	"github.com/yeelight/yeelight-home/internal/storage"
)

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
	qrClient      auth.QRClient
	tokenStore    credential.Store
	metadataStore credential.FileMetadataStore
	planStore     plan.Store
	memoryStore   storage.JSONStore
	sleep         func(context.Context, time.Duration) error
}

func newAppFromEnv() *app {
	paths := config.ResolveFromEnv()
	fileTokenStore := credential.NewFileTokenStore(filepath.Join(paths.ConfigDir, "tokens.json"))
	return &app{
		tokenStore:    credential.NewFallbackStore(credential.NewSystemStore("yeelight-home"), fileTokenStore),
		metadataStore: credential.NewFileMetadataStore(filepath.Join(paths.ConfigDir, "profiles.json")),
		planStore:     plan.NewStore(filepath.Join(paths.DataDir, "pending_plans.json")),
		memoryStore:   storage.NewJSONStore(filepath.Join(paths.DataDir, "memory.json")),
	}
}

func (app *app) run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "missing command")
		return exitInvalidInput
	}
	switch args[0] {
	case "api":
		return app.runAPI(args[1:], stdout, stderr)
	case "approve":
		return app.runApprove(args[1:], stdout, stderr)
	case "auth":
		return app.runAuth(args[1:], stdout, stderr)
	case "config":
		return app.runConfig(args[1:], stdout, stderr)
	case "doctor":
		return app.runDoctor(args[1:], stdout, stderr)
	case "dev":
		return app.runDev(args[1:], stdout, stderr)
	case "home":
		return app.runHome(args[1:], stdout, stderr)
	case "invoke":
		return app.runInvoke(args[1:], stdin, stdout, stderr)
	case "profile":
		return app.runProfile(args[1:], stdout, stderr)
	case "version":
		_, _ = fmt.Fprintln(stdout, "yeelight-home contract 1.0")
		return exitOK
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "doctor: %v\n", err)
		return exitInvalidInput
	}
	if err := requireJSONFlag(flags, "usage: yeelight-home doctor --json [--profile <name>] [--region <region>] [--house-id <id>]"); err != nil {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home doctor --json")
		return exitInvalidInput
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "doctor: %v\n", err)
		return exitInvalidInput
	}
	authStatus := auth.StatusFromEnv()
	status := "ok"
	warnings := []string{}
	if !context.TokenPresent && !authStatus.Authenticated {
		status = "warning"
		warnings = append(warnings, "auth_required")
	}
	paths := config.ResolveFromEnv()
	response := map[string]any{
		"status":        status,
		"warnings":      warnings,
		"authenticated": context.TokenPresent || authStatus.Authenticated,
		"profile":       context.Profile,
		"region":        context.Region,
		"clientId":      context.ClientID,
		"houseId":       context.HouseID,
		"tokenPresent":  context.TokenPresent,
		"tokenSource":   context.TokenSource,
		"homeDir":       paths.HomeDir,
		"configDir":     paths.ConfigDir,
		"dataDir":       paths.DataDir,
		"cacheDir":      paths.CacheDir,
		"install": map[string]any{
			"cli":        "yeelight-home",
			"publicRepo": "yeelight/yeelight-home",
		},
		"memoryMigrations": map[string]any{
			"status": "available",
			"count":  len(storage.MemoryMigrations()),
		},
	}
	return writeJSON(stdout, stderr, response)
}
