package main

import (
	"context"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) prepareLightingDesignImport(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	houseID = lightingDesignImportTargetHouseID(request, request.Intent, houseID)
	payload := request.Parameters
	if payload == nil {
		payload = map[string]any{}
	}
	normalized, err := api.NormalizeLightingDesignImportPayload(houseID, payload)
	if err != nil {
		return configureClarificationResponseWithGuide(request, "invalid_lighting_design_import_payload", lightingDesignImportAcceptedFields(), lightingDesignImportPayloadGuide()), nil
	}
	entities := api.EntityListResult{Region: endpoint.Region}
	if strings.TrimSpace(houseID) != "" {
		loaded, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: authorization,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		entities = loaded
	}
	intent := request.Intent
	summary := "导入照明设计并预建设备槽位"
	if intent == "device.slot.create" {
		summary = "创建设备预留槽位"
	}
	if strings.TrimSpace(houseID) == "" && intent == "device.slot.create" {
		return configureClarificationResponse(request, "missing_house_id_for_device_slot_create", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldName),
			"local profile houseId",
		}), nil
	}
	if intent == "device.slot.create" && strings.TrimSpace(houseID) != "" {
		if duplicates := deviceSlotCreateExistingRoomCandidates(normalized, entities); len(duplicates) > 0 {
			return configureClarificationResponseWithCandidates(
				request,
				"device_slot_create_existing_room_would_duplicate_room",
				[]string{
					semantic.ParameterPath(semantic.FieldHouseID),
					semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldName),
					semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldDeviceSlots),
				},
				map[string]any{
					semantic.FieldNextStep: "device.slot.create creates the supplied rooms as design rooms. Use a new room name, or use a dedicated room/device-slot capability when Runtime exposes safe append-to-existing-room support.",
				},
				duplicates,
			), nil
		}
	}
	preconditions := []string{
		"已有家庭导入前重新读取家庭实体列表；新家庭导入由 Runtime 创建家庭并返回 houseId",
		"调用方提交标准照明设计模型，包含房间、设备槽位、灯组、情景和自动化",
		"设备槽位代表照明设计占位，不代表设备已配网或可被真实控制",
		"执行后通过家庭实体列表验证房间和设备槽位可见",
	}
	risk := operation.RiskR2
	recordHouseID := strings.TrimSpace(houseID)
	if recordHouseID == "" {
		recordHouseID = operation.AccountScopeHouseID
	}
	record, err := operation.NewPreparedWithRisk(profile, region, recordHouseID, intent, request.RequestID, summary, risk, normalized, preconditions, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		semantic.FieldMode:                "lighting_design_import",
		semantic.FieldCounts:              lightingDesignImportPayloadCounts(normalized),
		semantic.FieldProductResolution:   lightingDesignProductResolutionPreview(normalized),
		semantic.FieldPersistentWrites:    true,
		semantic.FieldCreatesDeviceSlots:  true,
		semantic.FieldDeviceSlotsPhysical: false,
		semantic.FieldTargetMode:          lightingDesignImportTargetMode(houseID),
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func lightingDesignImportTargetHouseID(request contract.Request, intent string, fallback string) string {
	if houseID := requestHouseID(request); houseID != "" {
		return houseID
	}
	if intent == "lighting.design.import" {
		if requestBool(request.HomeRef, semantic.FieldUseCurrent) {
			return strings.TrimSpace(fallback)
		}
		return ""
	}
	if strings.TrimSpace(firstRequestString(request.HomeRef, semantic.FieldName, semantic.FieldHouseName)) != "" {
		return strings.TrimSpace(fallback)
	}
	if intent == "device.slot.create" {
		return strings.TrimSpace(fallback)
	}
	return ""
}

func lightingDesignImportTargetMode(houseID string) string {
	if strings.TrimSpace(houseID) == "" || operation.IsAccountScope(houseID) {
		return "create_new_home"
	}
	return "import_into_existing_home"
}

func deviceSlotCreateExistingRoomCandidates(payload map[string]any, entities api.EntityListResult) []map[string]any {
	if len(entities.Entities) == 0 {
		return nil
	}
	requested := lightingDesignRoomNames(payload)
	if len(requested) == 0 {
		return nil
	}
	candidates := make([]map[string]any, 0)
	seen := map[string]bool{}
	for _, entity := range entities.Entities {
		if entity.Type != "room" {
			continue
		}
		for _, name := range requested {
			if semantic.NameMatchAutoAccept(semantic.ScoreNameMatch(name, entity.Name), semantic.NameMatch{}) {
				key := entity.Type + ":" + entity.ID
				if !seen[key] {
					candidates = append(candidates, entitySummaryMap(entity))
					seen[key] = true
				}
			}
		}
	}
	return candidates
}

func lightingDesignRoomNames(payload map[string]any) []string {
	gateway, ok := payload[semantic.FieldGateway].(map[string]any)
	if !ok {
		return nil
	}
	rawRooms, ok := gateway[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)].([]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(rawRooms))
	for _, rawRoom := range rawRooms {
		room, ok := rawRoom.(map[string]any)
		if !ok {
			continue
		}
		if name := strings.TrimSpace(requestString(room[semantic.FieldName])); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func lightingDesignImportAcceptedFields() []string {
	return []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semantic.ParameterPath(semantic.FieldKey),
		semantic.ParameterPath(semantic.FieldName),
		semantic.ParameterPath(semantic.FieldGatewayName),
		semantic.ParameterPath(semantic.FieldGatewayDeviceID),
		semantic.ParameterPath(semantic.FieldRooms),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldKey),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldDeviceSlots),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldKey),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct, semantic.FieldSKUCode),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct, semantic.FieldCapabilityProductID),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct, semantic.FieldProductComponentID),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldDeviceSlots), semantic.FieldProduct, semantic.FieldProductName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.FieldGroups),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldKey),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldGroupCategory),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldGroupCapability),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldRooms), semantic.ArrayField(semantic.FieldGroups), semantic.FieldSlotKeys),
		semantic.ParameterPath(semantic.FieldAreas),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAreas), semantic.FieldRoomKeys),
		semantic.ParameterPath(semantic.FieldScenes),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.FieldActions),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetType),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetKey),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldScenes), semantic.ArrayField(semantic.FieldActions), semantic.FieldSet),
		semantic.ParameterPath(semantic.FieldAutomations),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldActiveWindow),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldRepeat),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldTrigger),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldConditions),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.FieldActions),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetType),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetKey),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.ArrayField(semantic.FieldActions), semantic.FieldTargetName),
		semantic.ParameterPath(semantic.ArrayField(semantic.FieldAutomations), semantic.ArrayField(semantic.FieldActions), semantic.FieldSet),
	}
}

