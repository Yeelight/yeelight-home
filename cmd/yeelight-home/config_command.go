package main

import (
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
		semantic.FieldOK: true,
		semantic.FieldPrecedence: []string{
			"command flags",
			"environment variables",
			"profile metadata and credential store",
			"defaults",
		},
		semantic.FieldProfile:      context.Profile,
		semantic.FieldRegion:       context.Region,
		semantic.FieldHouseID:      context.HouseID,
		semantic.FieldLanguage:     context.Language,
		semantic.FieldControlMode:  context.ControlMode,
		semantic.FieldGatewayIP:    context.GatewayIP,
		semantic.FieldLANEndpoint:  context.LANEndpoint,
		semantic.FieldTokenPresent: context.TokenPresent,
		semantic.FieldTokenSource:  context.TokenSource,
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "profile=%s\nregion=%s\nhouseId=%s\nlanguage=%s\ncontrolMode=%s\ngatewayIp=%s\nlanEndpoint=%s\ntokenPresent=%v\n", context.Profile, context.Region, context.HouseID, context.Language, context.ControlMode, context.GatewayIP, context.LANEndpoint, context.TokenPresent)
	return exitOK
}

func (app *app) runConfigSet(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInvalidInput
	}
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInternalError
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInternalError
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		semantic.FieldRegion:      flags.string("region", ""),
		semantic.FieldHouseID:     flags.string("house-id", ""),
		semantic.FieldQRDevice:    flags.string("qr-device", ""),
		semantic.FieldLanguage:    flags.string("language", ""),
		semantic.FieldControlMode: flags.string("control-mode", ""),
		semantic.FieldGatewayIP:   flags.string("gateway-ip", ""),
		semantic.FieldLANEndpoint: flags.string("lan-endpoint", ""),
	})
	if flags.string("gateway-ip", "") != "" && flags.string("lan-endpoint", "") == "" {
		metadata.LANEndpoint = ""
	}
	if metadata.Language != "" {
		language, supported := i18n.Normalize(metadata.Language)
		if !supported {
			_, _ = fmt.Fprintln(stderr, "config set: language must be zh-CN or en-US")
			return exitInvalidInput
		}
		metadata.Language = language
	}
	metadata.ControlMode, err = normalizeControlMode(metadata.ControlMode)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInvalidInput
	}
	metadata.LANEndpoint, err = resolveLANEndpoint(metadata.GatewayIP, metadata.LANEndpoint)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInvalidInput
	}
	if metadata.ControlMode != controlModeCloud && metadata.LANEndpoint == "" {
		_, _ = fmt.Fprintf(stderr, "config set: control mode %s requires --gateway-ip or --lan-endpoint\n", metadata.ControlMode)
		return exitInvalidInput
	}
	if metadata.Region == "" {
		metadata.Region = defaultRuntimeRegion
	}
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "config set: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{semantic.FieldOK: true, semantic.FieldProfile: profileMetadataMap(metadata)})
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
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInternalError
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInternalError
	}
	if flags.bool("region") {
		metadata.Region = ""
	}
	if flags.bool("house-id") {
		metadata.HouseID = ""
	}
	if flags.bool("qr-device") {
		metadata.QRDevice = ""
	}
	if flags.bool("language") {
		metadata.Language = ""
	}
	if flags.bool("control-mode") {
		metadata.ControlMode = ""
	}
	if flags.bool("gateway-ip") {
		metadata.GatewayIP = ""
	}
	if flags.bool("lan-endpoint") {
		metadata.LANEndpoint = ""
	}
	if metadata.ControlMode != "" && metadata.ControlMode != controlModeCloud && metadata.GatewayIP == "" && metadata.LANEndpoint == "" {
		_, _ = fmt.Fprintln(stderr, "config unset: unset --control-mode with the LAN address, or keep a gateway address")
		return exitInvalidInput
	}
	metadata.Profile = profile
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "config unset: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, map[string]any{semantic.FieldOK: true, semantic.FieldProfile: profileMetadataMap(metadata)})
	}
	_, _ = fmt.Fprintf(stdout, "updated profile=%s\n", profile)
	return exitOK
}
