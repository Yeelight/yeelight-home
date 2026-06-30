package api

import (
	"fmt"
	"strings"
)

func NormalizeLightingDesignImportPayload(houseID string, payload map[string]any) (map[string]any, error) {
	if _, err := parseID(houseID, "house id"); err != nil {
		return nil, err
	}
	return normalizeLightingDesignHouseMeta(houseID, payload)
}

func normalizeLightingDesignHouseMeta(houseID string, payload map[string]any) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	expanded := lightingDesignExpandShortKeys(payload)
	if boolFromMap(expanded, "clearAll") || boolFromMap(expanded, "overwrite") {
		return nil, fmt.Errorf("clearAll/overwrite is not supported by lighting.design.import HouseMeta; use dedicated semantic delete/update intents for existing configured homes")
	}
	gateway, ok := mapFromAny(expanded["gateway"])
	if !ok {
		if looksLikeNaturalLightingDesign(expanded) {
			return nil, fmt.Errorf("lighting.design.import now requires HouseMeta payload for /v1/meta/import; Skill must generate gateway.roomList/deviceList/groupList/sceneList/automationList instead of natural rooms/items")
		}
		return nil, fmt.Errorf("gateway is required")
	}
	meta := copyLightingDesignDeepMap(expanded)
	meta["houseId"] = requestNumberOrStringForAPI(houseID)
	if name := strings.TrimSpace(stringFromMap(meta, "name")); name == "" {
		meta["name"] = firstNonEmpty(stringFromMap(meta, "houseName"), "AI照明设计")
	}
	if strings.TrimSpace(stringFromMap(meta, "tempId")) == "" {
		meta["tempId"] = "hm1"
	}
	if version, ok := lightingDesignIntFromAny(meta["version"]); !ok || version <= 0 {
		meta["version"] = 2
	}
	index := lightingDesignMetaIndex{
		RoomsByTempID:   map[string]string{},
		DevicesByTempID: map[string]string{},
		GroupsByTempID:  map[string]string{},
		ScenesByTempID:  map[string]string{},
	}
	normalizedGateway, err := normalizeLightingDesignHouseMetaGateway(gateway, &index)
	if err != nil {
		return nil, err
	}
	meta["gateway"] = normalizedGateway
	areaList, err := normalizeLightingDesignHouseMetaAreas(meta["areaList"], index)
	if err != nil {
		return nil, err
	}
	if len(areaList) > 0 {
		meta["areaList"] = areaList
	} else {
		delete(meta, "areaList")
	}
	sceneList, err := normalizeLightingDesignHouseMetaScenes(meta["sceneList"], &index)
	if err != nil {
		return nil, err
	}
	if len(sceneList) > 0 {
		meta["sceneList"] = sceneList
	} else {
		delete(meta, "sceneList")
	}
	automationList, err := normalizeLightingDesignHouseMetaAutomations(meta["automationList"], index)
	if err != nil {
		return nil, err
	}
	if len(automationList) > 0 {
		meta["automationList"] = automationList
	} else {
		delete(meta, "automationList")
	}
	for _, key := range []string{"rooms", "roomList", "devices", "deviceList", "groups", "groupList", "scenes", "automations"} {
		delete(meta, key)
	}
	return meta, nil
}

func normalizeLightingDesignHouseMetaGateway(gateway map[string]any, index *lightingDesignMetaIndex) (map[string]any, error) {
	result := copyLightingDesignDeepMap(gateway)
	if strings.TrimSpace(stringFromMap(result, "name")) == "" {
		result["name"] = "默认网关"
	}
	if strings.TrimSpace(stringFromMap(result, "tempId")) == "" && result["gatewayDeviceId"] == nil {
		result["tempId"] = "gw1"
	}
	if _, ok := lightingDesignIntFromAny(result["pid"]); !ok {
		result["pid"] = int64(17000001)
	}
	rooms, ok := mapListFromAny(result["roomList"])
	if !ok || len(rooms) == 0 {
		return nil, fmt.Errorf("gateway.roomList is required")
	}
	if len(rooms) > lightingDesignMaxRooms {
		return nil, fmt.Errorf("gateway.roomList count exceeds limit %d", lightingDesignMaxRooms)
	}
	normalizedRooms := make([]any, 0, len(rooms))
	deviceCount := 0
	groupCount := 0
	for roomIndex, room := range rooms {
		normalized, devices, groups, err := normalizeLightingDesignHouseMetaRoom(room, roomIndex, index)
		if err != nil {
			return nil, err
		}
		deviceCount += devices
		groupCount += groups
		if deviceCount > lightingDesignMaxDevices {
			return nil, fmt.Errorf("deviceList count exceeds limit %d", lightingDesignMaxDevices)
		}
		if groupCount > lightingDesignMaxGroups {
			return nil, fmt.Errorf("groupList count exceeds limit %d", lightingDesignMaxGroups)
		}
		normalizedRooms = append(normalizedRooms, normalized)
	}
	result["roomList"] = normalizedRooms
	return result, nil
}