func (app *app) executeLightingDesignImport(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	houseID := record.HouseID
	if operation.IsAccountScope(houseID) {
		houseID = ""
	}
	result, err := api.NewLightingDesignImportClient(endpoint, nil).Run(ctx, api.LightingDesignImportRequest{
		HouseID:        houseID,
		Intent:         record.Intent,
		Payload:        record.Payload,
		VerifyAttempts: 12,
		VerifyInterval: time.Second,
		Credentials: api.LightingDesignImportCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return app.lightingDesignImportFailureResponse(ctx, request, endpoint, record, authorization, clientID, err), nil
	}
	if err := app.selectImportedHouse(record.Profile, record.Region, record.HouseID, result.HouseID); err != nil {
		return contract.Response{}, err
	}
	return lightingDesignImportExecuteResponse(request, record, result), nil
}

func (app *app) lightingDesignImportFailureResponse(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string, cause error) contract.Response {
	credentials := api.HomeSummaryCredentials{Authorization: authorization, ClientID: clientID}
	houseID, houses, readbackCalls, readbackWarnings := lightingDesignImportFailureTargetHouse(ctx, endpoint, record, credentials)
	entityCalls := 0
	entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID}
	if strings.TrimSpace(houseID) != "" {
		loaded, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
			HouseID: houseID,
			Credentials: api.EntityListCredentials{
				Authorization: authorization,
				ClientID:      clientID,
			},
		})
		entities = loaded
		entityCalls = firstPositive(loaded.APICalls, api.HouseScopedEntityListCallCount())
		if err != nil {
			readbackWarnings = append(readbackWarnings, "entity_list_readback_failed")
		}
	}
	partialState := map[string]any{
		semantic.FieldError:          cause.Error(),
		semantic.FieldExpectedCounts: lightingDesignImportPayloadCounts(record.Payload),
		semantic.FieldTargetMode:     lightingDesignImportTargetMode(record.HouseID),
	}
	if strings.TrimSpace(houseID) != "" {
		partialState[semantic.FieldHouseID] = houseID
		partialState[semantic.FieldTotal] = entities.Total
		partialState[semantic.FieldObservedCounts] = entities.Counts
	}
	if len(houses) > 0 {
		partialState[semantic.FieldHouses] = lightingDesignHouseCandidates(houses)
	}
	response := contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "partial",
		UserMessage:     "照明设计导入未完全验证；Runtime 已自动读回可能的部分写入状态。",
		Result: map[string]any{
			semantic.FieldRegion:              endpoint.Region,
			semantic.FieldHouseID:             houseID,
			semantic.FieldCapability:          record.Intent,
			semantic.FieldVerified:            false,
			semantic.FieldVerifiedBy:          "partial_readback_after_import_failure",
			semantic.FieldPartialState:        partialState,
			semantic.FieldPersistentWrites:    true,
			semantic.FieldDeviceSlotsPhysical: false,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "partial",
		},
		Warnings: append([]string{"lighting_design_import_not_fully_verified"}, readbackWarnings...),
		TraceID:  "lighting-design-import-partial",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  readbackCalls + entityCalls,
			semantic.FieldCacheHits: 0,
		},
		Error: &contract.Error{
			Code:    "lighting_design_import_not_fully_verified",
			Message: cause.Error(),
		},
	}
	return responseWithVerifiedTopology(response, entities)
}

