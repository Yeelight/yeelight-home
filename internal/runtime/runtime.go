package runtime

import (
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
)

type Engine struct {
	authenticated bool
	now           func() time.Time
}

func NewEngine(authenticated bool) Engine {
	return Engine{
		authenticated: authenticated,
		now:           time.Now,
	}
}

func (engine Engine) Invoke(request contract.Request) contract.Response {
	start := engine.now()
	if !engine.authenticated {
		return contract.Response{
			ContractVersion: contract.Version,
			RequestID:       request.RequestID,
			Status:          "auth_required",
			UserMessage:     "需要先在本机完成 Yeelight 登录，请运行 yeelight-home auth login。",
			Warnings:        []string{},
			TraceID:         "local-auth-required",
			Metrics: map[string]any{
				"apiCalls":  0,
				"cacheHits": 0,
				"runtimeMs": engine.now().Sub(start).Milliseconds(),
			},
		}
	}
	if policy, ok := governedFallbackPolicyForIntent(request.Intent); ok {
		return governedFallbackResponse(request, policy, engine.now().Sub(start).Milliseconds())
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "not_supported",
		UserMessage:     "当前本地 Runtime 只完成了契约校验，尚未启用真实 Yeelight API 操作。",
		Warnings:        []string{},
		TraceID:         "local-contract-only",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
			"runtimeMs": engine.now().Sub(start).Milliseconds(),
		},
	}
}

type governedFallbackPolicy struct {
	policyStatus string
	risk         string
	blockReason  string
	userMessage  string
	nextAction   string
}

func governedFallbackPolicyForIntent(intent string) (governedFallbackPolicy, bool) {
	switch intent {
	default:
		return governedFallbackPolicy{}, false
	}
}

func ownerReviewPolicy(risk string, blockReason string, userMessage string) governedFallbackPolicy {
	return governedFallbackPolicy{
		policyStatus: "blocked_owner_review",
		risk:         risk,
		blockReason:  blockReason,
		userMessage:  userMessage,
		nextAction:   "等待 adapter、规则、测试数据和写后验证完成 owner review。",
	}
}

func governedFallbackResponse(request contract.Request, policy governedFallbackPolicy, runtimeMs int64) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     policy.userMessage,
		Result: map[string]any{
			"intent":           request.Intent,
			"policyStatus":     policy.policyStatus,
			"risk":             policy.risk,
			"blockReason":      policy.blockReason,
			"persistentWrites": false,
			"nextAction":       policy.nextAction,
		},
		Warnings: []string{policy.policyStatus, policy.blockReason},
		TraceID:  "governed-intent-blocked",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
			"runtimeMs": runtimeMs,
		},
		Error: &contract.Error{
			Code:    policy.blockReason,
			Message: policy.userMessage,
		},
	}
}