func normalizeLightingDesignHouseMetaRoom(room map[string]any, roomIndex int, index *lightingDesignMetaIndex) (map[string]any, int, int, error) {
	result := copyLightingDesignDeepMap(room)
	name := strings.TrimSpace(stringFromMap(result, "name"))
	if name == "" {
		return nil, 0, 0, fmt.Errorf("gateway.roomList[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, "tempId")) == "" {
		result["tempId"] = fmt.Sprintf("rm%d", roomIndex+1)
	}
	roomTempID := stringFromMap(result, "tempId")
	if strings.TrimSpace(stringFromMap(result, "icon")) == "" {
		result["icon"] = "room_1"
	}
	index.RoomsByTempID[roomTempID] = name
	devices, _ := mapListFromAny(result["deviceList"])
	normalizedDevices := make([]any, 0, len(devices))
	for deviceIndex, device := range devices {
		normalized, err := normalizeLightingDesignHouseMetaDevice(device, roomTempID, deviceIndex)
		if err != nil {
			return nil, 0, 0, err
		}
		index.DevicesByTempID[stringFromMap(normalized, "tempId")] = stringFromMap(normalized, "name")
		normalizedDevices = append(normalizedDevices, normalized)
	}
	result["deviceList"] = normalizedDevices
	groups, _ := mapListFromAny(result["groupList"])
	normalizedGroups := make([]any, 0, len(groups))
	for groupIndex, group := range groups {
		normalized, err := normalizeLightingDesignHouseMetaGroup(group, groupIndex, index)
		if err != nil {
			return nil, 0, 0, err
		}
		index.GroupsByTempID[stringFromMap(normalized, "tempId")] = stringFromMap(normalized, "name")
		normalizedGroups = append(normalizedGroups, normalized)
	}
	result["groupList"] = normalizedGroups
	return result, len(normalizedDevices), len(normalizedGroups), nil
}

func normalizeLightingDesignHouseMetaDevice(device map[string]any, roomTempID string, index int) (map[string]any, error) {
	result := copyLightingDesignDeepMap(device)
	if strings.TrimSpace(stringFromMap(result, "name")) == "" {
		return nil, fmt.Errorf("deviceList[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, "tempId")) == "" {
		result["tempId"] = fmt.Sprintf("dv%d", index+1)
	}
	if strings.TrimSpace(stringFromMap(result, "roomTempId")) == "" {
		result["roomTempId"] = roomTempID
	}
	match := lightingDesignExplicitProduct(result)
	if _, ok := lightingDesignIntFromAny(result["pid"]); !ok && match.Entry.PID > 0 {
		result["pid"] = match.Entry.PID
	}
	if _, ok := lightingDesignIntFromAny(result["pid"]); !ok {
		return nil, fmt.Errorf("deviceList[].pid is required; Skill must select a product before import")
	}
	lightingDesignMergeExtraMeta(result, result)
	return result, nil
}

func normalizeLightingDesignHouseMetaGroup(group map[string]any, index int, metaIndex *lightingDesignMetaIndex) (map[string]any, error) {
	result := copyLightingDesignDeepMap(group)
	if strings.TrimSpace(stringFromMap(result, "name")) == "" {
		return nil, fmt.Errorf("groupList[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, "tempId")) == "" {
		result["tempId"] = fmt.Sprintf("gp%d", index+1)
	}
	if _, ok := lightingDesignIntFromAny(result["componentId"]); !ok {
		if cid, ok := lightingDesignIntFromAny(result["cid"]); ok {
			result["componentId"] = cid
		} else {
			return nil, fmt.Errorf("groupList[].componentId is required")
		}
	}
	ids, ok := lightingDesignStringListFromAny(result["deviceTempIdList"])
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("groupList[].deviceTempIdList is required")
	}
	for _, tempID := range ids {
		if metaIndex.DevicesByTempID[tempID] == "" {
			return nil, fmt.Errorf("groupList[].deviceTempIdList references unknown device tempId %q", tempID)
		}
	}
	result["deviceTempIdList"] = ids
	delete(result, "cid")
	return result, nil
}
