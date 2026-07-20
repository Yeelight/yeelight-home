package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func localizeLegacyInvokeResponse(locale string, response contract.Response) contract.Response {
	if locale != i18n.English || !legacyChineseSentence(response.UserMessage) {
		return response
	}
	response.UserMessage = englishLegacyInvokeMessage(response)
	response.Warnings = appendWarning(response.Warnings, "user_message_locale_fallback")
	return response
}

func legacyChineseSentence(message string) bool {
	if !strings.ContainsAny(message, "。，；：！？") {
		return false
	}
	for _, character := range message {
		if unicode.Is(unicode.Han, character) {
			return true
		}
	}
	return false
}

func englishLegacyInvokeMessage(response contract.Response) string {
	switch response.TraceID {
	case "home-summary-readonly", "home-search-readonly":
		if count := metricInt(response.Result[semantic.FieldHouseCount]); count >= 0 {
			return fmt.Sprintf("Found %d homes.", count)
		}
	case "entity-list-readonly":
		if count := metricInt(response.Result[semantic.FieldTotal]); count >= 0 {
			return fmt.Sprintf("Found %d entities.", count)
		}
	case "state-query-readonly":
		return "Read the current device state."
	}
	switch response.Status {
	case "success":
		return i18n.Text(i18n.English, i18n.RuntimeLegacySuccess)
	case "partial":
		return i18n.Text(i18n.English, i18n.RuntimeLegacyPartial)
	case "clarification_required":
		return i18n.Text(i18n.English, i18n.RuntimeLegacyClarification)
	case "blocked":
		return i18n.Text(i18n.English, i18n.RuntimeLegacyBlocked)
	case "auth_required":
		return i18n.Text(i18n.English, i18n.RuntimeLegacyAuthRequired)
	case "not_supported":
		return i18n.Text(i18n.English, i18n.RuntimeLegacyNotSupported)
	default:
		return i18n.Text(i18n.English, i18n.RuntimeInvokeFailed)
	}
}
