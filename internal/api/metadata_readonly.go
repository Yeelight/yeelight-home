package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type MetadataReadonlyCredentials struct {
	Authorization string
	ClientID      string
}

type MetadataReadonlyRequest struct {
	HouseID     string
	DeviceID    string
	Utterance   string
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
	response, err := client.call(ctx, http.MethodPost, "/v1/house/r/memberlistV2", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
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
			semantic.FieldMembers: projectMemberRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunHomeMemberCurrentGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	uid := strings.TrimSpace(firstNonEmpty(
		stringFromAny(request.Parameters[semantic.FieldUID]),
		stringFromAny(request.Parameters[semantic.FieldUserID]),
		stringFromAny(request.Parameters[semantic.FieldMemberID]),
	))
	apiCalls := 0
	if houseID != "" && uid == "" {
		currentUID, calls, err := NewHomeMemberClient(client.endpoint, client.client).CurrentUserID(ctx, HomeMemberCredentials{
			Authorization: request.Credentials.Authorization,
			ClientID:      request.Credentials.ClientID,
		})
		apiCalls += calls
		if err != nil {
			return MetadataReadonlyResult{}, err
		}
		uid = currentUID
	}
	if houseID == "" || uid == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "home.member.current.get", "member_context_missing")
		result.HouseID = houseID
		return result, nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/house/r/memberinfoV2", map[string]any{semantic.FieldHouseID: houseID, semantic.FieldUID: uid}, request.Credentials)
	apiCalls++
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
			semantic.FieldMembers: projectMemberRows(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: apiCalls,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) RunFavoriteList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "favorite.list", "house_context_missing"), nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/favourite/r/all", map[string]any{semantic.FieldHouseID: houseID}, request.Credentials)
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
			semantic.FieldFavorites: projectFavoriteRows(response["data"]),
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
	result.Data.(map[string]any)[semantic.FieldDetail] = projectPanelDetail(detail["data"])
	result.RawShape = "detail:" + responseDataType(detail)

	buttons, err := client.call(ctx, http.MethodPost, "/v1/panel/r/button/info/"+deviceID, nil, request.Credentials)
	result.APICalls++
	if err != nil || !isBusinessOK(buttons) {
		result.Partial = true
		result.Warnings = append(result.Warnings, "panel_button_read_unavailable")
		return result, nil
	}
	result.Data.(map[string]any)[semantic.FieldButtons] = projectPanelButtons(buttons["data"])
	result.RawShape += ",buttons:" + responseDataType(buttons)
	return result, nil
}

func (client MetadataReadonlyClient) RunPanelList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "panel.list", "house_context_missing"), nil
	}
	return client.readPath(ctx, request, "panel.list", "/v1/panel/r/list/"+pathSegment(houseID), http.MethodGet, nil, map[string]any{semantic.FieldPanels: nil})
}

