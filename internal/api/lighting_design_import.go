package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	LightingDesignImportCapability = "lighting.design.import"
	DeviceSlotCreateCapability     = "device.slot.create"

	lightingDesignDefaultGatewayLocalID = int64(1)
	lightingDesignMaxRooms              = 50
	lightingDesignMaxDevices            = 120
	lightingDesignMaxGroups             = 50
	lightingDesignMaxScenes             = 30
	lightingDesignMaxAutomations        = 30
)

type LightingDesignImportCredentials struct {
	Authorization string
	ClientID      string
}

type LightingDesignImportRequest struct {
	HouseID        string
	Intent         string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    LightingDesignImportCredentials
}

type LightingDesignImportResult struct {
	Region     string         `json:"region"`
	HouseID    string         `json:"houseId"`
	Capability string         `json:"capability"`
	Mode       string         `json:"mode"`
	Counts     map[string]int `json:"counts"`
	Mappings   map[string]any `json:"mappings,omitempty"`
	Verified   bool           `json:"verified"`
	VerifiedBy string         `json:"verifiedBy,omitempty"`
	ClearAll   bool           `json:"clearAll,omitempty"`
	APICalls   int            `json:"apiCalls"`
	Warnings   []string       `json:"warnings,omitempty"`
}

type LightingDesignImportClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewLightingDesignImportClient(endpoint Endpoint, client *http.Client) LightingDesignImportClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return LightingDesignImportClient{endpoint: endpoint, client: client}
}

