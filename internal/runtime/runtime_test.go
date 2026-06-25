package runtime

import (
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
)

func TestInvokeReturnsGenericNotSupportedForUnknownImplementedFallback(t *testing.T) {
	engine := NewEngine(true)
	engine.now = fixedClock()
	request := contract.Request{
		ContractVersion: contract.Version,
		RequestID:       "req-generic",
		Locale:          "zh-CN",
		Utterance:       "执行",
		Intent:          "home.summary",
	}

	response := engine.Invoke(request)
	if response.Status != "not_supported" {
		t.Fatalf("status = %s", response.Status)
	}
}

func fixedClock() func() time.Time {
	now := time.Unix(1_700_000_000, 0)
	return func() time.Time {
		return now
	}
}
