package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
			"query":          query,
			"total":          productPediaTotal(response["data"], len(products)),
			"returned":       len(products),
			"products":       products,
			"resourceStatus": "candidate_urls_may_not_exist",
			"cachePolicy": map[string]any{
				"scope":      "profile_global_product_pedia",
				"ttlSeconds": 86400,
				"persistent": false,
			},
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func productPediaQueryFromReadonlyRequest(request MetadataReadonlyRequest) string {
	for _, key := range []string{
		"multiField",
		"keyword",
		"query",
		"queryString",
		"name",
		"productName",
		"productShortName",
		"materialCode",
		"sku",
		"productSku",
		"productSkuFullText",
		"spu",
		"productSpu",
		"model",
		"productModel",
		"modelNo",
		"barcode",
		"pid",
	} {
		if value := stringFromAny(request.Parameters[key]); value != "" {
			return value
		}
	}
	return strings.TrimSpace(request.Utterance)
}

func productPediaLimitFromRequest(request MetadataReadonlyRequest) int {
	limit := intFromAny(request.Parameters["limit"])
	if limit == 0 {
		limit = intFromAny(request.Parameters["pageSize"])
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
		if total := intFromAny(item["total"]); total > 0 {
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
	materialCode := firstAnyString(item, "materialCode")
	product := map[string]any{}
	copyProductPediaFields(product, item, []string{
		"id",
		"materialCode",
		"oldMaterialCode",
		"pid",
		"productName",
		"productBrand",
		"productModel",
		"productSku",
		"productSpu",
		"productLine",
		"productCategoryName",
		"productLargeClass",
		"productSmallClass",
		"productShortName",
		"productStatus",
		"productStage",
		"productStatusStage",
		"productSeries",
		"specsCode",
		"barcode",
		"modelNo",
		"productType",
		"productSource",
		"productUsePlatform",
		"productSaleType",
		"productLevel",
		"productMeasureUnit",
		"productSellingPoint",
		"lightingDesignLineStyle",
		"lightingDesignDeviceCategory",
		"productDeclareNo",
		"productDeclareName",
		"productDeclareUnit",
		"stockType",
		"baseUnit",
		"saleUnit",
		"baseUnitNum",
		"minOrderQty",
		"isAccessNet",
		"valid",
		"isSupportYeelightPro",
		"isSupportHomekit",
		"threeViewsStatus",
		"publicityMapStatus",
		"parameterStatus",
		"opticalFileStatus",
		"completion",
		"supplierName",
		"terminalDeliveryMode",
		"terminalDeliveryPeriod",
		"pediaDisplay",
		"productWxQrcode",
		"manualWxQrcode",
		"quotationType",
		"productSkuEn",
		"overseaSaleRegion",
		"productStatusName",
		"productStageName",
		"productTypeName",
		"productUsePlatformName",
		"productSaleTypeName",
		"productLevelName",
		"quotationTypeDesc",
		"overseaSaleRegionDesc",
		"volumn",
		"weight",
		"factory",
		"features",
	})
	if value := productPediaFeatureValue(item["features"]); value != nil {
		product["features"] = value
	}
	copyProductPediaAlias(product, item, "brand", "productBrand", "brand")
	copyProductPediaAlias(product, item, "model", "productModel", "model")
	copyProductPediaAlias(product, item, "sku", "productSku", "sku")
	copyProductPediaAlias(product, item, "spu", "productSpu", "spu")
	copyProductPediaAlias(product, item, "shortName", "productShortName")
	copyProductPediaAlias(product, item, "category", "productCategoryName", "categoryName")
	copyProductPediaAlias(product, item, "largeClass", "productLargeClass")
	copyProductPediaAlias(product, item, "smallClass", "productSmallClass")
	copyProductPediaAlias(product, item, "status", "productStatusName", "productStatus")
	copyProductPediaAlias(product, item, "stage", "productStageName", "productStage")
	copyProductPediaAlias(product, item, "series", "productSeries")
	copyProductPediaAlias(product, item, "type", "productTypeName", "productType")
	copyProductPediaAlias(product, item, "usePlatform", "productUsePlatformName", "productUsePlatform")
	copyProductPediaAlias(product, item, "supportYeelightPro", "isSupportYeelightPro")
	copyProductPediaAlias(product, item, "supportHomekit", "isSupportHomekit")
	attachments := projectProductPediaAttachments(item["attachments"])
	if len(attachments) > 0 {
		product["attachments"] = attachments
	}
	if materialCode != "" {
		product["resources"] = productPediaResources(materialCode, attachments)
	} else {
		product["resources"] = map[string]any{
			"attachments": attachments,
		}
	}
	return product
}

func copyProductPediaFields(output map[string]any, source map[string]any, keys []string) {
	for _, key := range keys {
		value, ok := source[key]
		if !ok || isSensitiveCloudField(strings.ToLower(strings.TrimSpace(key))) {
			continue
		}
		if projected, keep := productPediaProjectedValue(value); keep {
			output[key] = projected
		}
	}
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

func productPediaResources(materialCode string, attachments []any) map[string]any {
	manualURL := fmt.Sprintf("%s/%s/split/%s_split.pdf", productPediaResourceBaseURL, materialCode, materialCode)
	faqURL := fmt.Sprintf("%s/%s/faq/%s/.json", productPediaResourceBaseURL, materialCode, materialCode)
	return map[string]any{
		"materialCode":       materialCode,
		"manualCandidateUrl": manualURL,
		"faqCandidateUrl":    faqURL,
		"candidateStatus":    "not_verified",
		"attachments":        attachments,
		"manualAttachments":  filterProductPediaAttachments(attachments, "manual"),
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
		url := firstAnyString(item, "url")
		if url == "" {
			continue
		}
		attachment := map[string]any{}
		for _, key := range []string{
			"id",
			"bizId",
			"bizType",
			"materialCode",
			"url",
			"type",
			"name",
			"sort",
			"createUid",
			"createTime",
			"updateUid",
			"updateTime",
		} {
			value, ok := item[key]
			if !ok || isSensitiveCloudField(strings.ToLower(strings.TrimSpace(key))) {
				continue
			}
			if projected, keep := productPediaProjectedValue(value); keep {
				attachment[key] = projected
			}
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}

func filterProductPediaAttachments(attachments []any, kind string) []any {
	result := []any{}
	for _, attachment := range attachments {
		item, ok := attachment.(map[string]any)
		if !ok {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			stringFromAny(item["type"]),
			stringFromAny(item["name"]),
			stringFromAny(item["url"]),
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
