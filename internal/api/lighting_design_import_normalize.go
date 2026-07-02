package api

import (
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func NormalizeLightingDesignImportPayload(houseID string, payload map[string]any) (map[string]any, error) {
	if strings.TrimSpace(houseID) != "" {
		if _, err := parseID(houseID, "house id"); err != nil {
			return nil, err
		}
	}
	if lightingDesignImportHasUnsupportedOverwrite(payload) {
		return nil, fmt.Errorf("clearAll/overwrite is not supported by lighting.design.import; use dedicated delete/update intents for existing configured homes")
	}
	semanticResult, err := semantic.NormalizeLightingDesignImport(payload, semantic.LightingDesignOptions{HouseID: houseID})
	if err != nil {
		return nil, err
	}
	payload = semanticResult.Payload
	return normalizeLightingDesignHouseMeta(houseID, payload)
}

func lightingDesignImportHasUnsupportedOverwrite(payload map[string]any) bool {
	for _, key := range semantic.ImportOverwriteFields() {
		if boolFromMap(payload, key) {
			return true
		}
	}
	return false
}

func normalizeLightingDesignHouseMeta(houseID string, payload map[string]any) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	expanded := copyLightingDesignDeepMap(payload)
	if lightingDesignImportHasUnsupportedOverwrite(expanded) {
		return nil, fmt.Errorf("clearAll/overwrite is not supported by lighting.design.import; use dedicated delete/update intents for existing configured homes")
	}
	gateway, ok := mapFromAny(expanded[semantic.FieldGateway])
	if !ok {
		if looksLikeNaturalLightingDesign(expanded) {
			return nil, fmt.Errorf("lighting.design.import requires the CLI lighting design model: rooms[].deviceSlots[], rooms[].groups[], scenes[].actions[], and automations[].actions[]")
		}
		return nil, fmt.Errorf("lighting design model is required")
	}
	meta := copyLightingDesignDeepMap(expanded)
	if strings.TrimSpace(houseID) != "" {
		meta[semantic.FieldHouseID] = requestNumberOrStringForAPI(houseID)
	} else {
		delete(meta, semantic.FieldHouseID)
	}
	if name := strings.TrimSpace(stringFromMap(meta, semantic.FieldName)); name == "" {
		meta[semantic.FieldName] = firstNonEmpty(stringFromMap(meta, semantic.FieldHouseName), "AI照明设计")
	}
	if strings.TrimSpace(stringFromMap(meta, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
		meta[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = "hm1"
	}
	if version, ok := lightingDesignIntFromAny(meta[semantic.FieldVersion]); !ok || version <= 0 {
		meta[semantic.FieldVersion] = 2
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
	meta[semantic.FieldGateway] = normalizedGateway
	areaList, err := normalizeLightingDesignHouseMetaAreas(meta[semantic.InternalField(semantic.DomainImport, semantic.FieldAreas)], index)
	if err != nil {
		return nil, err
	}
	if len(areaList) > 0 {
		meta[semantic.InternalField(semantic.DomainImport, semantic.FieldAreas)] = areaList
	} else {
		delete(meta, semantic.InternalField(semantic.DomainImport, semantic.FieldAreas))
	}
	sceneList, err := normalizeLightingDesignHouseMetaScenes(meta[semantic.InternalField(semantic.DomainImport, semantic.FieldScenes)], &index)
	if err != nil {
		return nil, err
	}
	if len(sceneList) > 0 {
		meta[semantic.InternalField(semantic.DomainImport, semantic.FieldScenes)] = sceneList
	} else {
		delete(meta, semantic.InternalField(semantic.DomainImport, semantic.FieldScenes))
	}
	automationList, err := normalizeLightingDesignHouseMetaAutomations(meta[semantic.InternalField(semantic.DomainImport, semantic.FieldAutomations)], index)
	if err != nil {
		return nil, err
	}
	if len(automationList) > 0 {
		meta[semantic.InternalField(semantic.DomainImport, semantic.FieldAutomations)] = automationList
	} else {
		delete(meta, semantic.InternalField(semantic.DomainImport, semantic.FieldAutomations))
	}
	for _, key := range semantic.ImportCleanupFields() {
		delete(meta, key)
	}
	return meta, nil
}

func normalizeLightingDesignHouseMetaGateway(gateway map[string]any, index *lightingDesignMetaIndex) (map[string]any, error) {
	result := copyLightingDesignDeepMap(gateway)
	if strings.TrimSpace(stringFromMap(result, semantic.FieldName)) == "" {
		result[semantic.FieldName] = "默认网关"
	}
	if strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" && result[semantic.FieldGatewayDeviceID] == nil {
		result[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = "1"
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)]); !ok {
		result[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)] = int64(17000001)
	}
	rooms, ok := mapListFromAny(result[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)])
	if !ok || len(rooms) == 0 {
		return nil, fmt.Errorf("rooms[] is required")
	}
	if len(rooms) > lightingDesignMaxRooms {
		return nil, fmt.Errorf("rooms[] count exceeds limit %d", lightingDesignMaxRooms)
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
			return nil, fmt.Errorf("deviceSlots[] count exceeds limit %d", lightingDesignMaxDevices)
		}
		if groupCount > lightingDesignMaxGroups {
			return nil, fmt.Errorf("groups[] count exceeds limit %d", lightingDesignMaxGroups)
		}
		normalizedRooms = append(normalizedRooms, normalized)
	}
	result[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)] = normalizedRooms
	return result, nil
}

