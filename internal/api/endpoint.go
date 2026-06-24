package api

import (
	"errors"
	"os"
	"strings"
)

type Endpoint struct {
	Region  string
	BaseURL string
}

func ResolveEndpoint(region string) (Endpoint, error) {
	normalized := strings.ToLower(strings.TrimSpace(region))
	switch normalized {
	case "dev", "local-dev":
		return Endpoint{Region: "dev", BaseURL: "http://api-dev.yeedev.com/apis/iot"}, nil
	case "cn", "cloud_region_cn":
		return Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com"}, nil
	case "sg", "cloud_region_sg":
		return Endpoint{Region: "sg", BaseURL: "https://api-sg.yeelight.com"}, nil
	case "us", "cloud_region_us":
		return Endpoint{Region: "us", BaseURL: "https://api-us.yeelight.com"}, nil
	case "eu", "de", "cloud_region_eu":
		return Endpoint{Region: "eu", BaseURL: "https://api-de.yeelight.com"}, nil
	default:
		return Endpoint{}, errors.New("unknown Yeelight API region")
	}
}

func ResolveEndpointFromEnv() (Endpoint, error) {
	return ResolveEndpointFromLookup(os.LookupEnv)
}

func ResolveEndpointFromLookup(lookup func(string) (string, bool)) (Endpoint, error) {
	if baseURL, ok := lookup("YEELIGHT_API_BASE_URL"); ok && strings.TrimSpace(baseURL) != "" {
		return Endpoint{Region: "custom", BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/")}, nil
	}
	if region, ok := lookup("YEELIGHT_CLOUD_REGION"); ok && strings.TrimSpace(region) != "" {
		return ResolveEndpoint(region)
	}
	return ResolveEndpoint("dev")
}

func (endpoint Endpoint) AccountBaseURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(endpoint.BaseURL), "/")
	return strings.TrimSuffix(baseURL, "/apis/iot")
}
