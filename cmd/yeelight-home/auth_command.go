package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/auth"
	"github.com/yeelight/yeelight-home/internal/credential"
	localoutput "github.com/yeelight/yeelight-home/internal/output"
)

func (app *app) runAuth(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth <status|login>")
		return exitInvalidInput
	}
	switch args[0] {
	case "qr-check":
		return app.runAuthQRCheck(args[1:], stdout, stderr)
	case "status":
		flags, err := parseFlags(args[1:])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth status: %v\n", err)
			return exitInvalidInput
		}
		if err := requireJSONFlag(flags, "usage: yeelight-home auth status --json [--profile <name>]"); err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return exitInvalidInput
		}
		return writeJSON(stdout, stderr, app.authStatus(flags))
	case "login":
		return app.runAuthLogin(args[1:], stdout, stderr)
	case "token":
		return app.runAuthToken(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported auth command %q\n", args[0])
		return exitInvalidInput
	}
}

func (app *app) runAuthQRCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth qr-check: %v\n", err)
		return exitInvalidInput
	}
	qrCodeID := flags.string("qr-code-id", "")
	if qrCodeID == "" {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth qr-check --qr-code-id <id> --json")
		return exitInvalidInput
	}
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth qr-check: %v\n", err)
		return exitInvalidInput
	}
	client := app.qrClient
	if client == nil {
		client = auth.NewQRLoginClient(endpoint.AccountBaseURL(), &http.Client{Timeout: 15 * time.Second})
	}
	info, err := client.Check(context.Background(), qrCodeID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth qr-check: %v\n", err)
		return exitInternalError
	}
	response := map[string]any{
		"ok":     true,
		"status": info.Status,
	}
	if auth.IsQRLoginStatus(info.Status) {
		credentials := auth.ExtractQRLoginCredentials(info)
		if credentials.Authorization == "" {
			_, _ = fmt.Fprintln(stderr, "auth qr-check: QR login response did not contain access token")
			return exitInternalError
		}
		profile := flags.string("profile", profileFromEnv())
		device, err := app.resolveQRDevice(profile, flags.string("device", ""))
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth qr-check: %v\n", err)
			return exitInvalidInput
		}
		if err := app.tokenStore.Save(credential.TokenRecord{Profile: profile, AccessToken: credentials.Authorization}); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth qr-check: save credential: %v\n", err)
			return exitInternalError
		}
		if err := app.metadataStore.Save(credential.ProfileMetadata{
			Profile:  profile,
			Region:   endpoint.Region,
			ClientID: credentials.ClientID,
			HouseID:  credentials.HouseID,
			QRDevice: device,
		}); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth qr-check: save profile metadata: %v\n", err)
			return exitInternalError
		}
		response["credentials"] = map[string]any{
			"accessTokenPresent": true,
			"clientId":           credentials.ClientID,
			"houseId":            credentials.HouseID,
		}
	}
	return writeJSON(stdout, stderr, response)
}

func (app *app) authStatus(flags cliFlags) map[string]any {
	status := auth.StatusFromEnv()
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return map[string]any{
			"authenticated": false,
			"error":         err.Error(),
			"profile":       flags.string("profile", status.Profile),
			"tokenPresent":  false,
			"tokenStore":    status.TokenStore,
		}
	}
	response := map[string]any{
		"authenticated": context.TokenPresent || status.Authenticated,
		"profile":       context.Profile,
		"region":        context.Region,
		"clientId":      context.ClientID,
		"houseId":       context.HouseID,
		"tokenPresent":  context.TokenPresent,
		"tokenSource":   context.TokenSource,
		"tokenStore":    status.TokenStore,
	}
	return response
}

func (app *app) runAuthLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("qr") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth login --qr [--json] [--region dev]")
		return exitInvalidInput
	}
	asJSON := flags.bool("json")
	profile := flags.string("profile", profileFromEnv())
	qrPNGPath := flags.string("qr-png", "")
	endpoint, err := resolveEndpointForFlags(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInvalidInput
	}
	client := app.qrClient
	if client == nil {
		client = auth.NewQRLoginClient(endpoint.AccountBaseURL(), &http.Client{Timeout: 15 * time.Second})
	}
	device, err := app.resolveQRDevice(profile, flags.string("device", ""))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInvalidInput
	}
	var printedPrompt bool
	result, err := auth.RunQRLoginFlow(context.Background(), auth.QRLoginOptions{
		Client:       client,
		Device:       device,
		HouseID:      flags.string("house-id", ""),
		NoWait:       flags.bool("no-wait"),
		PollInterval: time.Duration(flags.int("poll-interval-ms", auth.DefaultQRLoginPollIntervalMS)) * time.Millisecond,
		Timeout:      time.Duration(flags.int("timeout-ms", auth.DefaultQRLoginTimeoutMS)) * time.Millisecond,
		Sleep:        app.sleep,
		OnCreated: func(created auth.QRLoginResult) {
			if qrPNGPath != "" {
				_ = writeQRPNG(qrPNGPath, created.Payload)
			}
			if !asJSON {
				printQRLoginPrompt(stdout, created)
				printedPrompt = true
			}
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInternalError
	}
	if result.Credentials != nil {
		if err := app.tokenStore.Save(credential.TokenRecord{Profile: profile, AccessToken: result.Credentials.Authorization}); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth login: save credential: %v\n", err)
			return exitInternalError
		}
		if err := app.metadataStore.Save(credential.ProfileMetadata{
			Profile:  profile,
			Region:   endpoint.Region,
			ClientID: result.Credentials.ClientID,
			HouseID:  result.Credentials.HouseID,
			QRDevice: result.Device,
		}); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth login: save profile metadata: %v\n", err)
			return exitInternalError
		}
	}
	response := sanitizeQRLoginResult(profile, endpoint.Region, result)
	if qrPNGPath != "" {
		if err := writeQRPNG(qrPNGPath, result.Payload); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth login: write QR png: %v\n", err)
			return exitInternalError
		}
		response["qrPng"] = qrPNGPath
	}
	if asJSON {
		return writeJSON(stdout, stderr, response)
	}
	if !printedPrompt {
		printQRLoginPrompt(stdout, result)
	}
	if result.Credentials != nil {
		_, _ = fmt.Fprintf(stdout, "已保存凭据 profile=%s region=%s\n", profile, endpoint.Region)
	}
	return exitOK
}

