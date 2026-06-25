package auth

import (
	"encoding/json"
	"testing"
)

func TestStatusFromEnvDoesNotExposeTokenValue(t *testing.T) {
	env := map[string]string{
		"YEELIGHT_HOME_AUTHENTICATED": "1",
		"YEELIGHT_HOME_PROFILE":       "family-main",
		"YEELIGHT_HOME_ACCESS_TOKEN":  "secret-token-value",
	}

	status := StatusFromLookup(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}

	if !status.Authenticated {
		t.Fatal("expected authenticated status")
	}
	if status.Profile != "family-main" {
		t.Fatalf("profile = %s", status.Profile)
	}
	if !status.TokenPresent {
		t.Fatal("expected token presence to be reported")
	}
	if string(data) == "" || contains(string(data), "secret-token-value") {
		t.Fatalf("status leaked token value: %s", string(data))
	}
}

func TestStatusFromEnvDefaultsToUnauthenticated(t *testing.T) {
	status := StatusFromLookup(func(key string) (string, bool) {
		return "", false
	})

	if status.Authenticated {
		t.Fatal("expected unauthenticated status")
	}
	if status.Profile != "default" {
		t.Fatalf("profile = %s", status.Profile)
	}
}

func contains(text string, needle string) bool {
	return len(needle) > 0 && len(text) >= len(needle) && index(text, needle) >= 0
}

func index(text string, needle string) int {
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
