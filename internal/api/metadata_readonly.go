package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MetadataReadonlyCredentials struct {
	Authorization string
	ClientID      string
}

type MetadataReadonlyRequest struct {
	HouseID     string
	DeviceID    string
	Parameters  map[string]any
	Credentials MetadataReadonlyCredentials
}

type MetadataReadonlyResult struct {
	Region     string   `json:"region"`
	HouseID    string   `json:"houseId,omitempty"`
	DeviceID   string   `json:"deviceId,omitempty"`
	Capability string   `json:"capability"`
	Data       any      `json:"data,omitempty"`
	RawShape   string   `json:"rawShape"`
	APICalls   int      `json:"apiCalls"`
	Partial    bool     `json:"partial,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

type MetadataReadonlyClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewMetadataReadonlyClient(endpoint Endpoint, client *http.Client) MetadataReadonlyClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return MetadataReadonlyClient{endpoint: endpoint, client: client}
}

func (client MetadataReadonlyClient) RunHomeMemberList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.member.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/house/r/memberlistV2", map[string]any{"houseId": houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("home member list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.member.list",
		Data: map[string]any{
			"members": projectMemberRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunHomeMemberCurrentGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	uid := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters["uid"]),
		stringFromAny(request.Parameters["userId"]),
		stringFromAny(request.Parameters["memberId"]),
	))
	if houseID == "" || uid == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "home.member.current.get", "member_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/house/r/memberinfoV2", map[string]any{"houseId": houseID, "uid": uid}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("home member current", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.member.current.get",
		Data: map[string]any{
			"members": projectMemberRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunFavoriteList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "favorite.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/favourite/r/all", map[string]any{"houseId": houseID}, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("favorite list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "favorite.list",
		Data: map[string]any{
			"favorites": sanitizeCloudData(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunPanelGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "panel.get", "device_context_missing"), nil
	}
	result := MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: "panel.get",
		Data:       map[string]any{},
		Warnings:   []string{},
	}
	detail, err := client.call(ctx, http.MethodPost, "/v1/panel/r/detail/"+deviceID, nil, request.Credentials)
	result.APICalls++
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(detail) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("panel detail", detail)
	}
	result.Data.(map[string]any)["detail"] = sanitizeCloudData(detail["data"])
	result.RawShape = "detail:" + responseDataType(detail)

	buttons, err := client.call(ctx, http.MethodPost, "/v1/panel/r/button/info/"+deviceID, nil, request.Credentials)
	result.APICalls++
	if err != nil || !isBusinessOK(buttons) {
		result.Partial = true
		result.Warnings = append(result.Warnings, "panel_button_read_unavailable")
		return result, nil
	}
	result.Data.(map[string]any)["buttons"] = sanitizeCloudData(buttons["data"])
	result.RawShape += ",buttons:" + responseDataType(buttons)
	return result, nil
}

func (client MetadataReadonlyClient) RunPanelList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "panel.list", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "panel.list", "/v1/panel/r/list/"+pathSegment(houseID), http.MethodGet, nil, map[string]any{"panels": nil})
}

func (client MetadataReadonlyClient) RunPanelButtonTypeGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "panel.button.type.get", "device_context_missing"), nil
	}
	buttonType := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters["buttonType"]), stringFromAny(request.Parameters["type"])))
	if buttonType == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "panel.button.type.get", "button_type_context_missing")
		result.DeviceID = deviceID
		result.HouseID = strings.TrimSpace(request.HouseID)
		return result, nil
	}
	result, err := client.readPath(ctx, request, "panel.button.type.get", "/v1/panel/r/button/info/"+pathSegment(deviceID)+"/"+pathSegment(buttonType), http.MethodPost, nil, map[string]any{"buttons": nil})
	result.DeviceID = deviceID
	return result, err
}

func (client MetadataReadonlyClient) RunScreenControlList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "screen.control.list", "house_context_missing"), nil
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID != "" {
		result, err := client.readPath(ctx, request, "screen.control.list", "/v1/ai/"+pathSegment(houseID)+"/"+pathSegment(deviceID)+"/control/r/info", http.MethodPost, nil, map[string]any{"controls": nil})
		result.DeviceID = deviceID
		return result, err
	}
	body := map[string]any{"houseId": houseID}
	if ids, ok := request.Parameters["deviceIds"]; ok {
		body["deviceIds"] = ids
	}
	return client.readPath(ctx, request, "screen.control.list", "/v1/ai/"+pathSegment(houseID)+"/control/r/info", http.MethodPost, body, map[string]any{"controls": nil})
}

func (client MetadataReadonlyClient) RunKnobGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "knob.get", "device_context_missing"), nil
	}
	result := MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: "knob.get",
		Data:       map[string]any{},
		Warnings:   []string{},
	}
	single, err := client.call(ctx, http.MethodGet, "/v1/knobs/"+deviceID+"/detail", nil, request.Credentials)
	result.APICalls++
	if err == nil && isBusinessOK(single) {
		result.Data.(map[string]any)["single"] = sanitizeCloudData(single["data"])
		result.RawShape = "single:" + responseDataType(single)
	} else {
		result.Warnings = append(result.Warnings, "single_knob_read_unavailable")
	}

	multi, err := client.call(ctx, http.MethodGet, "/v1/multi-knob/"+deviceID+"/detail", nil, request.Credentials)
	result.APICalls++
	if err == nil && isBusinessOK(multi) {
		result.Data.(map[string]any)["multi"] = sanitizeCloudData(multi["data"])
		if result.RawShape == "" {
			result.RawShape = "multi:" + responseDataType(multi)
		} else {
			result.RawShape += ",multi:" + responseDataType(multi)
		}
	} else {
		result.Warnings = append(result.Warnings, "multi_knob_read_unavailable")
	}
	if result.RawShape == "" {
		result.Partial = true
		result.RawShape = "<unavailable>"
	}
	result.Partial = result.Partial || len(result.Warnings) > 0
	return result, nil
}

func (client MetadataReadonlyClient) RunHomeSortList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.sort.list", "house_context_missing"), nil
	}
	body := map[string]any{"houseId": houseID}
	for _, key := range []string{"typeId", "resId", "roomId", "type", "target", "subIndex"} {
		if value, ok := request.Parameters[key]; ok {
			body[key] = value
		}
	}
	if body["typeId"] == nil || body["resId"] == nil || body["roomId"] == nil {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "home.sort.list", "home_sort_query_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/sort/r/getSort", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("home sort list", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.sort.list",
		Data: map[string]any{
			"sort": sanitizeCloudData(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) call(ctx context.Context, method string, path string, body map[string]any, credentials MetadataReadonlyCredentials) (map[string]any, error) {
	return callJSON(ctx, client.client, method, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
}

func metadataReadonlyMissingContext(region string, capability string, warning string) MetadataReadonlyResult {
	return MetadataReadonlyResult{
		Region:     region,
		Capability: capability,
		RawShape:   "<missing_context>",
		APICalls:   0,
		Partial:    true,
		Warnings:   []string{warning},
	}
}

func metadataReadonlyBusinessError(name string, response map[string]any) error {
	return fmt.Errorf("%s returned non-success business response: code=%s message=%s dataType=%s", name, responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
}

func projectMemberRows(data any) []any {
	rows := rowsFromData(data)
	members := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		member := map[string]any{}
		if value := firstAnyString(item, "uid", "userId", "memberId", "id"); value != "" {
			member["memberIdMasked"] = maskIdentifier(value)
		}
		for _, key := range []string{"nickname", "nickName", "name", "displayName", "remark"} {
			if value := firstAnyString(item, key); value != "" {
				member["displayName"] = value
				break
			}
		}
		for _, key := range []string{"role", "userRole", "memberRole"} {
			if value := firstAnyString(item, key); value != "" {
				member["role"] = value
				break
			}
		}
		if value := firstAnyString(item, "phone", "phoneNumber", "mobile", "mobilePhone"); value != "" {
			member["phoneMasked"] = maskTail(value, 4)
		}
		if value := firstAnyString(item, "email", "mail"); value != "" {
			member["emailMasked"] = maskEmail(value)
		}
		members = append(members, member)
	}
	return members
}

func rowsFromData(data any) []any {
	switch value := data.(type) {
	case []any:
		return value
	case map[string]any:
		for _, key := range []string{"rows", "list", "members", "memberList", "data"} {
			if rows, ok := value[key].([]any); ok {
				return rows
			}
		}
		return []any{value}
	default:
		return []any{}
	}
}

func sanitizeCloudData(value any) any {
	switch typed := value.(type) {
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeCloudData(item))
		}
		return items
	case map[string]any:
		item := map[string]any{}
		for key, value := range typed {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if isSensitiveCloudField(normalized) {
				continue
			}
			switch {
			case strings.Contains(normalized, "email"):
				if text := stringFromAny(value); text != "" {
					item[key+"Masked"] = maskEmail(text)
				}
			case strings.Contains(normalized, "phone") || strings.Contains(normalized, "mobile"):
				if text := stringFromAny(value); text != "" {
					item[key+"Masked"] = maskTail(text, 4)
				}
			case normalized == "mac":
				if text := stringFromAny(value); text != "" {
					item["macMasked"] = maskTail(text, 4)
				}
			default:
				item[key] = sanitizeCloudData(value)
			}
		}
		return item
	default:
		return value
	}
}

func isSensitiveCloudField(normalized string) bool {
	compact := strings.NewReplacer("_", "", "-", "", ".", "").Replace(normalized)
	if strings.Contains(compact, "token") || strings.Contains(compact, "secret") ||
		strings.Contains(compact, "password") || strings.Contains(compact, "authorization") ||
		strings.Contains(compact, "cookie") || strings.Contains(compact, "credential") {
		return true
	}
	switch compact {
	case "key", "psk", "pskc", "ltk", "mibk", "midk", "hrbk", "meibk":
		return true
	}
	for _, prefix := range []string{"local", "bind", "device", "access", "private", "shared", "network", "wifi", "api", "app", "miot"} {
		if compact == prefix+"key" {
			return true
		}
	}
	return false
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}
