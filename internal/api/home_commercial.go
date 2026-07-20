package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	commercialSaaSRolePath    = "/apis/commercial/saas/v1/user/r/saas-role"
	commercialProjectRolePath = "/apis/commercial/saas/v1/user/r/project-role"
	commercialProjectPagePath = "/apis/commercial/saas/v1/project/r/page"
)

func (client HomeSummaryClient) runCommercialList(ctx context.Context, credentials HomeSummaryCredentials) (HomeSummaryResult, error) {
	credentials.BizType = BizTypeCommercial
	requestCredentials := requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
		BizType:       credentials.BizType,
	}
	baseURL := strings.TrimRight(client.endpoint.AccountBaseURL(), "/")
	roleResponse, err := callJSONBody(ctx, client.client, http.MethodGet, baseURL+commercialSaaSRolePath, nil, requestCredentials)
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(roleResponse) {
		return HomeSummaryResult{}, fmt.Errorf("commercial SaaS role returned non-success business response")
	}
	if strings.EqualFold(strings.TrimSpace(fmt.Sprint(roleResponse["data"])), "commercial_saas_user") {
		return HomeSummaryResult{Region: client.endpoint.Region, Houses: []HouseSummary{}, RawShape: "commercial_saas_user", APICalls: 1, Source: commercialSaaSRolePath}, nil
	}

	projectRoleResponse, err := callJSONBody(ctx, client.client, http.MethodGet, baseURL+commercialProjectRolePath, nil, requestCredentials)
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(projectRoleResponse) {
		return HomeSummaryResult{}, fmt.Errorf("commercial project role returned non-success business response")
	}
	projectResponse, err := callJSON(ctx, client.client, http.MethodPost, baseURL+commercialProjectPagePath, map[string]any{"pageNo": 1, "pageSize": 999}, requestCredentials)
	if err != nil {
		return HomeSummaryResult{}, err
	}
	if !isBusinessOK(projectResponse) {
		return HomeSummaryResult{}, fmt.Errorf("commercial project list returned non-success business response")
	}

	roles := commercialProjectRoles(projectRoleResponse["data"])
	rows := houseRowsFromAny(projectResponse["data"])
	allowed := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		houseID := firstString(item, "houseId", "id", "value")
		if commercialProjectRoleAllowed(roles[houseID]) {
			allowed = append(allowed, item)
		}
	}
	houses := extractHouseSummaries(map[string]any{"data": allowed})
	return HomeSummaryResult{
		Region: client.endpoint.Region, HouseCount: len(houses), Houses: houses,
		RawShape: responseDataType(projectResponse), APICalls: 3, Source: commercialProjectPagePath,
	}, nil
}

func commercialProjectRoles(value any) map[string]any {
	roles, _ := value.(map[string]any)
	if nested, ok := roles["roles"].(map[string]any); ok {
		return nested
	}
	return roles
}

func commercialProjectRoleAllowed(value any) bool {
	if item, ok := value.(map[string]any); ok {
		for _, key := range []string{"role", "roleCode", "roleId", "code", "value"} {
			if nested, exists := item[key]; exists && commercialProjectRoleAllowed(nested) {
				return true
			}
		}
		return false
	}
	if items, ok := value.([]any); ok {
		for _, item := range items {
			if commercialProjectRoleAllowed(item) {
				return true
			}
		}
		return false
	}
	switch strings.TrimSpace(fmt.Sprint(value)) {
	case "1", "2":
		return true
	default:
		return false
	}
}
