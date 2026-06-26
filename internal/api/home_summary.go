package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HomeSummaryCredentials struct {
	Authorization string
	ClientID      string
}

type HouseSummary struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Icon     string         `json:"icon,omitempty"`
	Desc     string         `json:"desc,omitempty"`
	AreaCode string         `json:"areaCode,omitempty"`
	AreaName string         `json:"areaName,omitempty"`
	Counts   map[string]int `json:"counts,omitempty"`
	Source   string         `json:"-"`
}

type HomeSummaryResult struct {
	Region     string         `json:"region"`
	HouseCount int            `json:"houseCount"`
	Houses     []HouseSummary `json:"houses"`
	RawShape   string         `json:"rawShape"`
	APICalls   int            `json:"apiCalls"`
	Source     string         `json:"source,omitempty"`
}

type HomeSummaryClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeSummaryClient(endpoint Endpoint, client *http.Client) HomeSummaryClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeSummaryClient{endpoint: endpoint, client: client}
}

func (client HomeSummaryClient) Run(ctx context.Context, credentials HomeSummaryCredentials) (HomeSummaryResult, error) {
	response, err := client.callHomeList(ctx, "/v1/house/r/list", credentials)
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(response) {
		return HomeSummaryResult{}, fmt.Errorf("home list returned non-success business response")
	}
	houses := extractHouseSummaries(response)
	return HomeSummaryResult{
		Region:     client.endpoint.Region,
		HouseCount: len(houses),
		Houses:     houses,
		RawShape:   responseDataType(response),
		APICalls:   1,
		Source:     "/v1/house/r/list",
	}, nil
}

func (client HomeSummaryClient) RunList(ctx context.Context, credentials HomeSummaryCredentials) (HomeSummaryResult, error) {
	allResponse, err := client.callHomeList(ctx, "/v1/house/r/all", credentials)
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(allResponse) {
		return HomeSummaryResult{}, fmt.Errorf("home list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(allResponse, "code"), responseScalar(allResponse, "message", "msg"), responseDataType(allResponse))
	}
	houses := extractHouseSummaries(allResponse)
	rawShapes := []string{"/v1/house/r/all:" + responseDataType(allResponse)}
	source := "/v1/house/r/all"
	apiCalls := 1
	if len(houses) == 0 {
		listResponse, err := client.callHomeList(ctx, "/v1/house/r/list", credentials)
		if err != nil {
			return HomeSummaryResult{}, err
		}
		apiCalls++
		if !isBusinessOK(listResponse) {
			return HomeSummaryResult{}, fmt.Errorf("home list fallback returned non-success business response: code=%s message=%s dataType=%s", responseScalar(listResponse, "code"), responseScalar(listResponse, "message", "msg"), responseDataType(listResponse))
		}
		fallbackHouses := extractHouseSummaries(listResponse)
		rawShapes = append(rawShapes, "/v1/house/r/list:"+responseDataType(listResponse))
		if len(fallbackHouses) > 0 {
			houses = fallbackHouses
			source = "/v1/house/r/list"
		} else {
			source = "/v1/house/r/all+/v1/house/r/list"
		}
	}
	return HomeSummaryResult{
		Region:     client.endpoint.Region,
		HouseCount: len(houses),
		Houses:     houses,
		RawShape:   strings.Join(rawShapes, ","),
		APICalls:   apiCalls,
		Source:     source,
	}, nil
}

func (client HomeSummaryClient) callHomeList(ctx context.Context, path string, credentials HomeSummaryCredentials) (map[string]any, error) {
	return callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+path, map[string]any{}, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
}

func (client HomeSummaryClient) RunSearch(ctx context.Context, parameters map[string]any, credentials HomeSummaryCredentials) (HomeSummaryResult, error) {
	fuzzyName := strings.TrimSpace(firstNonEmpty(
		stringFromAny(parameters["fuzzyName"]),
		stringFromAny(parameters["name"]),
		stringFromAny(parameters["keyword"]),
		stringFromAny(parameters["query"]),
	))
	if fuzzyName == "" {
		return HomeSummaryResult{}, fmt.Errorf("home search requires fuzzyName or name")
	}
	body := map[string]any{
		"fuzzyName": fuzzyName,
		"pageNo":    positiveInt(firstNonNil(parameters["pageNo"], parameters["page"]), 1),
		"pageSize":  positiveInt(firstNonNil(parameters["pageSize"], parameters["size"], parameters["limit"]), 20),
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/fuzzy", body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(response) {
		return HomeSummaryResult{}, fmt.Errorf("home search returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	houses := extractHouseSummaries(response)
	return HomeSummaryResult{
		Region:     client.endpoint.Region,
		HouseCount: len(houses),
		Houses:     houses,
		RawShape:   responseDataType(response),
		APICalls:   1,
	}, nil
}

func extractHouseSummaries(response map[string]any) []HouseSummary {
	rows := houseRowsFromAny(response["data"])
	houses := make([]HouseSummary, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		house := HouseSummary{
			ID:       firstString(item, "id", "houseId"),
			Name:     firstString(item, "name", "houseName"),
			Icon:     firstString(item, "icon", "img"),
			Desc:     firstString(item, "desc", "description"),
			AreaCode: firstString(item, "areaCode"),
			AreaName: firstString(item, "areaName"),
		}
		if house.ID == "" {
			continue
		}
		house.Counts = houseCountProjection(item)
		houses = append(houses, house)
	}
	return houses
}

func houseRowsFromAny(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		if looksLikeHouseRow(typed) {
			return []any{typed}
		}
		for _, key := range []string{"rows", "list", "data", "records", "items", "result", "page", "houses", "houseList", "house_list", "content", "results", "dataList"} {
			if rows := houseRowsFromAny(typed[key]); len(rows) > 0 {
				return rows
			}
		}
	}
	return nil
}

func looksLikeHouseRow(value map[string]any) bool {
	if firstString(value, "id", "houseId") == "" {
		return false
	}
	return firstString(value, "name", "houseName") != "" ||
		firstString(value, "areaCode", "areaName", "icon", "img", "desc", "description") != "" ||
		hasAnyHouseKey(value, "roomNum", "deviceNum", "gatewayNum", "sceneNum", "automationNum", "areaNum")
}

func hasAnyHouseKey(value map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := value[key]; ok {
			return true
		}
	}
	return false
}

func houseCountProjection(item map[string]any) map[string]int {
	counts := map[string]int{}
	for outputKey, inputKey := range map[string]string{
		"rooms":           "roomNum",
		"devices":         "deviceNum",
		"unboundDevices":  "unbindDeviceNum",
		"gateways":        "gatewayNum",
		"unboundGateways": "unbindGatewayNum",
		"scenes":          "sceneNum",
		"automations":     "automationNum",
		"areas":           "areaNum",
	} {
		if value, ok := item[inputKey]; ok {
			counts[outputKey] = intFromAny(value)
		}
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			text := stringFromAny(value)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func positiveInt(value any, fallback int) int {
	text := positiveIntString(value, fallback)
	if parsed, err := strconv.Atoi(text); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}