func lightingDesignImportFailureTargetHouse(ctx context.Context, endpoint api.Endpoint, record operation.Prepared, credentials api.HomeSummaryCredentials) (string, []api.HouseSummary, int, []string) {
	if houseID := strings.TrimSpace(record.HouseID); houseID != "" && !operation.IsAccountScope(houseID) {
		return houseID, nil, 0, nil
	}
	name := strings.TrimSpace(stringFromMap(record.Payload, semantic.FieldName))
	if name == "" {
		return "", nil, 0, []string{"target_home_name_unavailable_for_readback"}
	}
	summary, err := api.NewHomeSummaryClient(endpoint, nil).RunSearch(ctx, map[string]any{
		semantic.FieldName:  name,
		semantic.FieldLimit: 5,
	}, credentials)
	if err != nil {
		return "", nil, firstPositive(summary.APICalls, 1), []string{"home_search_readback_failed"}
	}
	ranked := semantic.RankNameMatches(name, summary.Houses, func(house api.HouseSummary) string {
		return house.Name
	})
	if len(ranked) == 0 {
		return "", summary.Houses, firstPositive(summary.APICalls, 1), []string{"target_home_not_found_after_import_failure"}
	}
	second := semantic.NameMatch{}
	if len(ranked) > 1 {
		second = ranked[1].Match
	}
	if semantic.NameMatchAutoAccept(ranked[0].Match, second) {
		return ranked[0].Value.ID, summary.Houses, firstPositive(summary.APICalls, 1), nil
	}
	return "", summary.Houses, firstPositive(summary.APICalls, 1), []string{"target_home_ambiguous_after_import_failure"}
}

func lightingDesignHouseCandidates(houses []api.HouseSummary) []any {
	result := make([]any, 0, len(houses))
	for _, house := range houses {
		item := map[string]any{
			semantic.FieldHouseID: house.ID,
			semantic.FieldID:      house.ID,
			semantic.FieldName:    house.Name,
		}
		if len(house.Counts) > 0 {
			item[semantic.FieldCounts] = house.Counts
		}
		result = append(result, item)
	}
	return result
}

func (app *app) selectImportedHouse(profile string, region string, recordHouseID string, resultHouseID string) error {
	if strings.TrimSpace(resultHouseID) == "" || (strings.TrimSpace(recordHouseID) != "" && !operation.IsAccountScope(recordHouseID)) {
		return nil
	}
	metadata, _, err := app.metadataStore.Load(profile)
	if err != nil {
		return err
	}
	metadata = mergeProfileMetadata(metadata, profile, map[string]string{
		semantic.FieldRegion:  region,
		semantic.FieldHouseID: resultHouseID,
	})
	if metadata.Region == "" {
		metadata.Region = defaultRuntimeRegion
	}
	return app.metadataStore.Save(metadata)
}

