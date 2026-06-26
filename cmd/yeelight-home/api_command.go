package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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
		var statusErr api.HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			_, _ = fmt.Fprintln(stderr, "api smoke: authorization failed; token is missing, invalid, expired, or not accepted by this region. Run yeelight-home auth login --qr --region "+contextInfo.Region+" or set a valid YEELIGHT_HOME_ACCESS_TOKEN.")
			return exitInvalidInput
		}
		_, _ = fmt.Fprintf(stderr, "api smoke: %v\n", err)
		return exitInternalError
	}
	if !flags.bool("json") {
		return writeAPISmokeText(stdout, result)
	}
	return writeJSON(stdout, stderr, result)
}

func writeAPISmokeText(stdout io.Writer, result api.SmokeResult) int {
	_, _ = fmt.Fprintln(stdout, "Yeelight Home API Smoke")
	_, _ = fmt.Fprintf(stdout, "Region: %s\n", result.Region)
	_, _ = fmt.Fprintf(stdout, "Account: %s\n", okText(result.AccountOK))
	_, _ = fmt.Fprintf(stdout, "Home list: %s\n", okText(result.HouseListOK))
	_, _ = fmt.Fprintf(stdout, "House count: %d\n", result.HouseCount)
	if result.HouseListSource != "" {
		_, _ = fmt.Fprintf(stdout, "Home list source: %s\n", result.HouseListSource)
	}
	if result.HouseListAPICalls > 0 {
		_, _ = fmt.Fprintf(stdout, "Home list API calls: %d\n", result.HouseListAPICalls)
	}
	return exitOK
}

func okText(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}
