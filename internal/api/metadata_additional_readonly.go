package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunSceneScopedList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	roomID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldRoomID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
	))
	if houseID == "" && roomID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "scene.scoped.list", "scene_scope_context_missing"), nil
	}
	body := readonlyBodyFromParameters(request.Parameters, semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldGatewayDeviceID, semantic.FieldPageNo, semantic.FieldPageSize)
	if houseID != "" {
		body[semantic.FieldHouseID] = requestNumberOrStringForAPI(houseID)
	}
	if roomID != "" {
		body[semantic.FieldRoomID] = requestNumberOrStringForAPI(roomID)
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/scene/r/list", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		if fallback, ok := client.runSceneScopedListFallback(ctx, houseID, roomID, request.Credentials); ok {
			fallback.APICalls++
			fallback.Warnings = append(fallback.Warnings, "scene_scoped_list_all_fallback")
			return fallback, nil
		}
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, houseID, "", "scene.scoped.list", response), nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "scene.scoped.list",
		Data: map[string]any{
			semantic.FieldRoomID: roomID,
			semantic.FieldScenes: projectSceneRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) runSceneScopedListFallback(ctx context.Context, houseID string, roomID string, credentials MetadataReadonlyCredentials) (MetadataReadonlyResult, bool) {
	result, err := client.RunSceneList(ctx, MetadataReadonlyRequest{
		HouseID:     houseID,
		Credentials: credentials,
	})
	if err != nil || result.Partial {
		return MetadataReadonlyResult{}, false
	}
	data, ok := result.Data.(map[string]any)
	if !ok {
		return MetadataReadonlyResult{}, false
	}
	scenes, ok := data[semantic.FieldScenes].([]any)
	if !ok {
		return MetadataReadonlyResult{}, false
	}
	filtered := make([]any, 0, len(scenes))
	for _, scene := range scenes {
		item, ok := scene.(map[string]any)
		if !ok {
			continue
		}
		if roomID == "" || strings.TrimSpace(stringFromAny(item[semantic.FieldRoomID])) == roomID {
			filtered = append(filtered, item)
		}
	}
	return MetadataReadonlyResult{
		Region:     result.Region,
		HouseID:    houseID,
		Capability: "scene.scoped.list",
		Data: map[string]any{
			semantic.FieldRoomID: roomID,
			semantic.FieldScenes: filtered,
		},
		RawShape: result.RawShape,
		APICalls: result.APICalls,
		Warnings: []string{},
	}, true
}

func (client MetadataReadonlyClient) RunScheduleJobList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "schedule_job.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/schedulejob/r/list", map[string]any{semantic.FieldHouseID: requestNumberOrStringForAPI(houseID)}, request.Credentials)
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
			semantic.FieldScheduleJobs: projectScheduleJobRows(response["data"]),
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
			semantic.FieldMessages: projectMessageRows(response["data"]),
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
			semantic.FieldDomains: projectCatalogRows(response["data"], projectProductDomainSummary),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunProductFAQList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := readonlyBodyFromParameters(request.Parameters, semantic.FieldCapabilityProductID, semantic.FieldKeyword, semantic.FieldLocale, semantic.FieldLanguageCode, semantic.FieldPageNo, semantic.FieldPageSize)
	productIDField := semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)
	if body[productIDField] == nil {
		if productID := stringFromAny(body[semantic.FieldCapabilityProductID]); productID != "" {
			body[productIDField] = productID
		}
	}
	delete(body, semantic.FieldCapabilityProductID)
	result, err := client.readFAQPath(ctx, request, "thing.product_faq.list", "/v1/platform/thing/product_faq/r/list", http.MethodPost, body, map[string]any{semantic.FieldFAQs: nil})
	return result, err
}

func (client MetadataReadonlyClient) RunProductFAQDetailGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	faqID := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldFAQID]), stringFromAny(request.Parameters[semantic.FieldID])))
	if faqID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.product_faq.detail.get", "faq_context_missing"), nil
	}
	return client.readFAQPath(ctx, request, "thing.product_faq.detail.get", "/v1/platform/thing/product_faq/r/"+pathSegment(faqID)+"/detail", http.MethodGet, nil, map[string]any{semantic.FieldFAQ: nil})
}

