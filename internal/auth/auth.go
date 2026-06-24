package auth

import (
	"os"
	"strings"
)

type Status struct {
	Authenticated bool   `json:"authenticated"`
	Profile       string `json:"profile"`
	TokenPresent  bool   `json:"tokenPresent"`
	TokenStore    string `json:"tokenStore"`
}

func StatusFromEnv() Status {
	return StatusFromLookup(os.LookupEnv)
}

func StatusFromLookup(lookup func(string) (string, bool)) Status {
	profile := "default"
	if value, ok := lookup("YEELIGHT_HOME_PROFILE"); ok && strings.TrimSpace(value) != "" {
		profile = strings.TrimSpace(value)
	}

	tokenPresent := false
	if value, ok := lookup("YEELIGHT_HOME_ACCESS_TOKEN"); ok && strings.TrimSpace(value) != "" {
		tokenPresent = true
	}

	authenticated := false
	if value, ok := lookup("YEELIGHT_HOME_AUTHENTICATED"); ok {
		authenticated = value == "1" || strings.EqualFold(value, "true")
	}

	return Status{
		Authenticated: authenticated,
		Profile:       profile,
		TokenPresent:  tokenPresent,
		TokenStore:    "system_credential_store",
	}
}
