package api

import (
	"context"
	"net/http"
	"strings"
)

func (client MetadataReadonlyClient) RunSceneScopedList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	roomID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["roomId"]),
		stringFromAny(request.Parameters["resId"]),
		stringFromAny(request.Parameters["id"]),
	))
	if houseID == "" && roomID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.scoped.list", "scene_scope_context_missing"), nil
	}
	body := readonlyBodyFromParameters(request.Parameters, "houseId", "roomId", "gatewayDeviceId", "pageNo", "pageSize")
	if houseID != "" {
		body["houseId"] = requestNumberOrStringForAPI(houseID)
	}
	if roomID != "" {
		body["roomId"] = requestNumberOrStringForAPI(roomID)
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/r/list", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("scene scoped list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "scene.scoped.list",
		Data: map[string]any{
			"roomId": roomID,
			"scenes": projectSceneRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunScheduleJobList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "schedule_job.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/schedulejob/r/list", map[string]any{"houseId": requestNumberOrStringForAPI(houseID)}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("schedule job list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "schedule_job.list",
		Data: map[string]any{
			"scheduleJobs": projectScheduleJobRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunMessageList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodGet, "/v1/messagecenter/r/messages", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("message list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: "message.list",
		Data: map[string]any{
			"messages": projectMessageRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunProductDomainList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodPost, "/v1/product-domain/r/list", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("product domain list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: "thing.product_domain.list",
		Data: map[string]any{
			"domains": projectCatalogRows(response["data"], projectProductDomainSummary),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunProductFAQList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, "pid", "productId", "keyword", "locale", "languageCode", "pageNo", "pageSize")
	if body["pid"] == nil {
		if productID := stringFromAny(body["productId"]); productID != "" {
			body["pid"] = productID
		}
	}
	delete(body, "productId")
	result, err := client.readFAQPath(ctx, request, "thing.product_faq.list", "/v1/platform/thing/product_faq/r/list", http.MethodPost, body, map[string]any{"faqs": nil})
	return result, err
}

func (client MetadataReadonlyClient) RunProductFAQDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	faqID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["faqId"]), stringFromAny(request.Parameters["id"])))
	if faqID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.product_faq.detail.get", "faq_context_missing"), nil
	}
	return client.readFAQPath(ctx, request, "thing.product_faq.detail.get", "/v1/platform/thing/product_faq/r/"+pathSegment(faqID)+"/detail", http.MethodGet, nil, map[string]any{"faq": nil})
}

func (client MetadataReadonlyClient) RunProductFAQTypeList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readFAQCatalogPath(ctx, request, "thing.product_faq.type.list", "/v1/platform/thing/product_faq/r/faq-types", "faqTypes")
}

func (client MetadataReadonlyClient) RunProductFAQItemTypeList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readFAQCatalogPath(ctx, request, "thing.product_faq.item_type.list", "/v1/platform/thing/product_faq/r/faq-item-types", "faqItemTypes")
}

func (client MetadataReadonlyClient) RunProductFAQLocaleList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodGet, "/v1/platform/thing/product_faq/r/locales", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("thing.product_faq.locale.list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: "thing.product_faq.locale.list",
		Data: map[string]any{
			"locales": projectCatalogRows(response["data"], projectLocaleSummary),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunProductFAQPageList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := productFAQQueryBody(request.Parameters)
	return client.readFAQPath(ctx, request, "thing.product_faq.page.list", "/v1/platform/thing/product_faq/r/page", http.MethodPost, body, map[string]any{"faqs": nil})
}

func (client MetadataReadonlyClient) RunProductFAQPageDetailList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := productFAQQueryBody(request.Parameters)
	return client.readFAQPath(ctx, request, "thing.product_faq.page_detail.list", "/v1/platform/thing/product_faq/r/pageDetail", http.MethodPost, body, map[string]any{"faqs": nil})
}

func (client MetadataReadonlyClient) RunThingCategoryList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.category.list", "/v2/thing/schema/category/r/list", map[string]any{"categories": nil}, "category_list")
}

func (client MetadataReadonlyClient) RunThingComponentList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.component.list", "/v2/thing/schema/component/r/list", map[string]any{"components": nil}, "component_list")
}

