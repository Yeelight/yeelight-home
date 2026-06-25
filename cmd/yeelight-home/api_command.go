package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
)

func (app *app) runAPI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home api <smoke>")
		return exitInvalidInput
	}
	switch args[0] {
	case "smoke":
		return app.runAPISmoke(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported api command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runAPISmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "api smoke: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home api smoke --json [--region cn]")
		return exitInvalidInput
	}
	contextInfo, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "api smoke: %v\n", err)
		return exitInvalidInput
	}
	credentials := api.SmokeCredentials{
		Authorization: contextInfo.AccessToken,
		ClientID:      contextInfo.ClientID,
		HouseID:       contextInfo.HouseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		_, _ = fmt.Fprintln(stderr, "api smoke: missing token; run auth login --qr or auth token set")
		return exitInvalidInput
	}
	result, err := api.NewSmokeClient(contextInfo.Endpoint, nil).Run(context.Background(), credentials)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "api smoke: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}
