package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (client MetadataReadonlyClient) RunGeoAreaChildrenList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	parentID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldParentID]),
	))
	if parentID == "" {
		parentID = "0"
	}
	if _, err := strconv.Atoi(parentID); err != nil {
		return metadataReadonlyMissingContext(client.endpoint.Region, "geo_area.children.list", "geo_area_parent_id_invalid"), nil
	}
	response, err := client.call(ctx, http.MethodGet, "/v1/area/r/"+pathSegment(parentID)+"/children", nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("geo area children", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		Capability: "geo_area.children.list",
		Data: map[string]any{
			semantic.FieldAreas: projectGeoAreaRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGeoAreaSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	name := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldName]),
		stringFromAny(request.Parameters[semantic.FieldKeyword]),
		stringFromAny(request.Parameters[semantic.FieldAreaName]),
	))
	if name == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "geo_area.search", "geo_area_name_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/area/r/areas", map[string]any{semantic.FieldName: name}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("geo area search", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		Capability: "geo_area.search",
		Data: map[string]any{
			semantic.FieldAreas: projectGeoAreaRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectGeoAreaRows(data any) []any {
	rows := rowsFromData(data)
	areas := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		area := map[string]any{}
		copyGeoAreaField(area, item, semantic.FieldID, "id")
		copyGeoAreaField(area, item, semantic.FieldName, "name")
		copyGeoAreaField(area, item, semantic.FieldFullName, "fullname")
		copyGeoAreaField(area, item, semantic.FieldFullName, "fullName")
		copyGeoAreaField(area, item, semantic.FieldCode, "code")
		copyGeoAreaField(area, item, semantic.FieldParentID, "parentId")
		copyGeoAreaField(area, item, semantic.FieldLevel, "level")
		copyGeoAreaField(area, item, semantic.FieldFetchWeather, "fetchWeather")
		copyGeoAreaField(area, item, semantic.FieldLeaf, "leaf")
		copyGeoAreaField(area, item, semantic.FieldLanguageCode, "lanCode")
		copyGeoAreaField(area, item, semantic.FieldLatitude, "latitude")
		copyGeoAreaField(area, item, semantic.FieldLongitude, "longitude")
		if len(area) > 0 {
			areas = append(areas, area)
		}
	}
	return areas
}

func copyGeoAreaField(target map[string]any, source map[string]any, outputKey string, inputKey string) {
	if _, exists := target[outputKey]; exists {
		return
	}
	value, ok := source[inputKey]
	if !ok {
		return
	}
	target[outputKey] = sanitizeCloudData(value)
}