func (client MetadataReadonlyClient) RunThingComponentGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	componentID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["componentId"]),
		stringFromAny(request.Parameters["cid"]),
		stringFromAny(request.Parameters["id"]),
	))
	if componentID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.component.get", "component_context_missing"), nil
	}
	result, err := client.readThingSchemaV2Path(ctx, request, "thing.component.get", "/v2/thing/schema/component/r/"+pathSegment(componentID), map[string]any{"component": nil}, "component_detail")
	result.Data = withThingSchemaProductID(result.Data, componentID)
	return result, err
}

func (client MetadataReadonlyClient) RunThingPropertyList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.property.list", "/v2/thing/schema/property/r/list", map[string]any{"properties": nil}, "property_list")
}

func (client MetadataReadonlyClient) readFAQPath(ctx context.Context, request MetadataReadonlyRequest, capability string, path string, method string, body map[string]any, projection map[string]any) (MetadataReadonlyResult, error) {
	result, err := client.readPath(ctx, request, capability, path, method, body, projection)
	if err != nil {
		return result, err
	}
	if data, ok := result.Data.(map[string]any); ok {
		for key, value := range data {
			data[key] = projectCatalogRows(value, projectFAQSummary)
		}
	}
	return result, nil
}

func (client MetadataReadonlyClient) readFAQCatalogPath(ctx context.Context, request MetadataReadonlyRequest, capability string, path string, dataKey string) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodGet, path, nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError(capability, response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: capability,
		Data: map[string]any{
			dataKey: projectCatalogRows(response["data"], projectCodeDescriptionSummary),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func productFAQQueryBody(parameters map[string]any) map[string]any {
	body := readonlyBodyFromParameters(parameters, "pid", "productId", "moduleId", "keyword", "locale", "languageCode", "pageNo", "pageSize")
	if body["moduleId"] == nil {
		if productID := stringFromAny(body["productId"]); productID != "" {
			body["moduleId"] = productID
		} else if pid := stringFromAny(body["pid"]); pid != "" {
			body["moduleId"] = pid
		}
	}
	if body["pid"] == nil {
		if productID := stringFromAny(body["productId"]); productID != "" {
			body["pid"] = productID
		}
	}
	delete(body, "productId")
	if body["pageNo"] == nil {
		body["pageNo"] = 1
	}
	if body["pageSize"] == nil {
		body["pageSize"] = 20
	}
	return body
}

func (client MetadataReadonlyClient) readThingSchemaV2Path(ctx context.Context, request MetadataReadonlyRequest, capability string, path string, projection map[string]any, shape string) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodGet, path, nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError(capability, response)
	}
	data := map[string]any{
		"schemaVersion": "cloud-v2",
		"cachePolicy": map[string]any{
			"scope":      "profile_region_thing_schema_v2",
			"ttlSeconds": 86400,
			"persistent": false,
		},
	}
	for key := range projection {
		data[key] = projectThingSchemaV2Data(response["data"], shape)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		Capability: capability,
		Data:       data,
		RawShape:   responseDataType(response),
		APICalls:   1,
		Warnings:   []string{},
	}, nil
}

func projectScheduleJobRows(data any) []any {
	rows := nestedRowsFromData(data, "list", "rows", "scheduleJobs")
	jobs := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		job := compactMap(map[string]any{
			"id":          firstAnyString(item, "id", "scheduleJobId"),
			"houseId":     firstAnyString(item, "houseId"),
			"name":        firstAnyString(item, "name"),
			"startTime":   firstAnyString(item, "startTime"),
			"endTime":     firstAnyString(item, "endTime"),
			"repeatType":  firstAnyString(item, "repeatType"),
			"repeatValue": firstAnyString(item, "repeatValue"),
			"status":      firstAnyString(item, "status"),
			"version":     firstAnyString(item, "version"),
			"actionCount": countRows(item["actions"]),
		})
		if len(job) > 0 {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func projectMessageRows(data any) []any {
	rows := rowsFromData(data)
	messages := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		message := compactMap(map[string]any{
			"id":         firstAnyString(item, "id", "messageId"),
			"title":      firstAnyString(item, "title"),
			"type":       firstAnyString(item, "type", "messageType"),
			"status":     firstAnyString(item, "status", "readStatus", "isRead"),
			"createdAt":  firstAnyString(item, "createdAt", "createTime", "time"),
			"targetType": firstAnyString(item, "targetType", "bizType"),
			"targetId":   firstAnyString(item, "targetId", "bizId"),
		})
		if content := firstAnyString(item, "content", "summary", "message"); content != "" {
			message["summary"] = truncateText(content, 160)
		}
		if len(message) > 0 {
			messages = append(messages, message)
		}
	}
	return messages
}

func projectCatalogRows(value any, project func(map[string]any) map[string]any) []any {
	rows := rowsFromData(value)
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		projected := project(item)
		if len(projected) > 0 {
			result = append(result, projected)
		}
	}
	return result
}

func projectProductDomainSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":      firstAnyString(item, "id", "domainId"),
		"name":    firstAnyString(item, "name", "domainName"),
		"code":    firstAnyString(item, "code", "domainCode"),
		"pid":     firstAnyString(item, "pid", "productId"),
		"desc":    firstAnyString(item, "desc", "description"),
		"version": firstAnyString(item, "version"),
	})
}

func projectFAQSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":        firstAnyString(item, "id", "faqId"),
		"pid":       firstAnyString(item, "pid", "productId"),
		"title":     firstAnyString(item, "title", "question", "name"),
		"type":      firstAnyString(item, "type", "faqType"),
		"locale":    firstAnyString(item, "locale", "languageCode"),
		"status":    firstAnyString(item, "status"),
		"answer":    truncateText(firstAnyString(item, "answer", "content"), 240),
		"itemCount": countRows(firstNonNil(item["items"], item["faqItems"])),
	})
}

func projectCodeDescriptionSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"code":          firstAnyString(item, "code", "value", "id"),
		"description":   firstAnyString(item, "description", "name", "label"),
		"enDescription": firstAnyString(item, "enDescription", "englishDescription", "enName"),
	})
}

func projectLocaleSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"code":        firstAnyString(item, "code", "languageCode", "locale"),
		"name":        firstAnyString(item, "name", "description", "languageName"),
		"nativeName":  firstAnyString(item, "nativeName", "localName"),
		"description": firstAnyString(item, "description"),
	})
}

func projectThingSchemaV2Data(value any, shape string) any {
	switch shape {
	case "category_list":
		return projectCatalogRows(value, projectThingCategorySummary)
	case "component_list":
		return projectCatalogRows(value, projectThingComponentSummary)
	case "component_detail":
		if item, ok := value.(map[string]any); ok {
			return projectThingComponentSummary(item)
		}
		return projectCatalogRows(value, projectThingComponentSummary)
	case "property_list":
		return projectCatalogRows(value, projectThingPropertySummary)
	default:
		return sanitizeCloudData(value)
	}
}

func projectThingCategorySummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":     firstAnyString(item, "id", "cid", "categoryId"),
		"name":   firstAnyString(item, "name", "categoryName"),
		"code":   firstAnyString(item, "code", "categoryCode"),
		"pid":    firstAnyString(item, "pid", "productId"),
		"status": firstAnyString(item, "status"),
	})
}

func projectThingComponentSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":            firstAnyString(item, "id", "cid", "componentId"),
		"name":          firstAnyString(item, "name", "componentName"),
		"code":          firstAnyString(item, "code", "componentCode"),
		"type":          firstAnyString(item, "type", "componentType"),
		"propertyCount": countRows(firstNonNil(item["properties"], item["props"])),
		"eventCount":    countRows(item["events"]),
		"actionCount":   countRows(item["actions"]),
	})
}

func projectThingPropertySummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		"id":       firstAnyString(item, "id", "propertyId"),
		"name":     firstAnyString(item, "name", "propertyName"),
		"code":     firstAnyString(item, "code", "propertyCode", "identifier"),
		"type":     firstAnyString(item, "type", "dataType", "format"),
		"unit":     firstAnyString(item, "unit"),
		"readable": firstAnyString(item, "readable", "read"),
		"writable": firstAnyString(item, "writable", "write"),
	})
}

func truncateText(value string, maxLength int) string {
	text := strings.TrimSpace(value)
	if maxLength <= 0 || len([]rune(text)) <= maxLength {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxLength])
}