func (client LightingDesignImportClient) Run(ctx context.Context, request LightingDesignImportRequest) (LightingDesignImportResult, error) {
	houseID := strings.TrimSpace(firstNonEmpty(request.HouseID, stringFromMap(request.Payload, "houseId")))
	if houseID == "" {
		return LightingDesignImportResult{}, fmt.Errorf("house id is required")
	}
	credentials := requestCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
		HouseID:       houseID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return LightingDesignImportResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	normalized, err := NormalizeLightingDesignImportPayload(houseID, request.Payload)
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	response, err := callJSONBody(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/design/syncMetadata", normalized, credentials)
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	if !isBusinessOK(response) {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	result := LightingDesignImportResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: firstNonEmpty(strings.TrimSpace(request.Intent), LightingDesignImportCapability),
		Mode:       lightingDesignImportMode(normalized),
		Counts:     lightingDesignImportCounts(normalized),
		Mappings:   lightingDesignImportMappings(response["data"]),
		Verified:   true,
		VerifiedBy: "design.syncMetadata",
		ClearAll:   boolFromMap(normalized, "clearAll"),
		APICalls:   1,
		Warnings:   []string{},
	}
	verified, calls, err := client.verify(ctx, houseID, credentials, result.Counts, request.VerifyAttempts, request.VerifyInterval)
	result.APICalls += calls
	if err != nil {
		return LightingDesignImportResult{}, err
	}
	if !verified {
		return LightingDesignImportResult{}, fmt.Errorf("lighting.design.import write verification mismatch")
	}
	result.VerifiedBy = "entity.list"
	return result, nil
}

func NormalizeLightingDesignImportPayload(houseID string, payload map[string]any) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	if _, err := parseID(houseID, "house id"); err != nil {
		return nil, err
	}
	if normalized, ok, err := normalizedLightingDesignImportPayload(houseID, payload); ok || err != nil {
		return normalized, err
	}
	clearAll := boolFromMap(payload, "clearAll") || boolFromMap(payload, "overwrite")
	rooms, err := normalizeLightingDesignRooms(payload)
	if err != nil {
		return nil, err
	}
	if len(rooms) == 0 {
		return nil, fmt.Errorf("at least one room is required")
	}
	if len(rooms) > lightingDesignMaxRooms {
		return nil, fmt.Errorf("room count exceeds limit %d", lightingDesignMaxRooms)
	}
	gatewayID := int64FromMap(payload, "gatewayLocalId", lightingDesignDefaultGatewayLocalID)
	if gatewayID <= 0 {
		gatewayID = lightingDesignDefaultGatewayLocalID
	}
	localSeq := newLightingDesignLocalSeq()
	normalizedRooms := make([]any, 0, len(rooms))
	devices := []any{}
	deviceGroups := []any{}
	slotsByRoomName := map[string][]lightingDesignSlotRef{}
	roomLocalIDByName := map[string]int64{}
	for _, room := range rooms {
		roomLocalID := int64FromMap(room, "localId", localSeq.next())
		roomName := strings.TrimSpace(firstNonEmpty(stringFromMap(room, "name"), stringFromMap(room, "roomName"), stringFromMap(room, "localName")))
		if roomName == "" {
			return nil, fmt.Errorf("room name is required")
		}
		roomLocalIDByName[roomName] = roomLocalID
		normalizedRooms = append(normalizedRooms, map[string]any{
			"localId":    roomLocalID,
			"localName":  roomName,
			"gatewayIds": []any{gatewayID},
		})
		slotItems, err := normalizeLightingDesignRoomSlots(room)
		if err != nil {
			return nil, err
		}
		if len(slotItems) == 0 {
			continue
		}
		for _, slot := range slotItems {
			count := intFromMap(slot, "quantity", intFromMap(slot, "count", 1))
			if count <= 0 {
				count = 1
			}
			if count > 50 {
				return nil, fmt.Errorf("slot quantity exceeds limit 50")
			}
			baseName := strings.TrimSpace(firstNonEmpty(stringFromMap(slot, "name"), stringFromMap(slot, "type"), stringFromMap(slot, "category"), "设备槽位"))
			productMatch, productCandidates := lightingDesignResolveSlotProduct(slot, baseName)
			for index := 1; index <= count; index++ {
				deviceLocalID := int64FromMap(slot, "localId", localSeq.next())
				if count > 1 || hasMapKey(slot, "quantity") || hasMapKey(slot, "count") {
					deviceLocalID = localSeq.next()
				}
				deviceName := lightingDesignSlotName(slot, baseName, index, count)
				pid := int64FromMap(slot, "pid", -1)
				if pid <= 0 && lightingDesignHasProductIdentity(productMatch) && productMatch.Entry.PID > 0 {
					pid = productMatch.Entry.PID
				}
				connectType := intFromMap(slot, "connectType", -1)
				if !hasMapKey(slot, "connectType") && lightingDesignHasProductIdentity(productMatch) && productMatch.Entry.ConnectType >= -1 {
					connectType = productMatch.Entry.ConnectType
				}
				attrs := lightingDesignSlotAttrs(slot, roomName, deviceName, productMatch, productCandidates)
				device := map[string]any{
					"localId":         deviceLocalID,
					"localName":       deviceName,
					"gatewayDeviceId": gatewayID,
					"roomId":          roomLocalID,
					"addr":            deviceLocalID,
					"pid":             pid,
					"connectType":     connectType,
					"attrs":           attrs,
				}
				pcID := int64FromMap(slot, "pcId", 0)
				if pcID <= 0 && lightingDesignHasProductIdentity(productMatch) && productMatch.Entry.PCID > 0 {
					pcID = productMatch.Entry.PCID
				}
				if pcID > 0 {
					device["pcId"] = pcID
				}
				if mac := strings.TrimSpace(stringFromMap(slot, "mac")); mac != "" {
					device["mac"] = mac
				}
				devices = append(devices, device)
				slotsByRoomName[roomName] = append(slotsByRoomName[roomName], lightingDesignSlotRef{
					DeviceLocalID: deviceLocalID,
					Name:          deviceName,
					BaseName:      baseName,
					Category:      stringFromMap(attrs, "category"),
					Series:        stringFromMap(attrs, "series"),
					ProductName:   stringFromMap(attrs, "productName"),
					MaterialCode:  stringFromMap(attrs, "materialCode"),
					GroupKey:      lightingDesignGroupKey(slot, baseName),
				})
			}
		}
	}
	explicitDeviceGroups, err := normalizeLightingDesignExplicitGroups(payload, slotsByRoomName, roomLocalIDByName, localSeq)
	if err != nil {
		return nil, err
	}
	if len(explicitDeviceGroups) > 0 {
		deviceGroups = append(explicitDeviceGroups, deviceGroups...)
		deviceGroups = dedupeLightingDesignDeviceGroups(deviceGroups)
	}
	if len(devices) > lightingDesignMaxDevices {
		return nil, fmt.Errorf("device slot count exceeds limit %d", lightingDesignMaxDevices)
	}
	if len(deviceGroups) > lightingDesignMaxGroups {
		return nil, fmt.Errorf("device group count exceeds limit %d", lightingDesignMaxGroups)
	}
	scenes, err := normalizeLightingDesignMetadataList(payload["scenes"], lightingDesignMaxScenes, "scene")
	if err != nil {
		return nil, err
	}
	automations, err := normalizeLightingDesignMetadataList(payload["automations"], lightingDesignMaxAutomations, "automation")
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"houseId":  requestNumberOrStringForAPI(houseID),
		"clearAll": clearAll,
		"gateways": []any{map[string]any{
			"localId":   gatewayID,
			"localName": firstNonEmpty(stringFromMap(payload, "gatewayName"), "AI照明设计网关槽位"),
			"pid":       int64FromMap(payload, "gatewayPid", 17000001),
			"pcId":      int64FromMap(payload, "gatewayPcId", 2),
		}},
		"rooms":   normalizedRooms,
		"devices": devices,
		"async":   boolFromMap(payload, "async"),
	}
	if len(deviceGroups) > 0 {
		result["deviceGroups"] = deviceGroups
	}
	if len(scenes) > 0 {
		result["scenes"] = scenes
	}
	if len(automations) > 0 {
		result["automations"] = automations
	}
	return result, nil
}

