package semantic

import (
	"fmt"
	"strings"
)

type LightingDesignOptions struct {
	HouseID string
}

type LightingDesignResult struct {
	Payload  map[string]any
	Semantic bool
}

func NormalizeLightingDesignImport(source map[string]any, options LightingDesignOptions) (LightingDesignResult, error) {
	if source == nil {
		source = map[string]any{}
	}
	if !looksLikeSemanticLightingDesign(source) {
		return LightingDesignResult{}, fmt.Errorf("lighting.design.import requires the CLI lighting design model: rooms[].deviceSlots[], rooms[].groups[], scenes[].actions[], and automations[].actions[]")
	}
	builder := lightingDesignBuilder{
		roomRefs:  map[string]string{},
		slotRefs:  map[string]string{},
		groupRefs: map[string]string{},
		sceneRefs: map[string]string{},
	}
	payload := map[string]any{}
	if strings.TrimSpace(options.HouseID) != "" {
		payload["houseId"] = options.HouseID
	}
	if name := FirstString(source, "name", "homeName", "designName"); name != "" {
		payload[FieldName] = name
	}
	if tempID := FirstString(source, FieldKey); tempID != "" {
		payload[internalTempID] = tempID
	}
	gateway := builder.gateway(source)
	rooms, err := builder.rooms(source)
	if err != nil {
		return LightingDesignResult{}, err
	}
	gateway[internalRoomList] = rooms
	payload["gateway"] = gateway
	if areas, err := builder.areas(source); err != nil {
		return LightingDesignResult{}, err
	} else if len(areas) > 0 {
		payload[internalAreaList] = areas
	}
	if scenes, err := builder.scenes(source); err != nil {
		return LightingDesignResult{}, err
	} else if len(scenes) > 0 {
		payload[internalSceneList] = scenes
	}
	if automations, err := builder.automations(source); err != nil {
		return LightingDesignResult{}, err
	} else if len(automations) > 0 {
		payload[internalAutomationList] = automations
	}
	return LightingDesignResult{Payload: payload, Semantic: true}, nil
}

func looksLikeSemanticLightingDesign(source map[string]any) bool {
	if _, ok := source[FieldRooms]; ok {
		return true
	}
	if _, ok := source[FieldDeviceSlots]; ok {
		return true
	}
	return false
}

type lightingDesignBuilder struct {
	roomRefs  map[string]string
	slotRefs  map[string]string
	groupRefs map[string]string
	sceneRefs map[string]string
}

func (builder *lightingDesignBuilder) gateway(source map[string]any) map[string]any {
	gateway := map[string]any{
		internalTempID: "1",
		FieldName:      "默认网关",
	}
	if name := FirstString(source, FieldGatewayName); name != "" {
		gateway[FieldName] = name
	}
	if id := FirstString(source, FieldGatewayDeviceID); id != "" {
		gateway[FieldGatewayDeviceID] = NumberOrString(id)
	}
	return gateway
}

