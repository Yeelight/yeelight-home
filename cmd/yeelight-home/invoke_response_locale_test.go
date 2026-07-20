package main

import (
	"testing"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func TestLocalizeLegacyInvokeResponseUsesPreciseEnglishSummary(t *testing.T) {
	response := localizeLegacyInvokeResponse("en-US", contract.Response{
		Status: "success", TraceID: "home-summary-readonly", UserMessage: "已找到 43 个家庭。",
		Result: map[string]any{semantic.FieldHouseCount: 43},
	})
	if response.UserMessage != "Found 43 homes." || len(response.Warnings) != 1 || response.Warnings[0] != "user_message_locale_fallback" {
		t.Fatalf("response = %#v", response)
	}
}

func TestLocalizeLegacyInvokeResponsePreservesTranslatedMessageAndEntityName(t *testing.T) {
	for _, message := range []string{
		"Set the brightness for Living room.",
		"Set the brightness for 客厅灯.",
	} {
		response := localizeLegacyInvokeResponse("en-US", contract.Response{Status: "success", UserMessage: message})
		if response.UserMessage != message || len(response.Warnings) != 0 {
			t.Fatalf("message %q localized to %#v", message, response)
		}
	}
}

func TestLocalizeLegacyInvokeResponseUsesStatusFallback(t *testing.T) {
	response := localizeLegacyInvokeResponse("en-US", contract.Response{
		Status: "clarification_required", UserMessage: "请补充要配置的必要信息。",
	})
	if response.UserMessage != "More information is needed before Yeelight Home can continue." {
		t.Fatalf("response = %#v", response)
	}
}
