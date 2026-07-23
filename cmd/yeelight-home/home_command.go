package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) runHome(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home home <list|select>")
		return exitInvalidInput
	}
	switch args[0] {
	case "list":
		return app.runHomeList(args[1:], stdout, stderr)
	case "select":
		return app.runHomeSelect(args[1:], stdin, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unsupported home command %q\n", args[0])
		return exitInvalidInput
	}
}

func isNativeHomeCommand(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "list", "select":
		return true
	default:
		return false
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
	ctx := api.WithBizType(context.Background(), contextInfo.BizType)
	summary, err := api.NewHomeSummaryClient(contextInfo.Endpoint, nil).RunList(ctx, api.HomeSummaryCredentials{
		Authorization: contextInfo.AccessToken,
		ClientID:      contextInfo.ClientID,
		BizType:       contextInfo.BizType,
	})
	if err != nil {
		var statusErr api.HTTPStatusError
		if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusUnauthorized || statusErr.StatusCode == http.StatusForbidden) {
			_, _ = fmt.Fprintln(stderr, "home list: authorization failed; token is missing, invalid, expired, or not accepted by this region. Run yeelight-home auth login --qr --region "+contextInfo.Region+" or set a valid YEELIGHT_HOME_ACCESS_TOKEN.")
			return exitInvalidInput
		}
		_, _ = fmt.Fprintf(stderr, "home list: %v\n", err)
		return exitInternalError
	}
	if flags.bool("json") {
		response := map[string]any{
			semantic.FieldOK:         true,
			semantic.FieldProfile:    contextInfo.Profile,
			semantic.FieldRegion:     contextInfo.Region,
			semantic.FieldBizType:    contextInfo.BizType,
			semantic.FieldHouses:     summary.Houses,
			semantic.FieldHouseCount: summary.HouseCount,
			semantic.FieldRawShape:   summary.RawShape,
			semantic.FieldAPICalls:   summary.APICalls,
			semantic.FieldSource:     summary.Source,
			semantic.FieldHouseID:    "",
		}
		if contextInfo.HouseID != "" {
			response[semantic.FieldSelectedHouseID] = contextInfo.HouseID
		}
		if summary.HouseCount == 0 {
			response[semantic.FieldWarnings] = []string{"empty_account_home_list"}
			response[semantic.FieldNext] = []string{
				"home list is account-scoped and does not require houseId",
				"verify the active profile and region with yeelight-home auth status --json",
				"if you already know a house id, run yeelight-home home select --house-id <id> --region " + contextInfo.Region,
			}
		}
		return writeJSON(stdout, stderr, response)
	}
	for _, house := range summary.Houses {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\n", house.ID, house.Name)
	}
	return exitOK
}

func (app *app) runHomeSelect(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInvalidInput
	}
	houseID := flags.string("house-id", flags.string("id", ""))
	profile, err := app.resolveTargetProfile(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInternalError
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInternalError
	}
	locale, err := resolveInteractiveLocale(flags, metadata.Language)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInvalidInput
	}
	bizType, err := resolveBizType(flags, metadata.BizType)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInvalidInput
	}
	selectedName := ""
	if houseID == "" {
		if !app.isTerminal(stdin) {
			_, _ = fmt.Fprintln(stderr, "home select: run this command in an interactive terminal to choose by name, or provide --house-id <id> for automation")
			return exitInvalidInput
		}
		listArgs := []string{"--json"}
		if profile != "" {
			listArgs = append(listArgs, "--profile", profile)
		}
		if region := flags.string("region", ""); region != "" {
			listArgs = append(listArgs, "--region", region)
		}
		if bizType != "" {
			listArgs = append(listArgs, "--biz-type", bizType)
		}
		var listOutput bytes.Buffer
		var listError bytes.Buffer
		if code := app.runHomeList(listArgs, &listOutput, &listError); code != exitOK {
			_, _ = io.Copy(stderr, &listError)
			return code
		}
		var result struct {
			Houses []setupHomeChoice `json:"houses"`
		}
		if err := json.Unmarshal(listOutput.Bytes(), &result); err != nil {
			_, _ = fmt.Fprintf(stderr, "home select: parse home list: %v\n", err)
			return exitInternalError
		}
		prompt := newSetupPrompt(stdin, stdout, isTerminalWriter(stdout) && !flags.bool("json"))
		houseID, err = prompt.chooseHome(locale, result.Houses)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
			return exitInvalidInput
		}
		for _, home := range result.Houses {
			if home.ID == houseID {
				selectedName = home.Name
				break
			}
		}
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		semantic.FieldRegion:   flags.string("region", ""),
		semantic.FieldHouseID:  houseID,
		semantic.FieldBizType:  bizType,
		semantic.FieldLanguage: locale,
	})
	if metadata.Region == "" {
		metadata.Region = defaultRuntimeRegion
	}
	if err := app.metadataStore.Save(metadata); err != nil {
		_, _ = fmt.Fprintf(stderr, "home select: %v\n", err)
		return exitInternalError
	}
	result := map[string]any{semantic.FieldOK: true, semantic.FieldProfile: metadata.Profile, semantic.FieldRegion: metadata.Region, semantic.FieldHouseID: metadata.HouseID, semantic.FieldBizType: metadata.BizType}
	if flags.bool("json") {
		return writeJSON(stdout, stderr, result)
	}
	if strings.TrimSpace(selectedName) != "" {
		_, _ = fmt.Fprintf(stdout, "selected home=%s for profile=%s\n", selectedName, metadata.Profile)
	} else {
		_, _ = fmt.Fprintf(stdout, "selected houseId=%s for profile=%s\n", metadata.HouseID, metadata.Profile)
	}
	return exitOK
}

func resolveInteractiveLocale(flags cliFlags, stored string) (string, error) {
	if value := flags.string("lang", flags.string("language", "")); value != "" {
		locale, ok := i18n.Normalize(value)
		if !ok {
			return "", fmt.Errorf("language must be zh-CN or en-US")
		}
		return locale, nil
	}
	if locale, ok := i18n.Normalize(stored); ok {
		return locale, nil
	}
	if locale, ok := i18n.Detect(os.LookupEnv); ok {
		return locale, nil
	}
	return i18n.Chinese, nil
}