func lightingDesignProductResolutionPreview(payload map[string]any) map[string]any {
	gateway, ok := payload[semantic.FieldGateway].(map[string]any)
	if !ok {
		return nil
	}
	matched := 0
	unresolved := 0
	samples := []any{}
	rooms, _ := gateway[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)].([]any)
	for _, rawRoom := range rooms {
		room, ok := rawRoom.(map[string]any)
		if !ok {
			continue
		}
		devices, _ := room[semantic.InternalField(semantic.DomainImport, semantic.FieldDeviceSlots)].([]any)
		for _, rawDevice := range devices {
			device, ok := rawDevice.(map[string]any)
			if !ok {
				continue
			}
			extra, _ := device[semantic.FieldExtraMeta].(map[string]any)
			item := map[string]any{
				semantic.FieldName:                device[semantic.FieldName],
				semantic.FieldCapabilityProductID: device[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)],
				semantic.FieldTargetKey:           device[semantic.InternalField(semantic.DomainImport, semantic.FieldKey)],
			}
			if value, ok := extra[semantic.InternalField(semantic.DomainProduct, semantic.FieldProductCode)]; ok {
				item[semantic.FieldSKUCode] = value
			}
			if value, ok := extra[semantic.FieldProductName]; ok {
				item[semantic.FieldProductName] = value
			}
			if device[semantic.InternalField(semantic.DomainProduct, semantic.FieldCapabilityProductID)] != nil {
				matched++
			} else {
				unresolved++
			}
			if len(samples) < 8 {
				samples = append(samples, item)
			}
		}
	}
	return map[string]any{
		semantic.FieldMatchedDeviceSlots:    matched,
		semantic.FieldUnresolvedDeviceSlots: unresolved,
		semantic.FieldCatalog:               "skill_selected_house_meta_products",
		semantic.FieldSamples:               samples,
	}
}

func lightingDesignImportPayloadCounts(payload map[string]any) map[string]int {
	counts := map[string]int{
		semantic.FieldGateways:    0,
		semantic.FieldRooms:       0,
		semantic.FieldDevices:     0,
		semantic.FieldGroups:      0,
		semantic.FieldAreas:       0,
		semantic.FieldScenes:      0,
		semantic.FieldAutomations: 0,
	}
	if areas, ok := payload[semantic.InternalField(semantic.DomainImport, semantic.FieldAreas)].([]any); ok {
		counts[semantic.FieldAreas] = len(areas)
	}
	if scenes, ok := payload[semantic.InternalField(semantic.DomainImport, semantic.FieldScenes)].([]any); ok {
		counts[semantic.FieldScenes] = len(scenes)
	}
	if automations, ok := payload[semantic.InternalField(semantic.DomainImport, semantic.FieldAutomations)].([]any); ok {
		counts[semantic.FieldAutomations] = len(automations)
	}
	gateway, ok := payload[semantic.FieldGateway].(map[string]any)
	if !ok {
		return counts
	}
	counts[semantic.FieldGateways] = 1
	rooms, _ := gateway[semantic.InternalField(semantic.DomainImport, semantic.FieldRooms)].([]any)
	counts[semantic.FieldRooms] = len(rooms)
	for _, rawRoom := range rooms {
		room, ok := rawRoom.(map[string]any)
		if !ok {
			continue
		}
		devices, _ := room[semantic.InternalField(semantic.DomainImport, semantic.FieldDeviceSlots)].([]any)
		groups, _ := room[semantic.InternalField(semantic.DomainImport, semantic.FieldGroups)].([]any)
		counts[semantic.FieldDevices] += len(devices)
		counts[semantic.FieldGroups] += len(groups)
	}
	return counts
}

func lightingDesignImportExecuteResponse(request contract.Request, record operation.Prepared, result api.LightingDesignImportResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已导入并验证照明设计，设备槽位已作为预建设计占位写入家庭。",
		Result: map[string]any{
			semantic.FieldRegion:              result.Region,
			semantic.FieldHouseID:             result.HouseID,
			semantic.FieldCapability:          result.Capability,
			semantic.FieldMode:                result.Mode,
			semantic.FieldCounts:              result.Counts,
			semantic.FieldMappings:            result.Mappings,
			semantic.FieldRequestKey:          result.RequestKey,
			semantic.FieldVerified:            result.Verified,
			semantic.FieldVerifiedBy:          result.VerifiedBy,
			semantic.FieldTargetMode:          lightingDesignImportTargetMode(record.HouseID),
			semantic.FieldSelectedHouseID:     selectedImportedHouseID(record.HouseID, result.HouseID),
			semantic.FieldPersistentWrites:    true,
			semantic.FieldDeviceSlotsPhysical: false,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: result.Warnings,
		TraceID:  "lighting-design-import-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, result.VerifiedEntities)
}

func selectedImportedHouseID(recordHouseID string, resultHouseID string) string {
	if operation.IsAccountScope(recordHouseID) {
		return strings.TrimSpace(resultHouseID)
	}
	return ""
}
