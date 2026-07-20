package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/credential"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/lanmcp"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

type runtimeContext struct {
	Profile      string
	Region       string
	Endpoint     api.Endpoint
	ClientID     string
	HouseID      string
	BizType      string
	Language     string
	ControlMode  string
	GatewayIP    string
	LANEndpoint  string
	AccessToken  string
	TokenSource  string
	Metadata     credential.ProfileMetadata
	MetadataOK   bool
	TokenPresent bool
}

const defaultRuntimeRegion = "cn"

const (
	controlModeCloud          = "cloud"
	controlModeLocalPreferred = "local-preferred"
	controlModeLocalOnly      = "local-only"
)

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
		defaultRuntimeRegion,
	)
	endpoint, err := resolveEndpoint(region)
	if err != nil {
		return runtimeContext{}, err
	}
	clientID := firstNonEmpty(metadata.ClientID)
	flagBizType := flags.string("biz-type", flags.string("bizType", ""))
	envBizType := strings.TrimSpace(os.Getenv("YEELIGHT_HOME_BIZ_TYPE"))
	bizType, err := api.NormalizeBizType(firstNonEmpty(flagBizType, envBizType, metadata.BizType, api.BizTypeConsumer))
	if err != nil {
		return runtimeContext{}, err
	}
	storedBizType, err := api.NormalizeBizType(metadata.BizType)
	if err != nil {
		return runtimeContext{}, err
	}
	houseID := flags.string("house-id", "")
	if houseID == "" {
		envHouseID := strings.TrimSpace(os.Getenv("YEELIGHT_HOME_HOUSE_ID"))
		if bizType == storedBizType || envBizType != "" {
			houseID = firstNonEmpty(envHouseID, metadata.HouseID)
		}
	}
	language, _ := i18n.Normalize(firstNonEmpty(flags.string("language", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_LANGUAGE")), metadata.Language))
	controlMode, err := normalizeControlMode(firstNonEmpty(flags.string("control-mode", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_CONTROL_MODE")), metadata.ControlMode, controlModeCloud))
	if err != nil {
		return runtimeContext{}, err
	}
	gatewayOverride := firstNonEmpty(flags.string("gateway-ip", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_GATEWAY_IP")))
	endpointOverride := firstNonEmpty(flags.string("lan-endpoint", ""), strings.TrimSpace(os.Getenv("YEELIGHT_HOME_LAN_ENDPOINT")))
	gatewayIP := firstNonEmpty(gatewayOverride, metadata.GatewayIP)
	lanEndpoint := metadata.LANEndpoint
	if endpointOverride != "" {
		lanEndpoint = endpointOverride
	} else if gatewayOverride != "" {
		lanEndpoint = ""
	}
	lanEndpoint, err = resolveLANEndpoint(gatewayIP, lanEndpoint)
	if err != nil {
		return runtimeContext{}, err
	}
	if controlMode != controlModeCloud && lanEndpoint == "" {
		return runtimeContext{}, fmt.Errorf("control mode %s requires --gateway-ip or --lan-endpoint", controlMode)
	}
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
		BizType:      bizType,
		Language:     language,
		ControlMode:  controlMode,
		GatewayIP:    gatewayIP,
		LANEndpoint:  lanEndpoint,
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
	region := firstNonEmpty(flags.string("region", ""), strings.TrimSpace(os.Getenv("YEELIGHT_CLOUD_REGION")), defaultRuntimeRegion)
	return resolveEndpoint(region)
}

func profileFromEnv() string {
	return envOrDefault("YEELIGHT_HOME_PROFILE", "default")
}

func (app *app) resolveTargetProfile(flags cliFlags) (string, error) {
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

func (app *app) resolveProfile(flags cliFlags) (string, error) {
	return app.resolveTargetProfile(flags)
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
	if value := strings.TrimSpace(patch[semantic.FieldRegion]); value != "" {
		base.Region = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldClientID]); value != "" {
		base.ClientID = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldHouseID]); value != "" {
		base.HouseID = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldBizType]); value != "" {
		base.BizType = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldQRDevice]); value != "" {
		base.QRDevice = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldLanguage]); value != "" {
		base.Language = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldControlMode]); value != "" {
		base.ControlMode = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldGatewayIP]); value != "" {
		base.GatewayIP = value
	}
	if value := strings.TrimSpace(patch[semantic.FieldLANEndpoint]); value != "" {
		base.LANEndpoint = value
	}
	return base
}

func normalizeControlMode(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", controlModeCloud:
		return controlModeCloud, nil
	case controlModeLocalPreferred:
		return controlModeLocalPreferred, nil
	case controlModeLocalOnly:
		return controlModeLocalOnly, nil
	default:
		return "", fmt.Errorf("control mode must be cloud, local-preferred, or local-only")
	}
}

func resolveBizType(flags cliFlags, stored string) (string, error) {
	return api.NormalizeBizType(firstNonEmpty(
		flags.string("biz-type", flags.string("bizType", "")),
		strings.TrimSpace(os.Getenv("YEELIGHT_HOME_BIZ_TYPE")),
		stored,
		api.BizTypeConsumer,
	))
}

func resolveLANEndpoint(gatewayIP string, endpoint string) (string, error) {
	if strings.TrimSpace(endpoint) != "" {
		return lanmcp.NormalizeEndpoint(endpoint)
	}
	if strings.TrimSpace(gatewayIP) != "" {
		return lanmcp.EndpointForGateway(gatewayIP)
	}
	return "", nil
}

func requireJSONFlag(flags cliFlags, usage string) error {
	if !flags.bool("json") {
		return fmt.Errorf("%s", usage)
	}
	return nil
}
