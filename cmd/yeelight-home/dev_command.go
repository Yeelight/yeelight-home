package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
)

func (app *app) runDev(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home dev <seed-house|seed-room|seed-scene|seed-automation>")
		return exitInvalidInput
	}
	switch args[0] {
	case "seed-house":
		return app.runDevSeedHouse(args[1:], stdout, stderr)
	case "seed-room":
		return app.runDevSeedRoom(args[1:], stdout, stderr)
	case "seed-scene":
		return app.runDevSeedScene(args[1:], stdout, stderr)
	case "seed-automation":
		return app.runDevSeedAutomation(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported dev command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runDevSeedAutomation(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-automation: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home dev seed-automation --json --region dev --house-id <id> --device-id <id> --name <name> --allow-write-dev")
		return exitInvalidInput
	}
	if !flags.bool("allow-write-dev") {
		_, _ = fmt.Fprintln(stderr, "dev seed-automation requires --allow-write-dev")
		return exitInvalidInput
	}
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-automation: %v\n", err)
		return exitInvalidInput
	}
	if endpoint.Region != "dev" {
		_, _ = fmt.Fprintln(stderr, "dev seed-automation is only allowed for dev region")
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, credentials, err := app.loadDevSeedProfile(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-automation: %v\n", err)
		return exitInvalidInput
	}
	houseID := flags.string("house-id", metadata.HouseID)
	if strings.TrimSpace(houseID) == "" {
		_, _ = fmt.Fprintln(stderr, "dev seed-automation: house id is required; run dev seed-house first or pass --house-id")
		return exitInvalidInput
	}
	deviceID := flags.string("device-id", "")
	if strings.TrimSpace(deviceID) == "" {
		_, _ = fmt.Fprintln(stderr, "dev seed-automation: device id is required")
		return exitInvalidInput
	}
	result, err := api.NewDevSeedClient(endpoint, nil).EnsureAutomation(context.Background(), api.DevSeedAutomationRequest{
		HouseID:        houseID,
		Name:           flags.string("name", "Codex Dev Test Automation"),
		DeviceID:       deviceID,
		DeviceName:     flags.string("device-name", ""),
		PropertyName:   flags.string("property", "p"),
		PropertyValue:  flags.bool("value"),
		AllowWriteDev:  true,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials:    credentials,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-automation: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}

func (app *app) runDevSeedHouse(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-house: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home dev seed-house --json --region dev --name <name> --allow-write-dev")
		return exitInvalidInput
	}
	if !flags.bool("allow-write-dev") {
		_, _ = fmt.Fprintln(stderr, "dev seed-house requires --allow-write-dev")
		return exitInvalidInput
	}
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-house: %v\n", err)
		return exitInvalidInput
	}
	if endpoint.Region != "dev" {
		_, _ = fmt.Fprintln(stderr, "dev seed-house is only allowed for dev region")
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, credentials, err := app.loadDevSeedProfile(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-house: %v\n", err)
		return exitInvalidInput
	}
	result, err := api.NewDevSeedClient(endpoint, nil).EnsureHouse(context.Background(), api.DevSeedHouseRequest{
		Name:             flags.string("name", "Codex Dev Test Home"),
		Description:      flags.string("description", "Runtime dev verification fixture"),
		AreaCode:         flags.string("area-code", ""),
		AreaName:         flags.string("area-name", ""),
		CandidateHouseID: metadata.HouseID,
		AllowWriteDev:    true,
		VerifyAttempts:   5,
		VerifyInterval:   time.Second,
		Credentials:      credentials,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-house: %v\n", err)
		return exitInternalError
	}
	metadata.Profile = profile
	metadata.Region = "dev"
	metadata.HouseID = result.HouseID
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-house: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}

func (app *app) runDevSeedRoom(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-room: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home dev seed-room --json --region dev --name <name> --allow-write-dev")
		return exitInvalidInput
	}
	if !flags.bool("allow-write-dev") {
		_, _ = fmt.Fprintln(stderr, "dev seed-room requires --allow-write-dev")
		return exitInvalidInput
	}
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-room: %v\n", err)
		return exitInvalidInput
	}
	if endpoint.Region != "dev" {
		_, _ = fmt.Fprintln(stderr, "dev seed-room is only allowed for dev region")
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, credentials, err := app.loadDevSeedProfile(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-room: %v\n", err)
		return exitInvalidInput
	}
	houseID := flags.string("house-id", metadata.HouseID)
	if strings.TrimSpace(houseID) == "" {
		_, _ = fmt.Fprintln(stderr, "dev seed-room: house id is required; run dev seed-house first or pass --house-id")
		return exitInvalidInput
	}
	result, err := api.NewDevSeedClient(endpoint, nil).EnsureRoom(context.Background(), api.DevSeedRoomRequest{
		HouseID:        houseID,
		Name:           flags.string("name", "Codex Dev Test Room"),
		Description:    flags.string("description", "Runtime dev verification room"),
		Icon:           flags.string("icon", ""),
		AllowWriteDev:  true,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials:    credentials,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-room: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}

func (app *app) runDevSeedScene(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-scene: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("json") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home dev seed-scene --json --region dev --house-id <id> --device-id <id> --name <name> --allow-write-dev")
		return exitInvalidInput
	}
	if !flags.bool("allow-write-dev") {
		_, _ = fmt.Fprintln(stderr, "dev seed-scene requires --allow-write-dev")
		return exitInvalidInput
	}
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-scene: %v\n", err)
		return exitInvalidInput
	}
	if endpoint.Region != "dev" {
		_, _ = fmt.Fprintln(stderr, "dev seed-scene is only allowed for dev region")
		return exitInvalidInput
	}
	profile := flags.string("profile", profileFromEnv())
	metadata, credentials, err := app.loadDevSeedProfile(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-scene: %v\n", err)
		return exitInvalidInput
	}
	houseID := flags.string("house-id", metadata.HouseID)
	if strings.TrimSpace(houseID) == "" {
		_, _ = fmt.Fprintln(stderr, "dev seed-scene: house id is required; run dev seed-house first or pass --house-id")
		return exitInvalidInput
	}
	deviceID := flags.string("device-id", "")
	if strings.TrimSpace(deviceID) == "" {
		_, _ = fmt.Fprintln(stderr, "dev seed-scene: device id is required")
		return exitInvalidInput
	}
	result, err := api.NewDevSeedClient(endpoint, nil).EnsureScene(context.Background(), api.DevSeedSceneRequest{
		HouseID:        houseID,
		Name:           flags.string("name", "Codex Dev Test Scene"),
		Description:    flags.string("description", "Runtime dev verification scene"),
		Icon:           flags.string("icon", ""),
		DeviceID:       deviceID,
		DeviceName:     flags.string("device-name", ""),
		PropertyName:   flags.string("property", "p"),
		PropertyValue:  flags.bool("value"),
		AllowWriteDev:  true,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials:    credentials,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "dev seed-scene: %v\n", err)
		return exitInternalError
	}
	return writeJSON(stdout, stderr, result)
}

func (app *app) loadDevSeedProfile(profile string) (credential.ProfileMetadata, api.DevSeedCredentials, error) {
	metadata, ok, err := app.metadataStore.Load(profile)
	if err != nil {
		return credential.ProfileMetadata{}, api.DevSeedCredentials{}, err
	}
	if !ok {
		metadata = credential.ProfileMetadata{Profile: profile, Region: "dev"}
	}
	credentials := api.DevSeedCredentials{
		Authorization: strings.TrimSpace(os.Getenv("YEELIGHT_HOME_ACCESS_TOKEN")),
		ClientID:      strings.TrimSpace(os.Getenv("YEELIGHT_HOME_CLIENT_ID")),
	}
	if credentials.ClientID == "" {
		credentials.ClientID = metadata.ClientID
	}
	if credentials.Authorization == "" {
		record, ok, err := app.tokenStore.Load(profile)
		if err != nil {
			return credential.ProfileMetadata{}, api.DevSeedCredentials{}, err
		}
		if ok {
			credentials.Authorization = record.AccessToken
		}
	}
	return metadata, credentials, nil
}