func (client MetadataReadonlyClient) RunPanelButtonTypeGet(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "panel.button.type.get", "device_context_missing"), nil
	}
	buttonType := strings.TrimSpace(firstNonEmpty(stringFromAny(request.Parameters[semantic.FieldButtonType]), stringFromAny(request.Parameters[semantic.FieldType])))
	if buttonType == "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "panel.button.type.get", "button_type_context_missing")
		result.DeviceID = deviceID
		result.HouseID = strings.TrimSpace(request.HouseID)
		return result, nil
	}
	response, err := client.call(ctx, http.MethodPost, "/v1/panel/r/button/info/"+pathSegment(deviceID)+"/"+pathSegment(buttonType), nil, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{}, metadataReadonlyBusinessError("panel.button.type.get", response)
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    strings.TrimSpace(request.HouseID),
		DeviceID:   deviceID,
		Capability: "panel.button.type.get",
		Data: map[string]any{
			semantic.FieldButtons: projectPanelButtons(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func projectPanelDetail(data any) map[string]any {
	detail := map[string]any{}
	if devices := projectDeviceRows(data); len(devices) > 0 {
		if summary, ok := devices[0].(map[string]any); ok {
			for key, value := range summary {
				detail[key] = value
			}
		}
	}
	removePanelDeviceSummaryFields(detail)
	item, ok := data.(map[string]any)
	if !ok {
		return detail
	}
	copyPanelAnyFields(detail, item, semantic.FieldID, semantic.FieldDeviceID, semantic.FieldName, semantic.FieldAlias, semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldGatewayDeviceID, semantic.FieldCapabilityProductID, semantic.FieldProductComponentID, semantic.FieldTypeName)
	if online, ok := boolFromAny(firstCloudAny(item, semantic.FieldOnline, "isOnline")); ok {
		detail[semantic.FieldOnline] = online
	}
	if bind, ok := boolFromAny(firstCloudAny(item, semantic.FieldBind, "isBind")); ok {
		detail[semantic.FieldBind] = bind
	}
	if virtual, ok := boolFromAny(firstCloudAny(item, semantic.FieldVirtual, "isVirtual")); ok {
		detail[semantic.FieldVirtual] = virtual
	}
	if text := stringFromAny(item[semantic.FieldMAC]); text != "" {
		detail[semantic.FieldMacMasked] = maskTail(text, 4)
	}
	if buttons := projectPanelButtons(firstCloudAny(item, semantic.FieldButtons)); buttons != nil {
		detail[semantic.FieldButtons] = buttons
	}
	return detail
}

func projectPanelButtons(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []any:
		return projectPanelButtonRows(typed)
	case map[string]any:
		result := map[string]any{}
		for key, raw := range typed {
			result[key] = projectPanelButtons(raw)
		}
		return result
	default:
		rows := rowsFromData(value)
		if len(rows) == 0 {
			return nil
		}
		return projectPanelButtonRows(rows)
	}
}

func projectPanelButtonRows(rows []any) []any {
	buttons := make([]any, 0, len(rows))
	for _, row := range rows {
		button := projectPanelButton(row)
		if len(button) > 0 {
			buttons = append(buttons, button)
		}
	}
	return buttons
}

func projectPanelButton(value any) map[string]any {
	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	button := map[string]any{}
	copyPanelAnyFields(button, item, semantic.FieldID, semantic.FieldDeviceID, semantic.FieldName, semantic.FieldAlias, semantic.FieldKeyValue, semantic.FieldIndex, semantic.FieldVisible, semantic.FieldIcon, semantic.FieldExtend, semantic.FieldTargetID, semantic.FieldTargetType, semantic.FieldTargetName, semantic.FieldStatus)
	copyPanelRenamedField(button, item, semantic.FieldSort, semantic.FieldRank)
	copyPanelRenamedField(button, item, semantic.FieldType, semantic.FieldButtonType)
	copyPanelRenamedField(button, item, semantic.FieldValid, semantic.FieldAvailable)
	if targetType := cloudResourceTypeName(item); targetType != "" && button[semantic.FieldTargetType] == nil {
		button[semantic.FieldTargetType] = targetType
	}
	if targetID := firstCloudAny(item, "resId", "resID"); targetID != nil && button[semantic.FieldTargetID] == nil {
		button[semantic.FieldTargetID] = sanitizeCloudData(targetID)
	}
	if events := projectPanelButtonEvents(firstCloudAny(item, semantic.FieldButtonEvents)); len(events) > 0 {
		button[semantic.FieldButtonEvents] = events
	}
	return button
}

func projectPanelButtonEvents(value any) []any {
	rows := rowsFromData(value)
	events := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		event := map[string]any{}
		copyPanelAnyFields(event, item, semantic.FieldID, semantic.FieldButtonEventID, semantic.FieldName, semantic.FieldAlias, semantic.FieldStatus)
		copyPanelRenamedField(event, item, semantic.FieldType, semantic.FieldEventTypeID)
		copyPanelRenamedField(event, item, semantic.FieldValid, semantic.FieldAvailable)
		if event[semantic.FieldButtonEventID] == nil {
			if id := firstCloudAny(item, semantic.FieldID); id != nil {
				event[semantic.FieldButtonEventID] = sanitizeCloudData(id)
			}
		}
		if event[semantic.FieldEventTypeID] == nil {
			if eventType := firstCloudAny(item, "buttonEventType", "eventType"); eventType != nil {
				event[semantic.FieldEventTypeID] = sanitizeCloudData(eventType)
			}
		}
		if actions, ok := mapListFromAny(firstCloudAny(item, semantic.FieldActions, semantic.FieldDetails)); ok {
			publicActions := make([]any, 0, len(actions))
			for _, action := range actions {
				publicActions = append(publicActions, semantic.ToPublicAction(action))
			}
			event[semantic.FieldActions] = publicActions
		}
		if len(event) > 0 {
			events = append(events, event)
		}
	}
	return events
}

func copyPanelAnyFields(output map[string]any, item map[string]any, keys ...string) {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			output[key] = sanitizeCloudData(value)
		}
	}
}

