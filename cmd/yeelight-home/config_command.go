package main

import (
	"fmt"
	"io"
)

func (app *app) runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home config <get|set|list|unset>")
		return exitInvalidInput
	}
	switch args[0] {
	case "get", "list":
		return app.runConfigGet(args[1:], stdout, stderr)
	case "set":
		return app.runConfigSet(args[1:], stdout, stderr)
	case "unset":
		return app.runConfigUnset(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported config command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runConfigGet(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config get: %v\n", err)
		return exitInvalidInput
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config get: %v\n", err)
		return exitInvalidInput
	}
	result := map[string]any{
		"ok": true,
		"precedence": []string{
			"command flags",
			"environment variables",
			"profile metadata and credential store",
			"defaults",
		},
		"profile":      context.Profile,
		"region":       context.Region,
		"clientId":     context.ClientID,
		"houseId":      context.HouseID,
		"tokenPresent": context.TokenPresent,
		"tokenSource":  context.TokenSource,
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "profile=%s\nregion=%s\nclientId=%s\nhouseId=%s\ntokenPresent=%v\n", context.Profile, context.Region, context.ClientID, context.HouseID, context.TokenPresent)
	return exitOK
}

func (app *app) runConfigSet(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInternalError
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		"region":   flags.string("region", ""),
		"clientId": flags.string("client-id", ""),
		"houseId":  flags.string("house-id", ""),
		"qrDevice": flags.string("qr-device", ""),
	})
	if metadata.Region == "" {
		metadata.Region = "dev"
	}
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{"ok": true, "profile": profileMetadataMap(metadata)})
	}
	_, _ = fmt.Fprintf(stdout, "updated profile=%s\n", profile)
	return exitOK
}

func (app *app) runConfigUnset(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInternalError
	}
	if flags.bool("region") {
		metadata.Region = ""
	}
	if flags.bool("client-id") {
		metadata.ClientID = ""
	}
	if flags.bool("house-id") {
		metadata.HouseID = ""
	}
	if flags.bool("qr-device") {
		metadata.QRDevice = ""
	}
	metadata.Profile = profile
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{"ok": true, "profile": profileMetadataMap(metadata)})
	}
	_, _ = fmt.Fprintf(stdout, "updated profile=%s\n", profile)
	return exitOK
}
