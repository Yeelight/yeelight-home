package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type HomeSummaryCredentials struct {
	Authorization string
	ClientID      string
	BizType       string
}

type HouseSummary struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Icon     string         `json:"icon,omitempty"`
	Desc     string         `json:"description,omitempty"`
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
	credentials.BizType = effectiveBizType(ctx, credentials.BizType)
	if credentials.BizType == BizTypeCommercial {
		return client.runCommercialList(ctx, credentials)
	}
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
	credentials.BizType = effectiveBizType(ctx, credentials.BizType)
	if credentials.BizType == BizTypeCommercial {
		return client.runCommercialList(ctx, credentials)
	}
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
		BizType:       credentials.BizType,
	})
}

func (client HomeSummaryClient) RunSearch(ctx context.Context, parameters map[string]any, credentials HomeSummaryCredentials) (HomeSummaryResult, error) {
	credentials.BizType = effectiveBizType(ctx, credentials.BizType)
	fuzzyName := strings.TrimSpace(firstNonEmpty(
		stringFromAny(parameters[semantic.FieldFuzzyName]),
		stringFromAny(parameters[semantic.FieldName]),
		stringFromAny(parameters[semantic.FieldKeyword]),
		stringFromAny(parameters[semantic.FieldQuery]),
	))
	if fuzzyName == "" {
		return HomeSummaryResult{}, fmt.Errorf("home search requires fuzzyName or name")
	}
	if credentials.BizType == BizTypeCommercial {
		summary, err := client.runCommercialList(ctx, credentials)
		if err != nil {
			return HomeSummaryResult{}, err
		}
		ranked := semantic.RankNameMatches(fuzzyName, summary.Houses, func(house HouseSummary) string { return house.Name })
		summary.Houses = make([]HouseSummary, 0, len(ranked))
		for _, match := range ranked {
			house := match.Value
			house.Source = match.Match.Kind
			summary.Houses = append(summary.Houses, house)
		}
		summary.HouseCount = len(summary.Houses)
		summary.Source += "+local_name_match"
		return summary, nil
	}
	body := map[string]any{
		semantic.FieldFuzzyName: fuzzyName,
		semantic.FieldPageNo:    positiveInt(parameters[semantic.FieldPageNo], 1),
		semantic.FieldPageSize:  positiveInt(firstNonNil(parameters[semantic.FieldPageSize], parameters[semantic.FieldLimit]), 20),
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/fuzzy", body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
		BizType:       credentials.BizType,
	})
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(response) {
		return HomeSummaryResult{}, fmt.Errorf("home search returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	houses := extractHouseSummaries(response)
	apiCalls := 1
	source := "/v1/house/r/fuzzy"
	rawShape := responseDataType(response)
	if len(houses) == 0 {
		fallback, calls, err := client.findHouseSearchFallback(ctx, fuzzyName, credentials)
		apiCalls += calls
		if err != nil {
			return HomeSummaryResult{}, err
		}
		if len(fallback) > 0 {
			houses = fallback
			source = "/v1/house/r/fuzzy+local_name_match"
			rawShape += ",local_name_match"
		}
	}
	return HomeSummaryResult{
		Region:     client.endpoint.Region,
		HouseCount: len(houses),
		Houses:     houses,
		RawShape:   rawShape,
		APICalls:   apiCalls,
		Source:     source,
	}, nil
}

func (client HomeSummaryClient) findHouseSearchFallback(ctx context.Context, name string, credentials HomeSummaryCredentials) ([]HouseSummary, int, error) {
	summary, err := client.RunList(ctx, credentials)
	if err != nil {
		return nil, summary.APICalls, err
	}
	ranked := semantic.RankNameMatches(name, summary.Houses, func(house HouseSummary) string {
		return house.Name
	})
	houses := make([]HouseSummary, 0, len(ranked))
	for _, match := range ranked {
		house := match.Value
		house.Source = match.Match.Kind
		houses = append(houses, house)
	}
	return houses, summary.APICalls, nil
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
			ID:       firstString(item, semantic.FieldID, semantic.FieldHouseID),
			Name:     firstString(item, semantic.FieldName, semantic.FieldHouseName),
			Icon:     firstString(item, semantic.HouseSummaryIconFields()...),
			Desc:     firstString(item, semantic.HouseSummaryDescriptionFields()...),
			AreaCode: firstString(item, semantic.FieldAreaCode),
			AreaName: firstString(item, semantic.FieldAreaName),
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
		for _, key := range semantic.ResponseRowsContainers() {
			if rows := houseRowsFromAny(typed[key]); len(rows) > 0 {
				return rows
			}
		}
	}
	return nil
}

func looksLikeHouseRow(value map[string]any) bool {
	if firstString(value, semantic.FieldID, semantic.FieldHouseID) == "" {
		return false
	}
	return firstString(value, semantic.FieldName, semantic.FieldHouseName) != "" ||
		firstString(value, semantic.HouseSummaryPresenceFields()...) != "" ||
		hasAnyHouseKey(value, semantic.HouseSummaryCountFields()...)
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
	for _, mapping := range semantic.HouseSummaryCountMappings() {
		for _, inputKey := range mapping.Internal {
			if value, ok := item[inputKey]; ok {
				counts[mapping.Public] = intFromAny(value)
				break
			}
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