func copyPanelRenamedField(output map[string]any, item map[string]any, sourceKey string, targetKey string) {
	if value, ok := item[sourceKey]; ok && value != nil {
		output[targetKey] = sanitizeCloudData(value)
	}
}

func removePanelDeviceSummaryFields(detail map[string]any) {
	removeDeviceDetailInternalFields(detail)
	delete(detail, semantic.FieldType)
}

func (client MetadataReadonlyClient) RunScreenControlList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "screen.control.list", "house_context_missing"), nil
	}
	deviceID := strings.TrimSpace(request.DeviceID)
	if deviceID != "" {
		result, err := client.readPath(ctx, request, "screen.control.list", "/v1/ai/"+pathSegment(houseID)+"/"+pathSegment(deviceID)+"/control/r/info", http.MethodPost, nil, map[string]any{semantic.FieldControls: nil})
		result.DeviceID = deviceID
		return result, err
	}
	body := map[string]any{semantic.FieldHouseID: houseID}
	if ids, ok := request.Parameters[semantic.FieldDeviceIDs]; ok {
		body[semantic.FieldDeviceIDs] = ids
	}
	return client.readPath(ctx, request, "screen.control.list", "/v1/ai/"+pathSegment(houseID)+"/control/r/info", http.MethodPost, body, map[string]any{semantic.FieldControls: nil})
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
		result.Data.(map[string]any)[semantic.FieldSingle] = projectKnobDetail(single["data"])
		result.RawShape = "single:" + responseDataType(single)
	} else {
		result.Warnings = append(result.Warnings, "single_knob_read_unavailable")
	}

	multi, err := client.call(ctx, http.MethodGet, "/v1/multi-knob/"+deviceID+"/detail", nil, request.Credentials)
	result.APICalls++
	if err == nil && isBusinessOK(multi) {
		result.Data.(map[string]any)[semantic.FieldMulti] = projectKnobDetail(multi["data"])
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

func projectKnobDetail(data any) map[string]any {
	detail := map[string]any{}
	if devices := projectDeviceRows(data); len(devices) > 0 {
		if summary, ok := devices[0].(map[string]any); ok {
			for key, value := range summary {
				detail[key] = value
			}
		}
	}
	removePanelDeviceSummaryFields(detail)
	item, ok := data.(map[string]any)
	if !ok {
		return detail
	}
	copyPanelAnyFields(detail, item, semantic.FieldID, semantic.FieldDeviceID, semantic.FieldName, semantic.FieldAlias, semantic.FieldHouseID, semantic.FieldRoomID, semantic.FieldGatewayDeviceID, semantic.FieldCapabilityProductID, semantic.FieldProductComponentID, semantic.FieldTypeName, semantic.FieldStatus)
	copyPanelRenamedField(detail, item, semantic.FieldValid, semantic.FieldAvailable)
	if online, ok := boolFromAny(firstCloudAny(item, semantic.FieldOnline, "isOnline")); ok {
		detail[semantic.FieldOnline] = online
	}
	if bind, ok := boolFromAny(firstCloudAny(item, semantic.FieldBind, "isBind")); ok {
		detail[semantic.FieldBind] = bind
	}
	if virtual, ok := boolFromAny(firstCloudAny(item, semantic.FieldVirtual, "isVirtual")); ok {
		detail[semantic.FieldVirtual] = virtual
	}
	if text := stringFromAny(item[semantic.FieldMAC]); text != "" {
		detail[semantic.FieldMacMasked] = maskTail(text, 4)
	}
	if actions, ok := mapListFromAny(firstCloudAny(item, semantic.FieldActions, semantic.FieldDetails)); ok {
		publicActions := make([]any, 0, len(actions))
		for _, action := range actions {
			publicActions = append(publicActions, semantic.ToPublicAction(action))
		}
		detail[semantic.FieldActions] = publicActions
	}
	return detail
}

func (client MetadataReadonlyClient) RunHomeSortList(ctx context.Context, request MetadataReadonlyRequest) (MetadataReadonlyResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return metadataReadonlyMissingContext(client.endpoint.Region, "home.sort.list", "house_context_missing"), nil
	}
	body, warning := NormalizeHomeSortPayload(houseID, request.Parameters)
	if warning != "" {
		result := metadataReadonlyMissingContext(client.endpoint.Region, "home.sort.list", "home_sort_query_context_missing")
		result.HouseID = houseID
		result.Warnings = append(result.Warnings, warning)
		return result, nil
	}
	if body[semantic.FieldType] == nil && body[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetType)] == nil && body[semantic.InternalField(semantic.DomainSort, semantic.FieldTargetID)] == nil && body[semantic.FieldRoomID] == nil {
		body = map[string]any{semantic.FieldHouseID: houseID}
	}
	if result, ok, err := client.runHomeSortSpecificRead(ctx, houseID, body, request.Credentials); ok || err != nil {
		return result, err
	}
	delete(body, semantic.FieldItems)
	response, err := client.call(ctx, http.MethodPost, "/v1/sort/r/getSort", body, request.Credentials)
	if err != nil {
		return MetadataReadonlyResult{}, err
	}
	if !isBusinessOK(response) {
		return MetadataReadonlyResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			Capability: "home.sort.list",
			Data: map[string]any{
				semantic.FieldQuery: sanitizeCloudData(body),
				semantic.FieldBackendEvidence: map[string]any{
					semantic.FieldStatus:  "failed",
					semantic.FieldCode:    responseScalar(response, "code"),
					semantic.FieldMessage: responseScalar(response, "message", "msg"),
				},
			},
			RawShape: responseDataType(response),
			APICalls: 1,
			Partial:  true,
			Warnings: []string{"home_sort_cloud_read_failed"},
		}, nil
	}
	return MetadataReadonlyResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: "home.sort.list",
		Data: map[string]any{
			semantic.FieldSort: sanitizeCloudData(response["data"]),
		},
		RawShape: responseDataType(response),
		APICalls: 1,
		Warnings: []string{},
	}, nil
}