type lightingDesignSlotRef struct {
	DeviceLocalID int64
	Name          string
	BaseName      string
	Category      string
	Series        string
	ProductName   string
	MaterialCode  string
	GroupKey      string
}

func normalizeLightingDesignExplicitGroups(payload map[string]any, slotsByRoomName map[string][]lightingDesignSlotRef, roomLocalIDByName map[string]int64, localSeq *lightingDesignLocalSeq) ([]any, error) {
	groups, ok := mapListFromAny(payload["groups"])
	if !ok || len(groups) == 0 {
		return nil, nil
	}
	if len(groups) > lightingDesignMaxGroups {
		return nil, fmt.Errorf("group count exceeds limit %d", lightingDesignMaxGroups)
	}
	result := make([]any, 0, len(groups))
	for _, group := range groups {
		groupName := strings.TrimSpace(firstNonEmpty(stringFromMap(group, "name"), stringFromMap(group, "localName")))
		if groupName == "" {
			return nil, fmt.Errorf("group name is required")
		}
		roomName := strings.TrimSpace(stringFromMap(group, "roomName"))
		if roomName == "" {
			return nil, fmt.Errorf("group roomName is required")
		}
		roomLocalID := roomLocalIDByName[roomName]
		if roomLocalID <= 0 {
			return nil, fmt.Errorf("group roomName %q does not match an imported room", roomName)
		}
		deviceIDs := lightingDesignExplicitGroupDeviceIDs(group, slotsByRoomName[roomName])
		if len(deviceIDs) == 0 {
			return nil, fmt.Errorf("group %q did not match any imported device slots", groupName)
		}
		result = append(result, map[string]any{
			"localId":   int64FromMap(group, "localId", localSeq.next()),
			"localName": groupName,
			"roomId":    roomLocalID,
			"deviceIds": int64sToAny(deviceIDs),
		})
	}
	return result, nil
}

func lightingDesignExplicitGroupDeviceIDs(group map[string]any, slots []lightingDesignSlotRef) []int64 {
	if ids, ok := anyInt64List(group["deviceIds"]); ok && len(ids) > 0 {
		return ids
	}
	match, _ := group["match"].(map[string]any)
	if len(match) == 0 {
		if groupKey := strings.TrimSpace(stringFromMap(group, "groupKey")); groupKey != "" {
			match = map[string]any{"groupKey": groupKey}
		}
	}
	if len(match) == 0 {
		return nil
	}
	result := []int64{}
	for _, slot := range slots {
		if lightingDesignSlotMatchesGroup(slot, match) {
			result = append(result, slot.DeviceLocalID)
		}
	}
	return result
}

