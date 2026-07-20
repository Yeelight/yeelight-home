package main

import (
	"fmt"
	"io"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
			semantic.FieldProfile:      profile.Profile,
			semantic.FieldActive:       profile.Profile == active,
			semantic.FieldRegion:       profile.Region,
			semantic.FieldHouseID:      profile.HouseID,
			semantic.FieldBizType:      profile.BizType,
			semantic.FieldLanguage:     profile.Language,
			semantic.FieldControlMode:  profile.ControlMode,
			semantic.FieldTokenPresent: tokenOK,
		})
	}
	result := map[string]any{semantic.FieldOK: true, semantic.FieldActiveProfile: active, semantic.FieldProfiles: items}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	for _, item := range items {
		row := item.(map[string]any)
		marker := " "
		if row[semantic.FieldActive] == true {
			marker = "*"
		}
		_, _ = fmt.Fprintf(stdout, "%s %s region=%s bizType=%s houseId=%s token=%v\n", marker, row[semantic.FieldProfile], row[semantic.FieldRegion], row[semantic.FieldBizType], row[semantic.FieldHouseID], row[semantic.FieldTokenPresent])
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
		semantic.FieldOK:           true,
		semantic.FieldProfile:      context.Profile,
		semantic.FieldRegion:       context.Region,
		semantic.FieldHouseID:      context.HouseID,
		semantic.FieldBizType:      context.BizType,
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
	_, _ = fmt.Fprintf(stdout, "profile=%s region=%s bizType=%s houseId=%s language=%s controlMode=%s gatewayIp=%s token=%v\n", context.Profile, context.Region, context.BizType, context.HouseID, context.Language, context.ControlMode, context.GatewayIP, context.TokenPresent)
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
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home profile use --profile <name> [--region <region>] [--biz-type <0|1>] [--house-id <id>] [--json]")
		return exitInvalidInput
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInternalError
	}
	bizType, err := resolveBizType(flags, metadata.BizType)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "profile use: %v\n", err)
		return exitInvalidInput
	}
	clearHouseIDForBizTypeChange(&metadata, flags, bizType)
	houseID := flags.string("house-id", "")
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		semantic.FieldRegion:  flags.string("region", ""),
		semantic.FieldHouseID: houseID,
		semantic.FieldBizType: bizType,
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
	result := map[string]any{semantic.FieldOK: true, semantic.FieldProfile: metadata.Profile, semantic.FieldRegion: metadata.Region, semantic.FieldHouseID: metadata.HouseID, semantic.FieldBizType: metadata.BizType}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	_, _ = fmt.Fprintf(stdout, "saved profile=%s\n", metadata.Profile)
	return exitOK
}

func clearHouseIDForBizTypeChange(metadata *credential.ProfileMetadata, flags cliFlags, bizType string) {
	previousBizType := metadata.BizType
	if previousBizType == "" {
		previousBizType = api.BizTypeConsumer
	}
	if _, explicitHouseID := flags.values["house-id"]; !explicitHouseID && bizType != previousBizType {
		metadata.HouseID = ""
	}
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
		return writeJSON(stdout, stderr, map[string]any{semantic.FieldOK: true, semantic.FieldProfile: profile})
	}
	_, _ = fmt.Fprintf(stdout, "deleted profile=%s\n", profile)
	return exitOK
}

func profileMetadataMap(metadata credential.ProfileMetadata) map[string]any {
	return map[string]any{
		semantic.FieldProfile:     metadata.Profile,
		semantic.FieldRegion:      metadata.Region,
		semantic.FieldHouseID:     metadata.HouseID,
		semantic.FieldBizType:     metadata.BizType,
		semantic.FieldQRDevice:    metadata.QRDevice,
		semantic.FieldLanguage:    metadata.Language,
		semantic.FieldControlMode: metadata.ControlMode,
		semantic.FieldGatewayIP:   metadata.GatewayIP,
		semantic.FieldLANEndpoint: metadata.LANEndpoint,
	}
}