func (client MetadataReadonlyClient) RunProductFAQTypeList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readFAQCatalogPath(ctx, request, "thing.product_faq.type.list", "/v1/platform/thing/product_faq/r/faq-types", semantic.FieldFAQTypes)
}

func (client MetadataReadonlyClient) RunProductFAQItemTypeList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readFAQCatalogPath(ctx, request, "thing.product_faq.item_type.list", "/v1/platform/thing/product_faq/r/faq-item-types", semantic.FieldFAQItemTypes)
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
			semantic.FieldLocales: projectCatalogRows(response["data"], projectLocaleSummary),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunProductFAQPageList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := productFAQQueryBody(request.Parameters)
	return client.readFAQPath(ctx, request, "thing.product_faq.page.list", "/v1/platform/thing/product_faq/r/page", http.MethodPost, body, map[string]any{semantic.FieldFAQs: nil})
}

func (client MetadataReadonlyClient) RunProductFAQPageDetailList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	body := productFAQQueryBody(request.Parameters)
	return client.readFAQPath(ctx, request, "thing.product_faq.page_detail.list", "/v1/platform/thing/product_faq/r/pageDetail", http.MethodPost, body, map[string]any{semantic.FieldFAQs: nil})
}

func (client MetadataReadonlyClient) RunThingCategoryList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.category.list", "/v2/thing/schema/category/r/list", map[string]any{semantic.FieldCategories: nil}, "category_list")
}

func (client MetadataReadonlyClient) RunThingComponentList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.component.list", "/v2/thing/schema/component/r/list", map[string]any{semantic.FieldComponents: nil}, "component_list")
}

func (client MetadataReadonlyClient) RunThingComponentGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	componentID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldComponentID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
	))
	componentName := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldComponentName]),
		stringFromAny(request.Parameters[semantic.FieldName]),
		stringFromAny(request.Parameters[semantic.FieldKeyword]),
		stringFromAny(request.Parameters[semantic.FieldQuery]),
	))
	if componentID != "" && !isNumericText(componentID) && componentName == "" {
		componentName = componentID
		componentID = ""
	}
	if componentID == "" && componentName != "" {
		resolvedID, calls, warnings, err := client.resolveThingComponentID(ctx, request, componentName)
		if err != nil {
			return MetadataReadonlyResult{}, err
		}
		if resolvedID == "" {
			result := metadataReadonlyMissingContext(client.endpoint.Region, "thing.component.get", "component_not_found_or_ambiguous")
			result.APICalls = calls
			result.Warnings = append(result.Warnings, warnings...)
			return result, nil
		}
		componentID = resolvedID
	}
	if componentID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.component.get", "component_context_missing"), nil
	}
	result, err := client.readThingSchemaV2Path(ctx, request, "thing.component.get", "/v2/thing/schema/component/r/"+pathSegment(componentID), map[string]any{"component": nil}, "component_detail")
	result.Data = withThingSchemaProductID(result.Data, componentID)
	return result, err
}

func (client MetadataReadonlyClient) resolveThingComponentID(ctx context.Context, request MetadataReadonlyRequest, componentName string) (string, int, []string, error) {
	list, err := client.RunThingComponentList(ctx, request)
	if err != nil {
		return "", list.APICalls, nil, err
	}
	components := thingComponentRows(list)
	ranked := semantic.RankNameMatches(componentName, components, func(item map[string]any) string {
		return firstAnyString(item, semantic.FieldName, semantic.FieldCode, semantic.FieldID)
	})
	if len(ranked) == 0 {
		return "", list.APICalls, []string{"component_name_not_found"}, nil
	}
	if ranked[0].Match.Kind == "name" {
		exact := 0
		id := ""
		for _, item := range ranked {
			if item.Match.Kind == "name" {
				exact++
				id = firstAnyString(item.Value, semantic.FieldID)
			}
		}
		if exact == 1 && id != "" {
			return id, list.APICalls, nil, nil
		}
		return "", list.APICalls, []string{"component_name_ambiguous"}, nil
	}
	second := semantic.NameMatch{}
	if len(ranked) > 1 {
		second = ranked[1].Match
	}
	if semantic.NameMatchAutoAccept(ranked[0].Match, second) {
		if id := firstAnyString(ranked[0].Value, semantic.FieldID); id != "" {
			return id, list.APICalls, nil, nil
		}
	}
	return "", list.APICalls, []string{"component_name_ambiguous"}, nil
}