func normalizeLightingDesignHouseMetaRoom(room map[string]any, roomIndex int, index *lightingDesignMetaIndex) (map[string]any, int, int, error) {
	result := copyLightingDesignDeepMap(room)
	name := strings.TrimSpace(stringFromMap(result, semantic.FieldName))
	if name == "" {
		return nil, 0, 0, fmt.Errorf("rooms[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
		result[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("rm%d", roomIndex+1)
	}
	roomTempID := stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))
	if strings.TrimSpace(stringFromMap(result, semantic.FieldIcon)) == "" {
		result[semantic.FieldIcon] = "room_1"
	}
	index.RoomsByTempID[roomTempID] = name
	devices, _ := mapListFromAny(result[semantic.InternalField(semantic.DomainImport, semantic.FieldDeviceSlots)])
	normalizedDevices := make([]any, 0, len(devices))
	for deviceIndex, device := range devices {
		normalized, err := normalizeLightingDesignHouseMetaDevice(device, deviceIndex)
		if err != nil {
			return nil, 0, 0, err
		}
		index.DevicesByTempID[stringFromMap(normalized, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))] = stringFromMap(normalized, semantic.FieldName)
		normalizedDevices = append(normalizedDevices, normalized)
	}
	result[semantic.InternalField(semantic.DomainImport, semantic.FieldDeviceSlots)] = normalizedDevices
	groups, _ := mapListFromAny(result[semantic.InternalField(semantic.DomainImport, semantic.FieldGroups)])
	normalizedGroups := make([]any, 0, len(groups))
	for groupIndex, group := range groups {
		normalized, err := normalizeLightingDesignHouseMetaGroup(group, groupIndex, index)
		if err != nil {
			return nil, 0, 0, err
		}
		index.GroupsByTempID[stringFromMap(normalized, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))] = stringFromMap(normalized, semantic.FieldName)
		normalizedGroups = append(normalizedGroups, normalized)
	}
	result[semantic.InternalField(semantic.DomainImport, semantic.FieldGroups)] = normalizedGroups
	return result, len(normalizedDevices), len(normalizedGroups), nil
}

func normalizeLightingDesignHouseMetaDevice(device map[string]any, index int) (map[string]any, error) {
	result := copyLightingDesignDeepMap(device)
	if strings.TrimSpace(stringFromMap(result, semantic.FieldName)) == "" {
		return nil, fmt.Errorf("rooms[].deviceSlots[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
		result[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("dv%d", index+1)
	}
	match := lightingDesignExplicitProduct(result)
	if strings.TrimSpace(firstAnyString(result, semantic.ProductCodeFields()...)) == "" {
		return nil, fmt.Errorf("rooms[].deviceSlots[].product.skuCode is required; Skill must choose a concrete SKU before import")
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)]); !ok && match.Entry.PID > 0 {
		result[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)] = match.Entry.PID
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)]); !ok {
		return nil, fmt.Errorf("rooms[].deviceSlots[].product.capabilityPid is required; Skill must choose a product with a known capability PID before import")
	}
	lightingDesignMergeExtraMeta(result, result)
	delete(result, semantic.ImportRoomTempIDField())
	delete(result, semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCategoryID))
	lightingDesignKeepDeviceHouseMetaFields(result)
	return result, nil
}

func normalizeLightingDesignHouseMetaGroup(group map[string]any, index int, metaIndex *lightingDesignMetaIndex) (map[string]any, error) {
	result := copyLightingDesignDeepMap(group)
	if strings.TrimSpace(stringFromMap(result, semantic.FieldName)) == "" {
		return nil, fmt.Errorf("rooms[].groups[].name is required")
	}
	if strings.TrimSpace(stringFromMap(result, semantic.InternalField(semantic.DomainImport, semantic.FieldKey))) == "" {
		result[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)] = fmt.Sprintf("gp%d", index+1)
	}
	if _, ok := lightingDesignIntFromAny(result[semantic.InternalGroupCapabilityIDField()]); !ok {
		if componentID, ok := lightingDesignIntFromAny(result[semantic.InternalCloudComponentIDField()]); ok {
			result[semantic.InternalGroupCapabilityIDField()] = componentID
		} else {
			return nil, fmt.Errorf("rooms[].groups[].groupCapability is required")
		}
	}
	ids, ok := lightingDesignStringListFromAny(result[semantic.InternalField(semantic.DomainImport, semantic.FieldSlotKeys)])
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("rooms[].groups[].slotKeys is required")
	}
	for _, tempID := range ids {
		if metaIndex.DevicesByTempID[tempID] == "" {
			return nil, fmt.Errorf("rooms[].groups[].slotKeys references unknown device slot key %q", tempID)
		}
	}
	result[semantic.InternalField(semantic.DomainImport, semantic.FieldSlotKeys)] = ids
	delete(result, semantic.InternalCloudComponentIDField())
	delete(result, semantic.FieldGroupCategory)
	delete(result, semantic.FieldGroupCapability)
	delete(result, semantic.FieldSlotKeys)
	return result, nil
}

func lightingDesignKeepDeviceHouseMetaFields(device map[string]any) {
	if extra, ok := device[semantic.FieldExtraMeta].(map[string]string); ok {
		if materialCode := strings.TrimSpace(extra[semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCode)]); materialCode != "" {
			device[semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCode)] = materialCode
		}
	}
}