func (client MetadataReadonlyClient) runHomeSortSpecificRead(ctx context.Context, houseID string, body map[string]any, credentials MetadataReadonlyCredentials) (MetadataReadonlyResult, bool, error) {
	sortType := strings.TrimSpace(stringFromAny(body[semantic.FieldType]))
	target := strings.TrimSpace(stringFromAny(body[semantic.FieldTarget]))
	if sortType == "" || target == "" {
		return MetadataReadonlyResult{}, false, nil
	}
	switch sortType {
	case "1":
		response, err := client.callWithHouseHeader(ctx, http.MethodPost, "/v1/node/r/1/"+pathSegment(target)+"/device", nil, credentials, houseID)
		if err != nil {
			return MetadataReadonlyResult{}, false, nil
		}
		if !isBusinessOK(response) {
			return MetadataReadonlyResult{}, false, nil
		}
		sortRows := projectSortedDeviceRows(response["data"])
		if enriched, err := client.enrichSortedDeviceRows(ctx, houseID, sortRows, credentials); err == nil {
			sortRows = enriched
		}
		return MetadataReadonlyResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			Capability: "home.sort.list",
			Data: map[string]any{
				semantic.FieldQuery:    sanitizeCloudData(body),
				semantic.FieldReadback: "node.sorted_device.list",
				semantic.FieldSort:     sortRows,
			},
			RawShape: responseDataType(response),
			APICalls: 1,
			Warnings: []string{},
		}, true, nil
	case "2":
		response, err := client.call(ctx, http.MethodPost, "/v1/sort/r/room/scene", map[string]any{
			semantic.FieldIDs: []any{requestNumberOrStringForAPI(target)},
		}, credentials)
		if err != nil {
			return MetadataReadonlyResult{}, false, nil
		}
		if !isBusinessOK(response) {
			return MetadataReadonlyResult{}, false, nil
		}
		return MetadataReadonlyResult{
			Region:     client.endpoint.Region,
			HouseID:    houseID,
			Capability: "home.sort.list",
			Data: map[string]any{
				semantic.FieldQuery:    sanitizeCloudData(body),
				semantic.FieldReadback: "room.scene.sort",
				semantic.FieldSort:     sanitizeCloudData(response["data"]),
			},
			RawShape: responseDataType(response),
			APICalls: 1,
			Warnings: []string{},
		}, true, nil
	default:
		return MetadataReadonlyResult{}, false, nil
	}
}

