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
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) runAuth(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth <status|login|token>")
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
		status := app.authStatus(flags)
		if flags.bool("json") {
			return writeJSON(stdout, stderr, status)
		}
		return writeAuthStatusText(stdout, status)
	case "login":
		return app.runAuthLogin(args[1:], stdout, stderr)
	case "token":
		return app.runAuthToken(args[1:], stdin, stdout, stderr)
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
		semantic.FieldOK:     true,
		semantic.FieldStatus: info.Status,
	}
	if auth.IsQRLoginStatus(info.Status) {
		credentials := auth.ExtractQRLoginCredentials(info)
		if credentials.Authorization == "" {
			_, _ = fmt.Fprintln(stderr, "auth qr-check: QR login response did not contain access token")
			return exitInternalError
		}
		profile, err := app.resolveTargetProfile(flags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth qr-check: %v\n", err)
			return exitInternalError
		}
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
		response[semantic.FieldCredentials] = map[string]any{
			semantic.FieldAccessTokenPresent: true,
			semantic.FieldHouseID:            credentials.HouseID,
		}
	}
	return writeJSON(stdout, stderr, response)
}

func (app *app) authStatus(flags cliFlags) map[string]any {
	status := auth.StatusFromEnv()
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return map[string]any{
			semantic.FieldAuthenticated: false,
			semantic.FieldError:         err.Error(),
			semantic.FieldProfile:       flags.string("profile", status.Profile),
			semantic.FieldTokenPresent:  false,
			semantic.FieldTokenStore:    status.TokenStore,
		}
	}
	response := map[string]any{
		semantic.FieldAuthenticated: context.TokenPresent || status.Authenticated,
		semantic.FieldProfile:       context.Profile,
		semantic.FieldRegion:        context.Region,
		semantic.FieldHouseID:       context.HouseID,
		semantic.FieldTokenPresent:  context.TokenPresent,
		semantic.FieldTokenSource:   context.TokenSource,
		semantic.FieldTokenStore:    status.TokenStore,
	}
	return response
}

func writeAuthStatusText(stdout io.Writer, response map[string]any) int {
	_, _ = fmt.Fprintln(stdout, "Yeelight Home Auth")
	_, _ = fmt.Fprintf(stdout, "Authenticated: %t\n", boolFromDiagnostic(response, semantic.FieldAuthenticated))
	_, _ = fmt.Fprintf(stdout, "Profile: %s\n", stringFromDiagnostic(response, semantic.FieldProfile))
	_, _ = fmt.Fprintf(stdout, "Region: %s\n", stringFromDiagnostic(response, semantic.FieldRegion))
	houseID := stringFromDiagnostic(response, semantic.FieldHouseID)
	if houseID == "" {
		houseID = "(not selected)"
	}
	_, _ = fmt.Fprintf(stdout, "House ID: %s\n", houseID)
	_, _ = fmt.Fprintf(stdout, "Token present: %t\n", boolFromDiagnostic(response, semantic.FieldTokenPresent))
	tokenSource := stringFromDiagnostic(response, semantic.FieldTokenSource)
	if tokenSource == "" {
		tokenSource = "(none)"
	}
	_, _ = fmt.Fprintf(stdout, "Token source: %s\n", tokenSource)
	if errText := stringFromDiagnostic(response, semantic.FieldError); errText != "" {
		_, _ = fmt.Fprintf(stdout, "Error: %s\n", errText)
	}
	return exitOK
}

func (app *app) runAuthLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInvalidInput
	}
	if !flags.bool("qr") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth login --qr [--json] [--region cn]")
		return exitInvalidInput
	}
	asJSON := flags.bool("json")
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "auth login: %v\n", err)
		return exitInternalError
	}
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
		response[semantic.FieldQRPng] = qrPNGPath
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

func (app *app) runAuthToken(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
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
		profile, err := app.resolveTargetProfile(flags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: %v\n", err)
			return exitInternalError
		}
		token := strings.TrimSpace(flags.string("token", ""))
		if token != "" && flags.bool("stdin") {
			_, _ = fmt.Fprintln(stderr, "auth token set: --token and --stdin are mutually exclusive")
			return exitInvalidInput
		}
		if token == "" && flags.bool("stdin") {
			data, err := io.ReadAll(io.LimitReader(stdin, 256*1024))
			if err != nil {
				_, _ = fmt.Fprintf(stderr, "auth token set: read stdin: %v\n", err)
				return exitInternalError
			}
			token = strings.TrimSpace(string(data))
		}
		if token == "" {
			token = strings.TrimSpace(os.Getenv("YEELIGHT_HOME_ACCESS_TOKEN"))
		}
		if token == "" {
			_, _ = fmt.Fprintln(stderr, "usage: yeelight-home auth token set (--token <access-token>|--stdin) [--profile <name>] [--region <region>] [--house-id <id>] [--json]")
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
			semantic.FieldRegion:  flags.string("region", ""),
			semantic.FieldHouseID: flags.string("house-id", ""),
		})
		if metadata.Region == "" {
			metadata.Region = defaultRuntimeRegion
		}
		if err := app.metadataStore.Save(metadata); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token set: save profile metadata: %v\n", err)
			return exitInternalError
		}
		result := map[string]any{
			semantic.FieldOK:           true,
			semantic.FieldProfile:      profile,
			semantic.FieldTokenPresent: true,
			semantic.FieldRegion:       metadata.Region,
			semantic.FieldHouseID:      metadata.HouseID,
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
		profile, err := app.resolveTargetProfile(flags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token delete: %v\n", err)
			return exitInternalError
		}
		if err := app.tokenStore.Delete(profile); err != nil {
			_, _ = fmt.Fprintf(stderr, "auth token delete: %v\n", err)
			return exitInternalError
		}
		if flags.bool("json") {
			return writeJSON(stdout, stderr, map[string]any{semantic.FieldOK: true, semantic.FieldProfile: profile, semantic.FieldTokenPresent: false})
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
		semantic.FieldOK:       result.OK,
		semantic.FieldStatus:   result.Status,
		semantic.FieldProfile:  profile,
		semantic.FieldRegion:   region,
		semantic.FieldQRCodeID: result.QRCodeID,
		semantic.FieldDevice:   result.Device,
		semantic.FieldPayload:  result.Payload,
	}
	if result.ExpireAt != 0 {
		response[semantic.FieldExpireAt] = result.ExpireAt
	}
	if result.Credentials == nil {
		response[semantic.FieldCredentials] = nil
		return response
	}
	response[semantic.FieldCredentials] = map[string]any{
		semantic.FieldAccessTokenPresent: true,
		semantic.FieldHouseID:            result.Credentials.HouseID,
	}
	return response
}
