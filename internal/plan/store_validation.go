package plan

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func randomID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate plan id: %w", err)
	}
	return "plan_" + hex.EncodeToString(bytes), nil
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func containsSensitive(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if tokenLike(key) || containsSensitive(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsSensitive(nested) {
				return true
			}
		}
	}
	return false
}

func tokenLike(value string) bool {
	normalized := strings.ToLower(value)
	for _, forbidden := range []string{"token", "secret", "authorization", "cookie"} {
		if strings.Contains(normalized, forbidden) {
			return true
		}
	}
	return false
}