func (builder *lightingDesignBuilder) rooms(source map[string]any) ([]any, error) {
	rawRooms, ok := mapList(source["rooms"])
	if !ok {
		return nil, fmt.Errorf("rooms[] is required")
	}
	rooms := make([]any, 0, len(rawRooms))
	for index, raw := range rawRooms {
		room := map[string]any{}
		name := FirstString(raw, FieldName, "roomName")
		if name == "" {
			return nil, fmt.Errorf("rooms[].name is required")
		}
		if hasAnyKey(raw, "items", "slots", "devices", "slotGroups") {
			return nil, fmt.Errorf("lighting.design.import requires rooms[].deviceSlots[] and rooms[].groups[]")
		}
		key := firstKey(raw, fmt.Sprintf("room%d", index+1))
		room[internalTempID] = key
		room[FieldName] = name
		if icon := FirstString(raw, FieldIcon); icon != "" {
			room[FieldIcon] = icon
		}
		builder.remember(builder.roomRefs, key, name, raw)
		slots, err := builder.slotsForRoom(raw, key)
		if err != nil {
			return nil, err
		}
		room[internalDeviceList] = slots
		groups, err := builder.groupsForRoom(raw, key)
		if err != nil {
			return nil, err
		}
		room[internalGroupList] = groups
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (builder *lightingDesignBuilder) slotsForRoom(room map[string]any, roomKey string) ([]any, error) {
	rawSlots, ok := mapList(firstAny(room, FieldDeviceSlots))
	if !ok {
		return []any{}, nil
	}
	slots := make([]any, 0, len(rawSlots))
	for index, raw := range rawSlots {
		slot := map[string]any{}
		name := FirstString(raw, FieldName, "slotName", "deviceName")
		if name == "" {
			return nil, fmt.Errorf("rooms[].deviceSlots[].name is required")
		}
		key := firstKey(raw, fmt.Sprintf("%s-slot%d", roomKey, index+1))
		slot[internalTempID] = key
		slot[FieldName] = name
		slot[internalRoomTempID] = roomKey
		if product, ok := raw[FieldProduct].(map[string]any); ok {
			mergeMissing(slot, NormalizeProduct(product))
		}
		for _, key := range []string{FieldProductName, FieldProductSKU, FieldProductSPU, FieldCategory, FieldSeries, FieldNotes, FieldConnectType} {
			if value, ok := raw[key]; ok {
				slot[key] = deepValue(value)
			}
		}
		builder.remember(builder.slotRefs, key, name, raw)
		slots = append(slots, slot)
	}
	return slots, nil
}

func (builder *lightingDesignBuilder) groupsForRoom(room map[string]any, roomKey string) ([]any, error) {
	rawGroups, ok := mapList(firstAny(room, FieldGroups))
	if !ok {
		return []any{}, nil
	}
	groups := make([]any, 0, len(rawGroups))
	for index, raw := range rawGroups {
		group := map[string]any{}
		name := FirstString(raw, FieldName, "groupName")
		if name == "" {
			return nil, fmt.Errorf("rooms[].groups[].name is required")
		}
		key := firstKey(raw, fmt.Sprintf("%s-group%d", roomKey, index+1))
		group[internalTempID] = key
		group[FieldName] = name
		for _, key := range []string{FieldGroupCategory, FieldGroupCapability, FieldIcon, FieldNotes} {
			if value, ok := raw[key]; ok {
				group[key] = deepValue(value)
			}
		}
		if _, ok := Int(group[internalComponentID]); !ok {
			if componentID, ok := GroupCapabilityComponentID(group); ok {
				group[internalComponentID] = componentID
			} else {
				return nil, fmt.Errorf("rooms[].groups[].groupCapability is required")
			}
		}
		members := stringList(firstAny(raw, actionAliasConfig.SlotMembers...))
		if len(members) == 0 {
			return nil, fmt.Errorf("rooms[].groups[].slotKeys is required")
		}
		group[internalDeviceTempIDList] = members
		builder.remember(builder.groupRefs, key, name, raw)
		groups = append(groups, group)
	}
	return groups, nil
}

func GroupCapabilityComponentID(group map[string]any) (int, bool) {
	for _, key := range []string{FieldGroupCapability, FieldGroupCategory} {
		switch strings.ToLower(strings.TrimSpace(String(group[key]))) {
		case "light", "lighting", "lamp", "dimming", "colortemperature", "color_temperature", "color-temperature", "color", "灯", "灯光", "照明", "调光", "色温":
			return ResourceMeshGroup, true
		}
	}
	return 0, false
}

func (builder *lightingDesignBuilder) areas(source map[string]any) ([]any, error) {
	rawAreas, ok := mapList(source[FieldAreas])
	if !ok {
		return nil, nil
	}
	areas := make([]any, 0, len(rawAreas))
	for index, raw := range rawAreas {
		name := FirstString(raw, FieldName, "areaName")
		if name == "" {
			return nil, fmt.Errorf("areas[].name is required")
		}
		area := map[string]any{}
		area[internalTempID] = firstKey(raw, fmt.Sprintf("area%d", index+1))
		area[FieldName] = name
		if icon := FirstString(raw, FieldIcon); icon != "" {
			area[FieldIcon] = icon
		}
		area[internalRoomTempIDList] = stringList(firstAny(raw, FieldRoomKeys))
		areas = append(areas, area)
	}
	return areas, nil
}

func (builder *lightingDesignBuilder) scenes(source map[string]any) ([]any, error) {
	rawScenes, ok := mapList(source[FieldScenes])
	if !ok {
		return nil, nil
	}
	scenes := make([]any, 0, len(rawScenes))
	for index, raw := range rawScenes {
		scene, err := builder.scene(raw, index)
		if err != nil {
			return nil, err
		}
		scenes = append(scenes, scene)
	}
	return scenes, nil
}

func (builder *lightingDesignBuilder) scene(raw map[string]any, index int) (map[string]any, error) {
	name := FirstString(raw, FieldName, "sceneName")
	if name == "" {
		return nil, fmt.Errorf("scenes[].name is required")
	}
	scene := map[string]any{}
	key := firstKey(raw, fmt.Sprintf("scene%d", index+1))
	scene[internalTempID] = key
	scene[FieldName] = name
	if icon := FirstString(raw, FieldIcon); icon != "" {
		scene[FieldIcon] = icon
	}
	actions, ok := mapList(firstAny(raw, FieldActions))
	if !ok {
		return nil, fmt.Errorf("scenes[].actions is required")
	}
	details, err := builder.importActions(actions, true)
	if err != nil {
		return nil, err
	}
	scene[FieldDetails] = details
	delete(scene, FieldActions)
	builder.remember(builder.sceneRefs, key, name, raw)
	return scene, nil
}

func (builder *lightingDesignBuilder) automations(source map[string]any) ([]any, error) {
	rawAutomations, ok := mapList(source[FieldAutomations])
	if !ok {
		return nil, nil
	}
	automations := make([]any, 0, len(rawAutomations))
	for index, raw := range rawAutomations {
		automation, err := builder.automation(raw, index)
		if err != nil {
			return nil, err
		}
		automations = append(automations, automation)
	}
	return automations, nil
}

func (builder *lightingDesignBuilder) automation(raw map[string]any, index int) (map[string]any, error) {
	name := FirstString(raw, FieldName, "automationName")
	if name == "" {
		return nil, fmt.Errorf("automations[].name is required")
	}
	automation := map[string]any{}
	automation[internalTempID] = firstKey(raw, fmt.Sprintf("automation%d", index+1))
	automation[FieldName] = name
	if schedule, ok := AutomationScheduleFromRequest(raw); ok {
		automation[FieldStartTime] = schedule.StartTime
		automation[FieldEndTime] = schedule.EndTime
		automation[InternalRepeatTypeField()] = schedule.RepeatType
		if schedule.RepeatValue != "" {
			automation[InternalRepeatValueField()] = schedule.RepeatValue
		}
	}
	if params := NormalizeAutomationParamsFromRequest(raw); params != nil {
		automation[internalParams] = params
	}
	actions, ok := mapList(raw[FieldActions])
	if !ok {
		return nil, fmt.Errorf("automations[].actions is required")
	}
	normalized, err := builder.importActions(actions, false)
	if err != nil {
		return nil, err
	}
	automation[FieldActions] = normalized
	return automation, nil
}

func (builder *lightingDesignBuilder) importActions(actions []map[string]any, scene bool) ([]any, error) {
	result := make([]any, 0, len(actions))
	for index, raw := range actions {
		action := map[string]any{}
		targetKey := FirstString(raw, actionAliasConfig.TargetKeys...)
		if targetKey == "" {
			return nil, fmt.Errorf("actions[].targetKey is required")
		}
		targetType := FirstString(raw, actionAliasConfig.TargetTypes...)
		if targetType == "" {
			targetType = builder.targetTypeForKey(targetKey)
		}
		typeID, ok := TargetTypeID(targetType, ResourceMeshGroup)
		if !ok {
			return nil, fmt.Errorf("actions[].targetType is required")
		}
		action[internalTypeID] = typeID
		action[internalTempID] = targetKey
		if name := FirstString(raw, actionAliasConfig.TargetNames...); name != "" {
			action[internalResourceName] = name
		}
		if action[internalResourceName] == nil {
			action[internalResourceName] = builder.nameForKey(targetKey)
		}
		if value, ok := raw[FieldRank]; ok {
			action[FieldRank] = deepValue(value)
		}
		if action[FieldRank] == nil {
			action[FieldRank] = index
		}
		if value, ok := raw[FieldAction]; ok {
			action[FieldAction] = deepValue(value)
		}
		if scene && action[FieldAction] == nil {
			action[FieldAction] = 0
		}
		if params, ok := ActionParamsFromRow(raw); ok {
			action[internalParams] = params
		}
		RemoveActionRowAliases(action)
		result = append(result, action)
	}
	return result, nil
}

func (builder *lightingDesignBuilder) remember(refs map[string]string, key string, name string, raw map[string]any) {
	refs[key] = name
	for _, alias := range []string{"id", FieldKey, "ref"} {
		if value := String(raw[alias]); value != "" {
			refs[value] = name
		}
	}
}

func (builder *lightingDesignBuilder) targetTypeForKey(key string) string {
	switch {
	case builder.slotRefs[key] != "":
		return "device"
	case builder.groupRefs[key] != "":
		return "group"
	case builder.roomRefs[key] != "":
		return "room"
	case builder.sceneRefs[key] != "":
		return "scene"
	default:
		return ""
	}
}

func (builder *lightingDesignBuilder) nameForKey(key string) string {
	for _, refs := range []map[string]string{builder.slotRefs, builder.groupRefs, builder.roomRefs, builder.sceneRefs} {
		if refs[key] != "" {
			return refs[key]
		}
	}
	return ""
}

func firstAny(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func hasAnyKey(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := values[key]; ok {
			return true
		}
	}
	return false
}

func firstKey(values map[string]any, fallback string) string {
	if value := FirstString(values, FieldKey, "ref", "id"); value != "" {
		return value
	}
	return fallback
}

func mergeMissing(target map[string]any, source map[string]any) {
	for key, value := range source {
		if _, ok := target[key]; !ok {
			target[key] = value
		}
	}
}

func NormalizeProduct(source map[string]any) map[string]any {
	product := map[string]any{}
	if value, ok := FirstPresent(source, actionAliasConfig.ProductCodes...); ok {
		product[internalProductCode] = value
	}
	if value, ok := FirstPresent(source, actionAliasConfig.ProductIDs...); ok {
		product[internalProductID] = value
	}
	if value, ok := FirstPresent(source, actionAliasConfig.ProductCategoryIDs...); ok {
		product[internalProductCategoryID] = value
	}
	for _, key := range []string{FieldProductName, FieldProductSKU, FieldProductSPU, FieldProductBrand, FieldProductModel, FieldProductLine, FieldCategory, FieldSeries, FieldNotes, FieldConnectType} {
		if value, ok := source[key]; ok {
			product[key] = deepValue(value)
		}
	}
	removeProductAliases(product)
	return product
}

func removeProductAliases(product map[string]any) {
	for _, key := range actionAliasConfig.ProductCodes {
		delete(product, key)
	}
	for _, key := range actionAliasConfig.ProductIDs {
		delete(product, key)
	}
	for _, key := range actionAliasConfig.ProductCategoryIDs {
		delete(product, key)
	}
}

func stringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := []string{}
		for _, item := range typed {
			if text := String(item); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		if text := String(value); text != "" {
			return []string{text}
		}
	}
	return nil
}