func thingComponentRows(result MetadataReadonlyResult) []map[string]any {
	data, ok := result.Data.(map[string]any)
	if !ok {
		return nil
	}
	values, ok := data[semantic.FieldComponents].([]any)
	if !ok {
		return nil
	}
	rows := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			rows = append(rows, item)
		}
	}
	return rows
}

func isNumericText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (client MetadataReadonlyClient) RunThingPropertyList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaV2Path(ctx, request, "thing.property.list", "/v2/thing/schema/property/r/list", map[string]any{semantic.FieldProperties: nil}, "property_list")
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
	body := readonlyBodyFromParameters(parameters, semantic.FieldCapabilityProductID, semantic.FieldModuleID, semantic.FieldKeyword, semantic.FieldLocale, semantic.FieldLanguageCode, semantic.FieldPageNo, semantic.FieldPageSize)
	if body[semantic.FieldModuleID] == nil {
		if productID := stringFromAny(body[semantic.FieldCapabilityProductID]); productID != "" {
			body[semantic.FieldModuleID] = productID
		}
	}
	productIDField := semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)
	if body[productIDField] == nil {
		if productID := stringFromAny(body[semantic.FieldCapabilityProductID]); productID != "" {
			body[productIDField] = productID
		}
	}
	delete(body, semantic.FieldCapabilityProductID)
	if body[semantic.FieldPageNo] == nil {
		body[semantic.FieldPageNo] = 1
	}
	if body[semantic.FieldPageSize] == nil {
		body[semantic.FieldPageSize] = 20
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
		semantic.FieldSchemaVersion: "cloud-v2",
		semantic.FieldCachePolicy: map[string]any{
			semantic.FieldScope:      "profile_region_thing_schema_v2",
			semantic.FieldTTLSeconds: 86400,
			semantic.FieldPersistent: false,
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
	rows := nestedRowsFromData(data, semantic.ScheduleJobRowContainers()...)
	jobs := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		job := compactMap(map[string]any{
			semantic.FieldID:          firstAnyString(item, semantic.ScheduleJobIDFields()...),
			semantic.FieldHouseID:     firstAnyString(item, semantic.FieldHouseID),
			semantic.FieldName:        firstAnyString(item, semantic.FieldName),
			semantic.FieldStatus:      firstAnyString(item, semantic.FieldStatus),
			semantic.FieldVersion:     firstAnyString(item, semantic.FieldVersion),
			semantic.FieldActionCount: countRows(item[semantic.FieldActions]),
		})
		start := strings.TrimSpace(firstAnyString(item, semantic.FieldStartTime))
		end := strings.TrimSpace(firstAnyString(item, semantic.FieldEndTime))
		if start != "" || end != "" {
			window := map[string]any{}
			if start != "" {
				window[semantic.FieldStart] = start
			}
			if end != "" {
				window[semantic.FieldEnd] = end
			}
			job[semantic.FieldActiveWindow] = window
		}
		if repeat := publicRepeat(item); repeat != nil {
			job[semantic.FieldRepeat] = repeat
		}
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
			semantic.FieldID:         firstAnyString(item, semantic.MessageIDFields()...),
			semantic.FieldTitle:      firstAnyString(item, semantic.FieldTitle),
			semantic.FieldType:       firstAnyString(item, semantic.MessageTypeFields()...),
			semantic.FieldStatus:     firstAnyString(item, semantic.MessageStatusFields()...),
			semantic.FieldCreatedAt:  firstAnyString(item, semantic.FieldCreatedAt, semantic.FieldCreateTime, semantic.FieldTime),
			semantic.FieldTargetType: firstAnyString(item, semantic.MessageTargetTypeFields()...),
			semantic.FieldTargetID:   firstAnyString(item, semantic.MessageTargetIDFields()...),
		})
		if content := firstAnyString(item, semantic.MessageContentFields()...); content != "" {
			message[semantic.FieldSummary] = truncateText(content, 160)
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
		semantic.FieldID:                  firstAnyString(item, semantic.ProductDomainIDFields()...),
		semantic.FieldName:                firstAnyString(item, semantic.ProductDomainNameFields()...),
		semantic.FieldCode:                firstAnyString(item, semantic.ProductDomainCodeFields()...),
		semantic.FieldCapabilityProductID: firstAnyString(item, semantic.DeviceCapabilityProductIDFields()...),
		semantic.FieldDescription:         firstAnyString(item, semantic.DescriptionFields()...),
		semantic.FieldVersion:             firstAnyString(item, semantic.FieldVersion),
	})
}

func projectFAQSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldID:                  firstAnyString(item, semantic.FieldID, semantic.FieldFAQID),
		semantic.FieldCapabilityProductID: firstAnyString(item, semantic.DeviceCapabilityProductIDFields()...),
		semantic.FieldTitle:               firstAnyString(item, semantic.FAQTitleFields()...),
		semantic.FieldType:                firstAnyString(item, semantic.FAQTypeFields()...),
		semantic.FieldLocale:              firstAnyString(item, semantic.LocaleCodeFields()...),
		semantic.FieldStatus:              firstAnyString(item, semantic.FieldStatus),
		semantic.FieldAnswer:              truncateText(firstAnyString(item, append([]string{semantic.FieldAnswer}, semantic.MessageContentFields()...)...), 240),
		semantic.FieldItemCount:           countRows(firstNonNil(item[semantic.FieldItems], item[semantic.FAQItemsField()])),
	})
}

func projectCodeDescriptionSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldCode:               firstAnyString(item, semantic.FieldCode, semantic.FieldValue, semantic.FieldID),
		semantic.FieldDescription:        firstAnyString(item, semantic.CodeDescriptionFields()...),
		semantic.FieldEnglishDescription: firstAnyString(item, semantic.EnglishDescriptionFields()...),
	})
}

func projectLocaleSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldCode:        firstAnyString(item, semantic.LocaleCodeFields()...),
		semantic.FieldName:        firstAnyString(item, semantic.LocaleNameFields()...),
		semantic.FieldNativeName:  firstAnyString(item, semantic.NativeNameFields()...),
		semantic.FieldDescription: firstAnyString(item, semantic.FieldDescription),
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
		semantic.FieldID:                  firstAnyString(item, semantic.ThingCategoryIDFields()...),
		semantic.FieldName:                firstAnyString(item, semantic.ThingCategoryNameFields()...),
		semantic.FieldCode:                firstAnyString(item, semantic.ThingCategoryCodeFields()...),
		semantic.FieldCapabilityProductID: firstAnyString(item, semantic.DeviceCapabilityProductIDFields()...),
		semantic.FieldStatus:              firstAnyString(item, semantic.FieldStatus),
	})
}

func projectThingComponentSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldID:            firstAnyString(item, append([]string{semantic.FieldID}, semantic.ComponentIDFields()...)...),
		semantic.FieldName:          firstAnyString(item, semantic.FieldName, semantic.FieldComponentName),
		semantic.FieldCode:          firstAnyString(item, semantic.ThingComponentCodeFields()...),
		semantic.FieldType:          firstAnyString(item, semantic.ThingComponentTypeFields()...),
		semantic.FieldPropertyCount: countRows(firstNonNil(item[semantic.FieldProperties], item[semantic.PropsField()])),
		semantic.FieldEventCount:    countRows(item[semantic.FieldEvents]),
		semantic.FieldActionCount:   countRows(item[semantic.FieldActions]),
	})
}

func projectThingPropertySummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldID:       firstAnyString(item, semantic.ThingPropertyIDFields()...),
		semantic.FieldName:     firstAnyString(item, semantic.ThingPropertyNameFields()...),
		semantic.FieldCode:     firstAnyString(item, semantic.ThingPropertyCodeFields()...),
		semantic.FieldType:     firstAnyString(item, semantic.FieldType, semantic.FieldDataType, semantic.FieldFormat),
		semantic.FieldUnit:     firstAnyString(item, semantic.FieldUnit),
		semantic.FieldReadable: firstAnyString(item, semantic.FieldReadable, semantic.FieldRead),
		semantic.FieldWritable: firstAnyString(item, semantic.FieldWritable, semantic.FieldWrite),
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
