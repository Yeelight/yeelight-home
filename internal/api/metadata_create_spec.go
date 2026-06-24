package api

import "net/http"

type metadataCreateSpec struct {
	kind       MetadataKind
	listMethod string
	listPath   string
	listBody   map[string]any
	singlePage bool
	createPath string
	source     string
}

type metadataSummary struct {
	ID     string
	Name   string
	Source string
}

var metadataCreateSpecs = map[MetadataKind]metadataCreateSpec{
	MetadataKindArea: {
		kind:       MetadataKindArea,
		listMethod: http.MethodGet,
		listPath:   "/v2/thing/manage/house/{houseId}/area/r/info/{pageNo}/100",
		createPath: "/v2/thing/manage/house/{houseId}/area/w/create",
		source:     "area_list",
	},
	MetadataKindGroup: {
		kind:       MetadataKindGroup,
		listMethod: http.MethodGet,
		listPath:   "/v2/thing/manage/house/{houseId}/group/r/info/{pageNo}/100",
		createPath: "/v2/thing/manage/house/{houseId}/group/w/create",
		source:     "group_list",
	},
	MetadataKindScene: {
		kind:       MetadataKindScene,
		listMethod: http.MethodPost,
		listPath:   "/v2/thing/manage/house/{houseId}/scene/r/info/{pageNo}/100",
		listBody:   map[string]any{},
		createPath: "/v2/thing/manage/house/{houseId}/scene/w/create",
		source:     "scene_list",
	},
	MetadataKindAutomation: {
		kind:       MetadataKindAutomation,
		listMethod: http.MethodPost,
		listPath:   "/v1/automations/r/list",
		singlePage: true,
		createPath: "/v2/thing/manage/house/{houseId}/automation/w/create",
		source:     "automation_list",
	},
}