func (app *app) runAuthToken(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth token <set|delete>")
		return exitInvalidInput
	}
	switch args[0] {
	case "set":
		flags, err := parseFlags(args[1:])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: %v\n", err)
			return exitInvalidInput
		}
		profile := flags.string("profile", profileFromEnv())
		token := strings.TrimSpace(flags.string("token", ""))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("YEELIGHT_HOME_ACCESS_TOKEN"))
		}
		if token == "" {
			_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth token set --token <access-token> [--profile <name>] [--region <region>] [--client-id <id>] [--house-id <id>] [--json]")
			return exitInvalidInput
		}
		if err := app.tokenStore.Save(credential.TokenRecord{Profile: profile, AccessToken: token}); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: save credential: %v\n", err)
			return exitInternalError
		}
		metadata, _, err := app.metadataStore.Load(profile)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: load profile metadata: %v\n", err)
			return exitInternalError
		}
		metadata = mergeProfileMetadata(metadata, profile, map[string]string{
			"region":   flags.string("region", ""),
			"clientId": flags.string("client-id", ""),
			"houseId":  flags.string("house-id", ""),
		})
		if metadata.Region == "" {
			metadata.Region = "dev"
		}
		if err := app.metadataStore.Save(metadata); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: save profile metadata: %v\n", err)
			return exitInternalError
		}
		result := map[string]any{
			"ok":           true,
			"profile":      profile,
			"tokenPresent": true,
			"region":       metadata.Region,
			"clientId":     metadata.ClientID,
			"houseId":      metadata.HouseID,
		}
		if flags.bool("json") {
			return writeJSON(stdout, stderr, result)
		}
		_, _ = fmt.Fprintf(stdout, "saved token for profile=%s\n", profile)
		return exitOK
	case "delete":
		flags, err := parseFlags(args[1:])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token delete: %v\n", err)
			return exitInvalidInput
		}
		profile := flags.string("profile", profileFromEnv())
		if err := app.tokenStore.Delete(profile); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token delete: %v\n", err)
			return exitInternalError
		}
		if flags.bool("json") {
			return writeJSON(stdout, stderr, map[string]any{"ok": true, "profile": profile, "tokenPresent": false})
		}
		_, _ = fmt.Fprintf(stdout, "deleted token for profile=%s\n", profile)
		return exitOK
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported auth token command %q\n", args[0])
		return exitInvalidInput
	}
}

func writeQRPNG(path string, payload string) error {
	data, err := localoutput.RenderQRPNG(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (app *app) resolveQRDevice(profile string, explicitDevice string) (string, error) {
	if explicitDevice != "" {
		device := auth.NormalizeDeviceMAC(explicitDevice)
		if device == "" {
			return "", fmt.Errorf("device is invalid")
		}
		if err := app.saveQRDevice(profile, device); err != nil {
			return "", err
		}
		return device, nil
	}
	if metadata, ok, err := app.metadataStore.Load(profile); err != nil {
		return "", err
	} else if ok && metadata.QRDevice != "" {
		return auth.NormalizeDeviceMAC(metadata.QRDevice), nil
	}
	device := auth.GenerateQRLoginDevice()
	if err := app.saveQRDevice(profile, device); err != nil {
		return "", err
	}
	return device, nil
}

func (app *app) saveQRDevice(profile string, device string) error {
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		return err
	}
	metadata.Profile = profile
	metadata.QRDevice = auth.NormalizeDeviceMAC(device)
	return app.metadataStore.Save(metadata)
}

func printQRLoginPrompt(stdout io.Writer, result auth.QRLoginResult) {
	_, _ = fmt.Fprintln(stdout, "请使用 Yeelight / 易来 APP 扫描下面的授权内容，并在手机上确认。")
	if rendered, err := localoutput.RenderQRText(result.Payload); err == nil {
		_, _ = fmt.Fprintln(stdout, rendered)
	}
	_, _ = fmt.Fprintf(stdout, "Payload: %s\n", result.Payload)
	if result.QRCodeID != "" {
		_, _ = fmt.Fprintf(stdout, "二维码 ID: %s\n", result.QRCodeID)
	}
}

func sanitizeQRLoginResult(profile string, region string, result auth.QRLoginResult) map[string]any {
	response := map[string]any{
		"ok":       result.OK,
		"status":   result.Status,
		"profile":  profile,
		"region":   region,
		"qrCodeId": result.QRCodeID,
		"device":   result.Device,
		"payload":  result.Payload,
	}
	if result.ExpireAt != 0 {
		response["expireAt"] = result.ExpireAt
	}
	if result.Credentials == nil {
		response["credentials"] = nil
		return response
	}
	response["credentials"] = map[string]any{
		"accessTokenPresent": true,
		"clientId":           result.Credentials.ClientID,
		"houseId":            result.Credentials.HouseID,
	}
	return response
}
