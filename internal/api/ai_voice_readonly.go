package api

import (
	"context"
	"net/http"
)

func (client MetadataReadonlyClient) RunAIVoiceProductList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readPath(ctx, request, "ai_voice.product.list", "/v1/ai/voice/product/r/list", http.MethodGet, nil, map[string]any{"products": nil})
}
