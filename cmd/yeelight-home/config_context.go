package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
)

type runtimeContext struct {
	Profile      string
	Region       string
	Endpoint     api.Endpoint
	ClientID     string
	HouseID      string
	AccessToken  string
	TokenSource  string
	Metadata     credential.ProfileMetadata
	MetadataOK   bool
	TokenPresent bool
}

func (app *app) resolveRuntimeContext(flags cliFlags) (runtimeContext, error) {
	profile, err := app.resolveProfile(flags)
	if err != nil {
		return runtimeContext{}, err
	}
	metadata, metadataOK, err := app.metadataStore.Load(profile)
	if err != nil {
		return runtimeContext{}, err
	}
	region := firstNonEmpty(
		flags.string("region", ""),
		strings.TrimSpace(os.Getenv("YEELIGHT_CLOUD_REGION")),
		metadata.Region,
		"dev",
	)
	endpoint, err := resolveEndpoint(region)
	if err != nil {
		return runtimeContext{}, err
	}
	clientID := firstNonEmpty(flags.string("client-id", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_CLIENT_ID")), metadata.ClientID)
	houseID := firstNonEmpty(flags.string("house-id", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_HOUSE_ID")), metadata.HouseID)
	accessToken := strings.TrimSpace(os.Getenv("YEELIGHT_HOME_ACCESS_TOKEN"))
	tokenSource := ""
	if accessToken != "" {
		tokenSource = "env"
	} else {
		record, ok, err := app.tokenStore.Load(profile)
		if err != nil {
			return runtimeContext{}, err
		}
		if ok {
			accessToken = record.AccessToken
			tokenSource = "store"
		}
	}
	return runtimeContext{
		Profile:      profile,
		Region:       endpoint.Region,
		Endpoint:     endpoint,
		ClientID:     clientID,
		HouseID:      houseID,
		AccessToken:  accessToken,
		TokenSource:  tokenSource,
		Metadata:     metadata,
		MetadataOK:   metadataOK,
		TokenPresent: strings.TrimSpace(accessToken) != "",
	}, nil
}

func resolveEndpoint(region string) (api.Endpoint, error) {
	if baseURL := strings.TrimSpace(os.Getenv("YEELIGHT_API_BASE_URL")); baseURL != "" {
		resolved, err := api.ResolveEndpoint(region)
		if err != nil {
			return api.Endpoint{}, err
		}
		return api.Endpoint{Region: resolved.Region, BaseURL: strings.TrimRight(baseURL, "/")}, nil
	}
	return api.ResolveEndpoint(region)
}

func resolveEndpointForFlags(flags cliFlags) (api.Endpoint, error) {
	region := firstNonEmpty(flags.string("region", ""), strings.TrimSpace(os.Getenv("YEELIGHT_CLOUD_REGION")), "dev")
	return resolveEndpoint(region)
}

func profileFromEnv() string {
	return envOrDefault("YEELIGHT_HOME_PROFILE", "default")
}

func (app *app) resolveProfile(flags cliFlags) (string, error) {
	if value := flags.string("profile", ""); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(os.Getenv("YEELIGHT_HOME_PROFILE")); value != "" {
		return value, nil
	}
	active, err := app.metadataStore.ActiveProfile()
	if err != nil {
		return "", err
	}
	if active != "" {
		return active, nil
	}
	return "default", nil
}

func envOrDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mergeProfileMetadata(base credential.ProfileMetadata, profile string, patch map[string]string) credential.ProfileMetadata {
	base.Profile = profile
	if value := strings.TrimSpace(patch["region"]); value != "" {
		base.Region = value
	}
	if value := strings.TrimSpace(patch["clientId"]); value != "" {
		base.ClientID = value
	}
	if value := strings.TrimSpace(patch["houseId"]); value != "" {
		base.HouseID = value
	}
	if value := strings.TrimSpace(patch["qrDevice"]); value != "" {
		base.QRDevice = value
	}
	return base
}

func requireJSONFlag(flags cliFlags, usage string) error {
	if !flags.bool("json") {
		return fmt.Errorf("%s", usage)
	}
	return nil
}
