package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

const productPediaResourceBaseURL = "https://rag-resources.yeelight.com/products/sku-res"

func (client MetadataReadonlyClient) RunProductPediaSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	query := productPediaQueryFromReadonlyRequest(request)
	if query == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "product.pedia.search", "product_pedia_query_missing"), nil
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, client.endpoint.PediaBaseURL()+"/v1/pedia/product/r/search", map[string]any{
		"multiField": query,
	}, requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	})
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("product pedia search", response)
	}
	products := projectProductPediaRows(response["data"], productPediaLimitFromRequest(request))
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		Capability: "product.pedia.search",
		Data: map[string]any{
			semantic.FieldQuery:          query,
			semantic.FieldTotal:          productPediaTotal(response[semantic.FieldData], len(products)),
			semantic.FieldReturned:       len(products),
			semantic.FieldProducts:       products,
			semantic.FieldResourceStatus: "candidate_urls_may_not_exist",
			semantic.FieldCachePolicy: map[string]any{
				semantic.FieldScope:      "profile_global_product_pedia",
				semantic.FieldTTLSeconds: 86400,
				semantic.FieldPersistent: false,
			},
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func productPediaQueryFromReadonlyRequest(request MetadataReadonlyRequest) string {
	for _, key := range semantic.ProductPediaQueryFields() {
		if value := stringFromAny(request.Parameters[key]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(request.Utterance)
}

func productPediaLimitFromRequest(request MetadataReadonlyRequest) int {
	limit := intFromAny(request.Parameters[semantic.FieldLimit])
	if limit == 0 {
		limit = intFromAny(request.Parameters[semantic.FieldPageSize])
	}
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func productPediaTotal(data any, fallback int) int {
	if item, ok := data.(map[string]any); ok {
		if total := intFromAny(item[semantic.FieldTotal]); total > 0 {
			return total
		}
	}
	return fallback
}

func projectProductPediaRows(data any, limit int) []any {
	rows := rowsFromData(data)
	products := make([]any, 0, productPediaMinInt(len(rows), limit))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		products = append(products, projectProductPediaSummary(item))
		if len(products) >= limit {
			break
		}
	}
	return products
}

func projectProductPediaSummary(item map[string]any) map[string]any {
	productCode := firstAnyString(item, semantic.ProductCodeFields()...)
	product := map[string]any{}
	copyProductPediaResponseMappings(product, item, semantic.ProductPediaSummaryMappings())
	if value := productPediaFeatureValue(item[semantic.FieldFeatures]); value != nil {
		product[semantic.FieldFeatures] = value
	}
	attachments := projectProductPediaAttachments(item[semantic.FieldAttachments])
	if len(attachments) > 0 {
		product[semantic.FieldAttachments] = attachments
	}
	if productCode != "" {
		product[semantic.FieldResources] = productPediaResources(productCode, attachments)
	} else {
		product[semantic.FieldResources] = map[string]any{
			semantic.FieldAttachments: attachments,
		}
	}
	return product
}

func copyProductPediaAlias(output map[string]any, source map[string]any, outputKey string, inputKeys ...string) {
	for _, key := range inputKeys {
		value, ok := source[key]
		if !ok || value == nil {
			continue
		}
		if projected, keep := productPediaProjectedValue(value); keep {
			if text, ok := projected.(string); ok && text == "" {
				continue
			}
			output[outputKey] = projected
			return
		}
	}
}

func productPediaProjectedValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, true
	case bool:
		return typed, true
	case string:
		return strings.TrimSpace(typed), true
	case float64, int, int64:
		return typed, true
	case []any, map[string]any:
		return sanitizeCloudData(typed), true
	default:
		return nil, false
	}
}

func productPediaResources(productCode string, attachments []any) map[string]any {
	manualURL := fmt.Sprintf("%s/%s/split/%s_split.pdf", productPediaResourceBaseURL, productCode, productCode)
	faqURL := fmt.Sprintf("%s/%s/faq/%s/.json", productPediaResourceBaseURL, productCode, productCode)
	return map[string]any{
		semantic.FieldProductCode:        productCode,
		semantic.FieldManualCandidateURL: manualURL,
		semantic.FieldFAQCandidateURL:    faqURL,
		semantic.FieldCandidateStatus:    "not_verified",
		semantic.FieldAttachments:        attachments,
		semantic.FieldManualAttachments:  filterProductPediaAttachments(attachments, "manual"),
	}
}

func projectProductPediaAttachments(value any) []any {
	rows := rowsFromData(value)
	attachments := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		url := firstAnyString(item, semantic.FieldURL)
		if url == "" {
			continue
		}
		attachment := map[string]any{}
		copyProductPediaResponseMappings(attachment, item, semantic.ProductPediaAttachmentMappings())
		copyProductPediaAlias(attachment, item, semantic.FieldProductCode, semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCode))
		attachments = append(attachments, attachment)
	}
	return attachments
}

func copyProductPediaResponseMappings(output map[string]any, source map[string]any, mappings []semantic.ResponseFieldMapping) {
	for _, mapping := range mappings {
		copyProductPediaMappedValue(output, source, mapping.Public, mapping.Internal...)
	}
}

func copyProductPediaMappedValue(output map[string]any, source map[string]any, outputKey string, inputKeys ...string) {
	for _, key := range inputKeys {
		value, ok := source[key]
		if !ok || isSensitiveCloudField(strings.ToLower(strings.TrimSpace(key))) {
			continue
		}
		if projected, keep := productPediaProjectedValue(value); keep {
			output[outputKey] = projected
			return
		}
	}
}

func filterProductPediaAttachments(attachments []any, kind string) []any {
	result := []any{}
	for _, attachment := range attachments {
		item, ok := attachment.(map[string]any)
		if !ok {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			stringFromAny(item[semantic.FieldType]),
			stringFromAny(item[semantic.FieldName]),
			stringFromAny(item[semantic.FieldURL]),
		}, " "))
		if (kind == "manual" && (strings.Contains(haystack, "说明书") || strings.Contains(haystack, "manual") || strings.Contains(haystack, "_split.pdf"))) ||
			(kind == "faq" && strings.Contains(haystack, "faq")) {
			result = append(result, item)
		}
	}
	return result
}

func productPediaFeatureValue(value any) any {
	switch typed := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			return truncateText(trimmed, 500)
		}
	case []any, map[string]any:
		return sanitizeCloudData(typed)
	}
	return nil
}

func productPediaMinInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