func lightingDesignSlotMatchesGroup(slot lightingDesignSlotRef, match map[string]any) bool {
	for key, raw := range match {
		expected := lightingDesignStringFromAny(raw)
		if expected == "" {
			continue
		}
		switch key {
		case "name", "slotName", "deviceName":
			if !strings.Contains(slot.Name, expected) && !strings.Contains(slot.BaseName, expected) {
				return false
			}
		case "category":
			if slot.Category != expected {
				return false
			}
		case "series":
			if slot.Series != expected {
				return false
			}
		case "productName":
			if slot.ProductName != expected {
				return false
			}
		case "materialCode":
			if slot.MaterialCode != expected {
				return false
			}
		case "groupKey":
			if slot.GroupKey != expected {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func anyInt64List(value any) ([]int64, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]int64, 0, len(items))
	for _, item := range items {
		parsed, ok := lightingDesignIntFromAny(item)
		if !ok || parsed <= 0 {
			return nil, false
		}
		result = append(result, int64(parsed))
	}
	return result, true
}

func dedupeLightingDesignDeviceGroups(groups []any) []any {
	seen := map[string]bool{}
	result := make([]any, 0, len(groups))
	for _, raw := range groups {
		group, ok := raw.(map[string]any)
		if !ok {
			result = append(result, raw)
			continue
		}
		key := lightingDesignDeviceGroupMemberKey(group)
		if key == "" {
			key = strings.TrimSpace(stringFromMap(group, "localName")) + "\x00" + lightingDesignStringFromAny(group["roomId"])
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, group)
	}
	return result
}

func lightingDesignDeviceGroupMemberKey(group map[string]any) string {
	roomID := lightingDesignStringFromAny(group["roomId"])
	deviceIDs, ok := group["deviceIds"].([]any)
	if roomID == "" || !ok || len(deviceIDs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(deviceIDs))
	for _, id := range deviceIDs {
		parsed := lightingDesignStringFromAny(id)
		if parsed == "" {
			return ""
		}
		parts = append(parts, parsed)
	}
	return roomID + "\x00" + strings.Join(parts, ",")
}

func normalizedLightingDesignImportPayload(houseID string, payload map[string]any) (map[string]any, bool, error) {
	rooms, hasRooms := mapListFromAny(payload["rooms"])
	devices, hasDevices := mapListFromAny(payload["devices"])
	gateways, hasGateways := mapListFromAny(payload["gateways"])
	if !hasRooms || (!hasDevices && !hasGateways && !lightingDesignRoomsLookNormalized(rooms)) {
		return nil, false, nil
	}
	if len(rooms) == 0 {
		return nil, true, fmt.Errorf("at least one room is required")
	}
	if len(rooms) > lightingDesignMaxRooms {
		return nil, true, fmt.Errorf("room count exceeds limit %d", lightingDesignMaxRooms)
	}
	if len(devices) > lightingDesignMaxDevices {
		return nil, true, fmt.Errorf("device slot count exceeds limit %d", lightingDesignMaxDevices)
	}
	groups, _ := mapListFromAny(payload["deviceGroups"])
	if len(groups) > lightingDesignMaxGroups {
		return nil, true, fmt.Errorf("device group count exceeds limit %d", lightingDesignMaxGroups)
	}
	scenes, err := normalizeLightingDesignMetadataList(payload["scenes"], lightingDesignMaxScenes, "scene")
	if err != nil {
		return nil, true, err
	}
	automations, err := normalizeLightingDesignMetadataList(payload["automations"], lightingDesignMaxAutomations, "automation")
	if err != nil {
		return nil, true, err
	}
	gatewayID := int64FromMap(payload, "gatewayLocalId", lightingDesignDefaultGatewayLocalID)
	if gatewayID <= 0 {
		gatewayID = lightingDesignDefaultGatewayLocalID
	}
	if !hasGateways {
		gateways = []map[string]any{{
			"localId":   gatewayID,
			"localName": firstNonEmpty(stringFromMap(payload, "gatewayName"), "AI照明设计网关槽位"),
			"pid":       int64FromMap(payload, "gatewayPid", 17000001),
			"pcId":      int64FromMap(payload, "gatewayPcId", 2),
		}}
	}
	result := map[string]any{
		"houseId":  requestNumberOrStringForAPI(houseID),
		"clearAll": boolFromMap(payload, "clearAll") || boolFromMap(payload, "overwrite"),
		"gateways": mapsToAny(gateways),
		"rooms":    mapsToAny(rooms),
		"devices":  mapsToAny(devices),
		"async":    boolFromMap(payload, "async"),
	}
	if len(groups) > 0 {
		result["deviceGroups"] = mapsToAny(groups)
	}
	if len(scenes) > 0 {
		result["scenes"] = scenes
	}
	if len(automations) > 0 {
		result["automations"] = automations
	}
	return result, true, nil
}

func lightingDesignRoomsLookNormalized(rooms []map[string]any) bool {
	for _, room := range rooms {
		if strings.TrimSpace(stringFromMap(room, "localName")) != "" || int64FromMap(room, "localId", 0) > 0 {
			return true
		}
	}
	return false
}

func (client LightingDesignImportClient) verify(ctx context.Context, houseID string, credentials requestCredentials, counts map[string]int, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	totalCalls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		entities, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
			HouseID: houseID,
			Credentials: EntityListCredentials{
				Authorization: credentials.Authorization,
				ClientID:      credentials.ClientID,
			},
		})
		totalCalls += HouseScopedEntityListCallCount()
		if err != nil || lightingDesignVerificationPasses(entities, counts) || attempt == attempts-1 {
			return err == nil && lightingDesignVerificationPasses(entities, counts), totalCalls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, totalCalls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, totalCalls, nil
}

func lightingDesignVerificationPasses(entities EntityListResult, counts map[string]int) bool {
	if counts["rooms"] > 0 && entities.Counts["room"] <= 0 {
		return false
	}
	if counts["devices"] > 0 && entities.Counts["device"] <= 0 {
		return false
	}
	return true
}

func normalizeLightingDesignRooms(payload map[string]any) ([]map[string]any, error) {
	for _, key := range []string{"rooms", "roomSlots", "spaces"} {
		if rooms, ok := mapListFromAny(payload[key]); ok {
			return rooms, nil
		}
	}
	if roomName := strings.TrimSpace(firstNonEmpty(stringFromMap(payload, "roomName"), stringFromMap(payload, "name"))); roomName != "" {
		room := map[string]any{"name": roomName}
		if items, ok := mapListFromAny(payload["items"]); ok {
			room["items"] = mapsToAny(items)
		}
		if slots, ok := mapListFromAny(payload["slots"]); ok {
			room["slots"] = mapsToAny(slots)
		}
		if devices, ok := mapListFromAny(payload["devices"]); ok {
			room["devices"] = mapsToAny(devices)
		}
		return []map[string]any{room}, nil
	}
	return nil, nil
}

func normalizeLightingDesignRoomSlots(room map[string]any) ([]map[string]any, error) {
	for _, key := range []string{"items", "slots", "devices"} {
		if items, ok := mapListFromAny(room[key]); ok {
			return items, nil
		}
	}
	return nil, nil
}

func normalizeLightingDesignMetadataList(value any, limit int, label string) ([]any, error) {
	items, ok := mapListFromAny(value)
	if !ok {
		return nil, nil
	}
	if len(items) > limit {
		return nil, fmt.Errorf("%s count exceeds limit %d", label, limit)
	}
	result := make([]any, 0, len(items))
	for index, item := range items {
		if strings.TrimSpace(stringFromMap(item, "name")) == "" {
			return nil, fmt.Errorf("%s name is required", label)
		}
		if int64FromMap(item, "localId", 0) <= 0 {
			item["localId"] = int64(800000 + index + 1)
		}
		result = append(result, item)
	}
	return result, nil
}

func lightingDesignSlotName(slot map[string]any, baseName string, index int, count int) string {
	if pattern := strings.TrimSpace(stringFromMap(slot, "namePattern")); pattern != "" {
		return strings.ReplaceAll(pattern, "{n}", strconv.Itoa(index))
	}
	if count <= 1 {
		return baseName
	}
	return fmt.Sprintf("%s%d", baseName, index)
}

func lightingDesignSlotAttrs(slot map[string]any, roomName string, deviceName string, productMatch lightingDesignProductMatch, productCandidates []lightingDesignProductMatch) map[string]any {
	attrs := map[string]any{
		"designSlot": true,
		"roomName":   roomName,
		"deviceName": deviceName,
	}
	for _, key := range []string{"category", "color", "installStyle", "beamAngle", "series", "materialCode", "productName", "productModel", "productSku", "productSpu", "modelNo", "notes"} {
		if value, ok := slot[key]; ok {
			attrs[key] = value
		}
	}
	for key, value := range lightingDesignProductAttrs(productMatch, productCandidates) {
		attrs[key] = value
	}
	return attrs
}

func lightingDesignGroupKey(slot map[string]any, fallback string) string {
	return firstNonEmpty(
		stringFromMap(slot, "groupKey"),
		stringFromMap(slot, "type"),
		stringFromMap(slot, "category"),
		stringFromMap(slot, "productName"),
		fallback,
	)
}

func lightingDesignImportMode(payload map[string]any) string {
	if boolFromMap(payload, "clearAll") {
		return "sync_metadata_overwrite"
	}
	return "sync_metadata_incremental"
}

func lightingDesignImportCounts(payload map[string]any) map[string]int {
	return map[string]int{
		"gateways":    len(anyListFromMap(payload, "gateways")),
		"rooms":       len(anyListFromMap(payload, "rooms")),
		"devices":     len(anyListFromMap(payload, "devices")),
		"groups":      len(anyListFromMap(payload, "deviceGroups")),
		"scenes":      len(anyListFromMap(payload, "scenes")),
		"automations": len(anyListFromMap(payload, "automations")),
	}
}

func lightingDesignImportMappings(data any) map[string]any {
	item, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	result := map[string]any{}
	for output, key := range map[string]string{
		"gatewayLocalIdToCloudIds":    "gatewayLocalIdToCloudSlotIds",
		"deviceLocalIdToCloudIds":     "deviceLocalIdToCloudSlotIds",
		"roomLocalIdToCloudIds":       "roomLocalIdToCloudSlotIds",
		"groupLocalIdToCloudIds":      "deviceGroupLocalIdToCloudSlotIds",
		"areaLocalIdToCloudIds":       "areaLocalIdToCloudSlotIds",
		"sceneLocalIdToCloudIds":      "sceneLocalIdToCloudSlotIds",
		"automationLocalIdToCloudIds": "automationLocalIdToCloudSlotIds",
		"progressKey":                 "progressKey",
	} {
		if value, ok := item[key]; ok {
			result[output] = value
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

type lightingDesignLocalSeq struct {
	nextValue int64
}

func newLightingDesignLocalSeq() *lightingDesignLocalSeq {
	return &lightingDesignLocalSeq{nextValue: 1000}
}

func (seq *lightingDesignLocalSeq) next() int64 {
	seq.nextValue++
	return seq.nextValue
}

func mapListFromAny(value any) ([]map[string]any, bool) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		result = append(result, typed)
	}
	return result, true
}

func mapsToAny(items []map[string]any) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func int64sToAny(values []int64) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func anyListFromMap(values map[string]any, key string) []any {
	items, ok := values[key].([]any)
	if !ok {
		return nil
	}
	return items
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return lightingDesignStringFromAny(values[key])
}

func intFromMap(values map[string]any, key string, fallback int) int {
	if parsed, ok := lightingDesignIntFromAny(values[key]); ok {
		return parsed
	}
	return fallback
}

func int64FromMap(values map[string]any, key string, fallback int64) int64 {
	if parsed, ok := lightingDesignIntFromAny(values[key]); ok {
		return int64(parsed)
	}
	return fallback
}

func boolFromMap(values map[string]any, key string) bool {
	parsed, ok := lightingDesignBoolFromAny(values[key])
	return ok && parsed
}

func hasMapKey(values map[string]any, key string) bool {
	_, ok := values[key]
	return ok
}

func lightingDesignBoolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		}
	}
	return false, false
}

func lightingDesignIntFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func lightingDesignStringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	}
	return ""
}
