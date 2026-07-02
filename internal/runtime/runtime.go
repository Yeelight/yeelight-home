package runtime

import (
	"time"

	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
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
				semantic.FieldAPICalls:  0,
				semantic.FieldCacheHits: 0,
				semantic.FieldRuntimeMs: engine.now().Sub(start).Milliseconds(),
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
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
			semantic.FieldRuntimeMs: engine.now().Sub(start).Milliseconds(),
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
			semantic.FieldIntent:           request.Intent,
			semantic.FieldPolicyStatus:     policy.policyStatus,
			semantic.FieldRisk:             policy.risk,
			semantic.FieldBlockReason:      policy.blockReason,
			semantic.FieldPersistentWrites: false,
			semantic.FieldNextAction:       policy.nextAction,
		},
		Warnings: []string{policy.policyStatus, policy.blockReason},
		TraceID:  "governed-intent-blocked",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
			semantic.FieldRuntimeMs: runtimeMs,
		},
		Error: &contract.Error{
			Code:    policy.blockReason,
			Message: policy.userMessage,
		},
	}
}
