package main

import (
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/credential"
)

func (app *app) runProfile(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home profile <list|show|use|delete>")
		return exitInvalidInput
	}
	switch args[0] {
	case "list":
		return app.runProfileList(args[1:], stdout, stderr)
	case "show":
		return app.runProfileShow(args[1:], stdout, stderr)
	case "use":
		return app.runProfileUse(args[1:], stdout, stderr)
	case "delete":
		return app.runProfileDelete(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported profile command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runProfileList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile list: %v\n", err)
		return exitInvalidInput
	}
	profiles, err := app.metadataStore.List()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile list: %v\n", err)
		return exitInternalError
	}
	active, err := app.resolveProfile(cliFlags{values: map[string]string{}})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile list: %v\n", err)
		return exitInternalError
	}
	items := make([]any, 0, len(profiles))
	for _, profile := range profiles {
		_, tokenOK, _ := app.tokenStore.Load(profile.Profile)
		items = append(items, map[string]any{
			"profile":      profile.Profile,
			"active":       profile.Profile == active,
			"region":       profile.Region,
			"houseId":      profile.HouseID,
			"tokenPresent": tokenOK,
		})
	}
	result := map[string]any{"ok": true, "activeProfile": active, "profiles": items}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	for _, item := range items {
		row := item.(map[string]any)
		marker := " "
		if row["active"] == true {
			marker = "*"
		}
		_, _ = fmt.Fprintf(stdout, "%s %s region=%s houseId=%s token=%v\n", marker, row["profile"], row["region"], row["houseId"], row["tokenPresent"])
	}
	return exitOK
}

func (app *app) runProfileShow(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile show: %v\n", err)
		return exitInvalidInput
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile show: %v\n", err)
		return exitInvalidInput
	}
	result := map[string]any{
		"ok":           true,
		"profile":      context.Profile,
		"region":       context.Region,
		"houseId":      context.HouseID,
		"tokenPresent": context.TokenPresent,
		"tokenSource":  context.TokenSource,
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "profile=%s region=%s houseId=%s token=%v\n", context.Profile, context.Region, context.HouseID, context.TokenPresent)
	return exitOK
}

func (app *app) runProfileUse(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInvalidInput
	}
	profile := flags.string("profile", "")
	if profile == "" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home profile use --profile <name> [--region <region>] [--house-id <id>] [--json]")
		return exitInvalidInput
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInternalError
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		"region":  flags.string("region", ""),
		"houseId": flags.string("house-id", ""),
	})
	if metadata.Region == "" {
		metadata.Region = defaultRuntimeRegion
	}
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInternalError
	}
	if err := app.metadataStore.SetActiveProfile(profile); err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInternalError
	}
	result := map[string]any{"ok": true, "profile": metadata.Profile, "region": metadata.Region, "houseId": metadata.HouseID}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "saved profile=%s\n", metadata.Profile)
	return exitOK
}

func (app *app) runProfileDelete(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile delete: %v\n", err)
		return exitInvalidInput
	}
	profile := flags.string("profile", "")
	if profile == "" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home profile delete --profile <name> [--json]")
		return exitInvalidInput
	}
	if err := app.metadataStore.Delete(profile); err != nil {
		_, _ = fmt.Fprintf(stderr, "profile delete: %v\n", err)
		return exitInternalError
	}
	if err := app.tokenStore.Delete(profile); err != nil {
		_, _ = fmt.Fprintf(stderr, "profile delete: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{"ok": true, "profile": profile})
	}
	_, _ = fmt.Fprintf(stdout, "deleted profile=%s\n", profile)
	return exitOK
}

func profileMetadataMap(metadata credential.ProfileMetadata) map[string]any {
	return map[string]any{
		"profile":  metadata.Profile,
		"region":   metadata.Region,
		"houseId":  metadata.HouseID,
		"qrDevice": metadata.QRDevice,
	}
}
