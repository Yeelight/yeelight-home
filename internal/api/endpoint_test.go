package api

import "testing"

func TestResolveEndpointSupportsDevAndCloudRegions(t *testing.T) {
	tests := map[string]string{
		"dev": "http://api-dev.yeedev.com/apis/iot",
		"cn":  "https://api.yeelight.com/apis/iot",
		"sg":  "https://api-sg.yeelight.com/apis/iot",
		"us":  "https://api-us.yeelight.com/apis/iot",
		"eu":  "https://api-de.yeelight.com/apis/iot",
	}

	for region, expected := range tests {
		t.Run(region, func(t *testing.T) {
			endpoint, err := ResolveEndpoint(region)
			if err != nil {
				t.Fatalf("ResolveEndpoint error: %v", err)
			}
			if endpoint.BaseURL != expected {
				t.Fatalf("BaseURL = %s", endpoint.BaseURL)
			}
		})
	}
}

func TestResolveEndpointRejectsUnknownRegion(t *testing.T) {
	if _, err := ResolveEndpoint("mars"); err == nil {
		t.Fatal("expected unknown region to be rejected")
	}
}

func TestResolveEndpointFromLookupUsesOverride(t *testing.T) {
	endpoint, err := ResolveEndpointFromLookup(func(key string) (string, bool) {
		switch key {
		case "YEELIGHT_API_BASE_URL":
			return "http://localhost:8080/apis/iot", true
		case "YEELIGHT_CLOUD_REGION":
			return "cn", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("ResolveEndpointFromLookup error: %v", err)
	}
	if endpoint.Region != "custom" {
		t.Fatalf("Region = %s", endpoint.Region)
	}
	if endpoint.BaseURL != "http://localhost:8080/apis/iot" {
		t.Fatalf("BaseURL = %s", endpoint.BaseURL)
	}
}

func TestResolveEndpointFromLookupDefaultsToCN(t *testing.T) {
	endpoint, err := ResolveEndpointFromLookup(func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("ResolveEndpointFromLookup error: %v", err)
	}
	if endpoint.Region != "cn" || endpoint.BaseURL != "https://api.yeelight.com/apis/iot" {
		t.Fatalf("endpoint = %#v", endpoint)
	}
}

func TestEndpointAccountBaseURLStripsIOTPrefix(t *testing.T) {
	tests := map[string]string{
		"http://api-dev.yeedev.com/apis/iot": "http://api-dev.yeedev.com",
		"https://api.yeelight.com/apis/iot":  "https://api.yeelight.com",
		"https://api.yeelight.com":           "https://api.yeelight.com",
		"http://localhost:8080/apis/iot/":    "http://localhost:8080",
	}

	for baseURL, want := range tests {
		endpoint := Endpoint{Region: "test", BaseURL: baseURL}
		if got := endpoint.AccountBaseURL(); got != want {
			t.Fatalf("AccountBaseURL(%q) = %q, want %q", baseURL, got, want)
		}
	}
}
