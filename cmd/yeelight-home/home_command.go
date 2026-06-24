package main

import (
	"context"
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/api"
)

func (app *app) runHome(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home home <list|select>")
		return exitInvalidInput
	}
	switch args[0] {
	case "list":
		return app.runHomeList(args[1:], stdout, stderr)
	case "select":
		return app.runHomeSelect(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported home command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runHomeList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home list: %v\n", err)
		return exitInvalidInput
	}
	contextInfo, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home list: %v\n", err)
		return exitInvalidInput
	}
	if contextInfo.AccessToken == "" {
		_, _ = fmt.Fprintln(stderr, "home list: missing token; run auth login --qr or auth token set")
		return exitInvalidInput
	}
	summary, err := api.NewHomeSummaryClient(contextInfo.Endpoint, nil).RunList(context.Background(), api.HomeSummaryCredentials{
		Authorization: contextInfo.AccessToken,
		ClientID:      contextInfo.ClientID,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home list: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{"ok": true, "profile": contextInfo.Profile, "region": contextInfo.Region, "houses": summary.Houses, "houseCount": summary.HouseCount})
	}
	for _, house := range summary.Houses {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\n", house.ID, house.Name)
	}
	return exitOK
}

func (app *app) runHomeSelect(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInvalidInput
	}
	houseID := flags.string("house-id", flags.string("id", ""))
	if houseID == "" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home home select --house-id <id> [--profile <name>] [--region <region>] [--json]")
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInternalError
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		"region":  flags.string("region", ""),
		"houseId": houseID,
	})
	if metadata.Region == "" {
		metadata.Region = "dev"
	}
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInternalError
	}
	result := map[string]any{"ok": true, "profile": metadata.Profile, "region": metadata.Region, "houseId": metadata.HouseID}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "selected houseId=%s for profile=%s\n", metadata.HouseID, metadata.Profile)
	return exitOK
}
