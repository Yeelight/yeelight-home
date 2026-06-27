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
		return Endpoint{Region: "cn", BaseURL: "https://api.yeelight.com/apis/iot"}, nil
	case "sg", "cloud_region_sg":
		return Endpoint{Region: "sg", BaseURL: "https://api-sg.yeelight.com/apis/iot"}, nil
	case "us", "cloud_region_us":
		return Endpoint{Region: "us", BaseURL: "https://api-us.yeelight.com/apis/iot"}, nil
	case "eu", "de", "cloud_region_eu":
		return Endpoint{Region: "eu", BaseURL: "https://api-de.yeelight.com/apis/iot"}, nil
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
	return ResolveEndpoint("cn")
}

func (endpoint Endpoint) AccountBaseURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(endpoint.BaseURL), "/")
	return strings.TrimSuffix(baseURL, "/apis/iot")
}

func (endpoint Endpoint) PediaBaseURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(endpoint.BaseURL), "/")
	if strings.HasSuffix(baseURL, "/apis/iot") {
		accountBaseURL := strings.TrimSuffix(baseURL, "/apis/iot")
		if isLocalTestBaseURL(accountBaseURL) {
			return accountBaseURL + "/apis/c"
		}
	}
	return "https://api.yeelight.com/apis/c"
}

func isLocalTestBaseURL(baseURL string) bool {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	switch {
	case strings.HasPrefix(normalized, "http://127.0.0.1:"),
		strings.HasPrefix(normalized, "http://localhost:"),
		strings.HasPrefix(normalized, "http://[::1]:"):
		return true
	default:
		return false
	}
}