func (client MetadataReadonlyClient) call(ctx context.Context, method string, path string, body map[string]any, credentials MetadataReadonlyCredentials) (map[string]any, error) {
	return client.callWithHouseHeader(ctx, method, path, body, credentials, "")
}

func (client MetadataReadonlyClient) callWithHouseHeader(ctx context.Context, method string, path string, body map[string]any, credentials MetadataReadonlyCredentials, houseID string) (map[string]any, error) {
	return callJSON(ctx, client.client, method, strings.TrimRight(client.endpoint.BaseURL, "/")+path, body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
		HouseID:       houseID,
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

func metadataReadonlyPartialBusinessResult(region string, houseID string, deviceID string, capability string, response map[string]any) MetadataReadonlyResult {
	return MetadataReadonlyResult{
		Region:     region,
		HouseID:    strings.TrimSpace(houseID),
		DeviceID:   strings.TrimSpace(deviceID),
		Capability: capability,
		RawShape:   responseDataType(response),
		APICalls:   1,
		Partial:    true,
		Warnings:   []string{"cloud_business_response_not_success"},
	}
}

func metadataReadonlyAuthBoundaryResult(region string, houseID string, deviceID string, capability string, statusCode int) MetadataReadonlyResult {
	return MetadataReadonlyResult{
		Region:     region,
		HouseID:    strings.TrimSpace(houseID),
		DeviceID:   strings.TrimSpace(deviceID),
		Capability: capability,
		Data: map[string]any{
			semantic.FieldHTTPStatus: statusCode,
		},
		RawShape: "<http_auth_boundary>",
		APICalls: 1,
		Partial:  true,
		Warnings: []string{"cloud_authorization_boundary"},
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
		if value := firstAnyString(item, semantic.MemberIDFields()...); value != "" {
			member[semantic.FieldMemberIDMasked] = maskIdentifier(value)
		}
		for _, key := range semantic.AccountDisplayNameFields() {
			if value := firstAnyString(item, key); value != "" {
				member[semantic.FieldDisplayName] = value
				break
			}
		}
		for _, key := range semantic.MemberRoleFields() {
			if value := firstAnyString(item, key); value != "" {
				member[semantic.FieldRole] = value
				break
			}
		}
		if value := firstAnyString(item, semantic.AccountPhoneFields()...); value != "" {
			member[semantic.FieldPhoneMasked] = maskTail(value, 4)
		}
		if value := firstAnyString(item, semantic.AccountEmailFields()...); value != "" {
			member[semantic.FieldEmailMasked] = maskEmail(value)
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
		for _, key := range semantic.MemberRowContainers() {
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
		if propertyMap := firstCloudAny(typed, "propertyMap", "propertiesMap", "property_map"); propertyMap != nil {
			item[semantic.FieldProperties] = sanitizeCloudPropertyMap(propertyMap)
			return item
		}
		if targetType := cloudResourceTypeName(typed); targetType != "" {
			item[semantic.FieldTargetType] = targetType
		}
		if targetID := firstCloudAny(typed, "resId", "resID"); targetID != nil {
			item[semantic.FieldTargetID] = sanitizeCloudData(targetID)
		}
		if targetName := firstCloudAny(typed, "resName", "resname"); targetName != nil {
			item[semantic.FieldTargetName] = sanitizeCloudData(targetName)
		}
		if actions := firstCloudAny(typed, semantic.FieldDetails, semantic.FieldActions); actions != nil {
			item[semantic.FieldActions] = sanitizeCloudData(actions)
		}
		if params := firstCloudAny(typed, "params", "param"); params != nil {
			if publicParams := sanitizeCloudParams(params); publicParams != nil {
				item[semantic.FieldSet] = publicParams
			}
		}
		if productCode := firstCloudAny(typed, "materialCode", "skuMaterialCode"); productCode != nil {
			item[semantic.FieldProductCode] = sanitizeCloudData(productCode)
		}
		if capabilityProductID := firstCloudAny(typed, "pid", "productId"); capabilityProductID != nil {
			item[semantic.FieldCapabilityProductID] = sanitizeCloudData(capabilityProductID)
		}
		if productCategoryID := firstCloudAny(typed, "pcId", "pcid", "categoryId"); productCategoryID != nil {
			item[semantic.FieldProductCategoryID] = sanitizeCloudData(productCategoryID)
		}
		if propertyID := firstCloudAny(typed, "propId", "propertyId"); propertyID != nil {
			if propertyName := semantic.PropertyName(stringFromAny(propertyID)); propertyName != "" {
				item[semantic.FieldProperty] = propertyName
			}
		}
		for key, value := range typed {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if isSensitiveCloudField(normalized) || isInternalCloudField(normalized) {
				continue
			}
			switch {
			case strings.Contains(normalized, "email"):
				if text := stringFromAny(value); text != "" {
					item[semantic.FieldEmailMasked] = maskEmail(text)
				}
			case strings.Contains(normalized, "phone") || strings.Contains(normalized, "mobile"):
				if text := stringFromAny(value); text != "" {
					item[semantic.FieldPhoneMasked] = maskTail(text, 4)
				}
			case normalized == semantic.FieldMAC:
				if text := stringFromAny(value); text != "" {
					item[semantic.FieldMacMasked] = maskTail(text, 4)
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

func sanitizeCloudPropertyMap(value any) any {
	typed, ok := value.(map[string]any)
	if !ok {
		return sanitizeCloudData(value)
	}
	properties := map[string]any{}
	for key, rawValue := range typed {
		propertyID, ok := semantic.PropertyID(key)
		if !ok || semantic.PropertySensitive(propertyID) {
			continue
		}
		publicName := semantic.PropertyName(propertyID)
		if publicName == "" {
			continue
		}
		properties[publicName] = sanitizeCloudData(rawValue)
	}
	return properties
}

func firstCloudAny(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func cloudResourceTypeName(item map[string]any) string {
	for _, key := range []string{"typeId", "resType", "resTypeId"} {
		if value, ok := item[key]; ok {
			if name := semantic.ResourceTypeName(value); name != "" {
				return name
			}
		}
	}
	return ""
}

func sanitizeCloudParams(value any) any {
	switch typed := value.(type) {
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(strings.TrimSpace(typed)), &decoded); err == nil {
			return sanitizeCloudParams(decoded)
		}
		return nil
	case map[string]any:
		if set, ok := typed[semantic.FieldSet].(map[string]any); ok {
			return semantic.ToPublicLightSet(set)
		}
		return sanitizeCloudData(typed)
	default:
		return sanitizeCloudData(value)
	}
}

func isSensitiveCloudField(normalized string) bool {
	compact := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(normalized))
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

func isInternalCloudField(normalized string) bool {
	compact := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(normalized))
	switch compact {
	case "typeid", "restype", "restypeid", "resid", "resname", "params", "param", "details",
		"pid", "pids", "pcid", "materialcode", "componentid", "cid", "propid", "propertyid",
		"p", "l", "ct", "c", "mv", "oc", "dc", "act", "alm", "dt", "pi", "pe", "hk", "mfl", "dver", "dpt", "pf":
		return true
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
