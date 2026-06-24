package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

func (client MetadataReadonlyClient) RunGeoAreaChildrenList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	parentID := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["parentId"]),
		stringFromAny(request.Parameters["parentID"]),
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
			"areas": projectGeoAreaRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunGeoAreaSearch(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	name := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["name"]),
		stringFromAny(request.Parameters["keyword"]),
		stringFromAny(request.Parameters["areaName"]),
	))
	if name == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "geo_area.search", "geo_area_name_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/area/r/areas", map[string]any{"name": name}, request.Credentials)
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
			"areas": projectGeoAreaRows(response["data"]),
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
		copyGeoAreaField(area, item, "id", "id")
		copyGeoAreaField(area, item, "name", "name")
		copyGeoAreaField(area, item, "fullName", "fullname")
		copyGeoAreaField(area, item, "fullName", "fullName")
		copyGeoAreaField(area, item, "code", "code")
		copyGeoAreaField(area, item, "parentId", "parentId")
		copyGeoAreaField(area, item, "level", "level")
		copyGeoAreaField(area, item, "fetchWeather", "fetchWeather")
		copyGeoAreaField(area, item, "leaf", "leaf")
		copyGeoAreaField(area, item, "lanCode", "lanCode")
		copyGeoAreaField(area, item, "latitude", "latitude")
		copyGeoAreaField(area, item, "longitude", "longitude")
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
