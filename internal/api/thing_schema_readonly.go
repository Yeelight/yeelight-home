package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunThingSchemaList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaPath(ctx, request, "thing.schema.list", "/v1/thing/schema/r/list", map[string]any{semantic.FieldProducts: nil}, "product_list")
}

func (client MetadataReadonlyClient) RunThingSchemaDetailList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaPath(ctx, request, "thing.schema.detail.list", "/v1/thing/schema/r/list/detail", map[string]any{semantic.FieldSchemas: nil}, "schema_detail_list")
}

func (client MetadataReadonlyClient) RunThingSchemaGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	productID := productIDFromReadonlyRequest(request)
	if productID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.schema.get", "product_context_missing"), nil
	}
	result, err := client.readThingSchemaPath(ctx, request, "thing.schema.get", "/v1/thing/schema/r/"+pathSegment(productID), map[string]any{semantic.FieldSchema: nil}, "schema_detail")
	result.Data = withThingSchemaProductID(result.Data, productID)
	return result, err
}

func (client MetadataReadonlyClient) RunThingSchemaEventList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	productID := productIDFromReadonlyRequest(request)
	if productID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.schema.event.list", "product_context_missing"), nil
	}
	result, err := client.readThingSchemaPath(ctx, request, "thing.schema.event.list", "/v1/thing/schema/r/getEvents/"+pathSegment(productID), map[string]any{semantic.FieldEvents: nil}, "event_list")
	result.Data = withThingSchemaProductID(result.Data, productID)
	return result, err
}

func (client MetadataReadonlyClient) RunThingProductInfoBatchGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	pids := productIDsFromReadonlyRequest(request)
	if pids == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.product.info.batch_get", "product_context_missing"), nil
	}
	path := "/v2/thing/schema/product/r/info?" + readonlyQueryValues(map[string]any{semantic.InternalProductIDsField(): pids})
	result, err := client.readThingSchemaPath(ctx, request, "thing.product.info.batch_get", path, map[string]any{semantic.FieldProducts: nil}, "schema_detail_list")
	result.Data = withThingSchemaProductID(result.Data, pids)
	return result, err
}

func (client MetadataReadonlyClient) RunThingProductInfoV3BatchGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	pids := productIDsFromReadonlyRequest(request)
	version := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldVersion]), stringFromAny(request.Parameters[semantic.FieldSchemaVersion])))
	if pids == "" || version == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "thing.product.info.v3.batch_get", "product_version_context_missing"), nil
	}
	path := "/v3/thing/schema/product/r/info?" + readonlyQueryValues(map[string]any{semantic.InternalProductIDsField(): pids, semantic.FieldVersion: version})
	result, err := client.readThingSchemaPath(ctx, request, "thing.product.info.v3.batch_get", path, map[string]any{semantic.FieldProducts: nil}, "schema_detail_list")
	result.Data = withThingSchemaProductID(result.Data, pids)
	return result, err
}

func (client MetadataReadonlyClient) RunThingProductListV3(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	return client.readThingSchemaPath(ctx, request, "thing.product.list.v3", "/v3/thing/schema/product/r/list", map[string]any{semantic.FieldProducts: nil}, "product_list")
}

func (client MetadataReadonlyClient) readThingSchemaPath(ctx context.Context, request MetadataReadonlyRequest, capability string, path string, projection map[string]any, shape string) (MetadataReadonlyResult, error) {
	response, err := client.call(ctx, http.MethodGet, path, nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return metadataReadonlyPartialBusinessResult(client.endpoint.Region, request.HouseID, request.DeviceID, capability, response), nil
	}
	data := map[string]any{
		semantic.FieldSchemaVersion: "cloud-v1",
		semantic.FieldCachePolicy: map[string]any{
			semantic.FieldScope:      "profile_region_product_schema",
			semantic.FieldTTLSeconds: 86400,
			semantic.FieldPersistent: false,
		},
	}
	for key := range projection {
		data[key] = projectThingSchemaData(response["data"], shape)
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

func productIDFromReadonlyRequest(request MetadataReadonlyRequest) string {
	return strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldCapabilityProductID]),
		stringFromAny(request.Parameters[semantic.FieldID]),
	))
}

func productIDsFromReadonlyRequest(request MetadataReadonlyRequest) string {
	if value := productIDListString(request.Parameters[semantic.FieldCapabilityProductIDs]); value != "" {
		return value
	}
	return productIDFromReadonlyRequest(request)
}

