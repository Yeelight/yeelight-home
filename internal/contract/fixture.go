package contract

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateSkillRequestFixture(fixture map[string]any) error {
	data, err := json.Marshal(fixture)
	if err != nil {
		return fmt.Errorf("encode request fixture: %w", err)
	}
	_, err = DecodeRequest(data)
	return err
}

func ValidateSkillResponseFixture(fixture map[string]any) error {
	for _, key := range []string{"contractVersion", "requestId", "status", "userMessage"} {
		if strings.TrimSpace(stringValue(fixture[key])) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	if stringValue(fixture["contractVersion"]) != Version {
		return fmt.Errorf("unsupported contractVersion %q", stringValue(fixture["contractVersion"]))
	}
	if !isKnownResponseStatus(stringValue(fixture["status"])) {
		return fmt.Errorf("unsupported status %q", stringValue(fixture["status"]))
	}
	for key := range fixture {
		if !isAllowedResponseField(key) {
			return fmt.Errorf("unsupported response field %q", key)
		}
	}
	if containsSensitiveKey(fixture) {
		return fmt.Errorf("response fixture contains sensitive token-like key")
	}
	return nil
}

func isKnownResponseStatus(status string) bool {
	switch status {
	case "success",
		"partial",
		"clarification_required",
		"confirmation_required",
		"blocked",
		"not_supported",
		"auth_required",
		"error":
		return true
	default:
		return false
	}
}

func isAllowedResponseField(field string) bool {
	switch field {
	case "contractVersion",
		"requestId",
		"status",
		"userMessage",
		"result",
		"clarification",
		"confirmation",
		"execution",
		"memory",
		"recommendation",
		"warnings",
		"traceId",
		"metrics",
		"error":
		return true
	default:
		return false
	}
}

func containsSensitiveKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.ToLower(key)
			for _, forbidden := range []string{"token", "secret", "authorization", "cookie"} {
				if strings.Contains(normalized, forbidden) {
					return true
				}
			}
			if containsSensitiveKey(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsSensitiveKey(nested) {
				return true
			}
		}
	}
	return false
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
