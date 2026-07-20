package api

import (
	"context"
	"fmt"
	"strings"
)

const (
	BizTypeConsumer   = "0"
	BizTypeCommercial = "1"
)

type bizTypeContextKey struct{}

func NormalizeBizType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "c", "c端", "consumer", "home", "普通", "普通家庭":
		return BizTypeConsumer, nil
	case "1", "b", "b端", "business", "commercial", "project", "商照", "商照家庭", "商照项目":
		return BizTypeCommercial, nil
	default:
		return "", fmt.Errorf("biz type must be 0 (consumer home) or 1 (commercial project)")
	}
}

func WithBizType(ctx context.Context, value string) context.Context {
	normalized, err := NormalizeBizType(value)
	if err != nil {
		return ctx
	}
	return context.WithValue(ctx, bizTypeContextKey{}, normalized)
}

func bizTypeFromContext(ctx context.Context) string {
	value, _ := ctx.Value(bizTypeContextKey{}).(string)
	return strings.TrimSpace(value)
}

func effectiveBizType(ctx context.Context, explicit string) string {
	value := strings.TrimSpace(explicit)
	if value == "" {
		value = bizTypeFromContext(ctx)
	}
	normalized, err := NormalizeBizType(value)
	if err != nil {
		return BizTypeConsumer
	}
	return normalized
}