func productIDListString(value any) string {
	switch typed := value.(type) {
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(stringFromAny(item)); text != "" {
				items = append(items, text)
			}
		}
		return strings.Join(items, ",")
	case []string:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				items = append(items, text)
			}
		}
		return strings.Join(items, ",")
	default:
		return strings.TrimSpace(stringFromAny(value))
	}
}

func withThingSchemaProductID(data any, productID string) any {
	item, ok := data.(map[string]any)
	if !ok {
		return data
	}
	item[semantic.FieldCapabilityProductID] = productID
	return item
}

func projectThingSchemaData(value any, shape string) any {
	switch shape {
	case "product_list":
		return projectThingSchemaRows(value, projectThingSchemaProductSummary)
	case "schema_detail_list":
		return projectThingSchemaRows(value, projectThingSchemaSummary)
	case "schema_detail":
		if item, ok := value.(map[string]any); ok {
			return projectThingSchemaSummary(item)
		}
		return projectThingSchemaRows(value, projectThingSchemaSummary)
	case "event_list":
		return projectThingSchemaRows(value, projectThingSchemaEventSummary)
	default:
		return sanitizeCloudData(value)
	}
}

func projectThingSchemaRows(value any, project func(map[string]any) map[string]any) []any {
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

func projectThingSchemaProductSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldCapabilityProductID: firstAnyString(item, append(semantic.DeviceCapabilityProductIDFields(), semantic.FieldID)...),
		semantic.FieldName:                firstAnyString(item, append([]string{semantic.FieldName, semantic.FieldProductName}, semantic.DescriptionFields()...)...),
		semantic.FieldCategory:            firstAnyString(item, append([]string{semantic.FieldCategory}, semantic.ThingCategoryNameFields()...)...),
		semantic.FieldStatus:              firstAnyString(item, semantic.FieldStatus),
		semantic.FieldVersion:             firstAnyString(item, semantic.FieldVersion, semantic.FieldSchemaVersion),
		semantic.FieldComponentCount:      countRows(item[semantic.FieldComponents]),
		semantic.FieldEventNo:             countRows(item[semantic.FieldEvents]),
	})
}

func projectThingSchemaSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldCapabilityProductID: firstAnyString(item, append(semantic.DeviceCapabilityProductIDFields(), semantic.FieldID)...),
		semantic.FieldName:                firstAnyString(item, append([]string{semantic.FieldName, semantic.FieldProductName}, semantic.DescriptionFields()...)...),
		semantic.FieldStatus:              firstAnyString(item, semantic.FieldStatus),
		semantic.FieldVersion:             firstAnyString(item, semantic.FieldVersion, semantic.FieldSchemaVersion),
		semantic.FieldCapability:          firstAnyString(item, semantic.FieldCapability),
		semantic.FieldEventUnitNum:        firstAnyString(item, semantic.FieldEventUnitNum),
		semantic.FieldComponents:          projectComponents(item[semantic.FieldComponents]),
		semantic.FieldEvents:              projectEvents(item[semantic.FieldEvents]),
		semantic.FieldActions:             projectActions(firstNonNil(item[semantic.FieldSupportActions], item[semantic.FieldActions])),
	})
}

func projectThingSchemaEventSummary(item map[string]any) map[string]any {
	return compactMap(map[string]any{
		semantic.FieldID:          firstAnyString(item, semantic.FieldEventID, semantic.FieldID),
		semantic.FieldType:        firstAnyString(item, semantic.ThingEventTypeFields()...),
		semantic.FieldName:        firstAnyString(item, semantic.FieldName),
		semantic.FieldDescription: firstAnyString(item, semantic.DescriptionFields()...),
		semantic.FieldLevel:       firstAnyString(item, semantic.FieldLevel),
		semantic.FieldParameters:  projectProperties(item[semantic.InternalAutomationParamsField()]),
	})
}

func compactMap(values map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range values {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				result[key] = typed
			}
		case []PropertyCapability:
			if len(typed) > 0 {
				result[key] = typed
			}
		case []ComponentCapability:
			if len(typed) > 0 {
				result[key] = typed
			}
		case []EventCapability:
			if len(typed) > 0 {
				result[key] = typed
			}
		case []ActionCapability:
			if len(typed) > 0 {
				result[key] = typed
			}
		case int:
			if typed > 0 {
				result[key] = typed
			}
		default:
			if value != nil {
				result[key] = value
			}
		}
	}
	return result
}

func countRows(value any) int {
	rows, ok := value.([]any)
	if !ok {
		return 0
	}
	return len(rows)
}
