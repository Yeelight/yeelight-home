package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) invokeAccountInfo(ctx context.Context, request contract.Request, endpoint api.Endpoint, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewAccountInfoClient(endpoint, nil).Run(ctx, api.AccountInfoCredentials{
		Authorization: authorization,
		ClientID:      clientID,
	})
	if err != nil {
		return contract.Response{}, err
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已读取本机登录账号的脱敏信息。",
		Result: map[string]any{
			semantic.FieldRegion:   result.Region,
			semantic.FieldAccount:  result.Summary,
			semantic.FieldRawShape: result.RawShape,
		},
		Warnings: []string{},
		TraceID:  "account-info-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}, nil
}

func (app *app) invokeMetadataReadonlyProjection(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, spec metadataReadonlySpec) (contract.Response, error) {
	if requestRunsHouseIndependent(request) {
		houseID = ""
	} else if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if len(spec.entityTypes) == 0 {
		return metadataReadonlyProjectionResponse(request, spec, endpoint.Region, houseID, nil, noAPIMetrics()), nil
	}
	entities, err := api.NewEntityListClient(endpoint, nil).Run(ctx, api.EntityListRequest{
		HouseID: houseID,
		Credentials: api.EntityListCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return metadataReadonlyProjectionResponse(request, spec, entities.Region, entities.HouseID, metadataEntityEvidence(entities.Entities, spec.entityTypes, spec.limit), map[string]any{
		semantic.FieldAPICalls:  entityListAPICalls(entities),
		semantic.FieldCacheHits: 0,
	}), nil
}

func (app *app) invokeMetadataCloudReadonly(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, spec metadataReadonlySpec) (contract.Response, error) {
	if requestRunsHouseIndependent(request) {
		houseID = ""
	} else if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	enrichedRequest, resolutionCalls, clarification, resolveErr := app.enrichMetadataReadonlyTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID)
	if clarification != nil || resolveErr != nil {
		return responseOrError(clarification, resolveErr)
	}
	request = enrichedRequest
	if request.Intent == "home.sort.list" {
		enrichedRequest, sortResolutionCalls, sortClarification, sortResolveErr := app.enrichHomeSortListTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID)
		resolutionCalls += sortResolutionCalls
		if sortClarification != nil || sortResolveErr != nil {
			return responseOrError(sortClarification, sortResolveErr)
		}
		request = enrichedRequest
	}
	target := entityGetTargetFromRequest(request)
	deviceID := firstNonEmptyString(target.id, firstRequestString(request.Parameters, semantic.FieldDeviceID))
	client := api.NewMetadataReadonlyClient(endpoint, nil)
	readonlyRequest := api.MetadataReadonlyRequest{
		HouseID:    houseID,
		DeviceID:   deviceID,
		Utterance:  request.Utterance,
		Parameters: request.Parameters,
		Credentials: api.MetadataReadonlyCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	}
	var result api.MetadataReadonlyResult
	var err error
	switch request.Intent {
	case "home.member.list":
		result, err = client.RunHomeMemberList(ctx, readonlyRequest)
	case "home.member.current.get":
		result, err = client.RunHomeMemberCurrentGet(ctx, readonlyRequest)
	case "device.detail.get":
		result, err = client.RunDeviceDetailGet(ctx, readonlyRequest)
	case "device.complex.get":
		result, err = client.RunDeviceComplexGet(ctx, readonlyRequest)
	case "device.shadow.get":
		result, err = client.RunDeviceShadowGet(ctx, readonlyRequest)
	case "device.attr.list":
		result, err = client.RunDeviceAttrList(ctx, readonlyRequest)
	case "device.list":
		result, err = client.RunDeviceList(ctx, readonlyRequest)
	case "room.detail.get":
		result, err = client.RunRoomDetailGet(ctx, readonlyRequest)
	case "room.list":
		result, err = client.RunRoomList(ctx, readonlyRequest)
	case "room.search":
		result, err = client.RunRoomSearch(ctx, readonlyRequest)
	case "area.detail.get":
		result, err = client.RunAreaDetailGet(ctx, readonlyRequest)
	case "area.list":
		result, err = client.RunAreaList(ctx, readonlyRequest)
	case "home.detail.get":
		result, err = client.RunHomeDetailGet(ctx, readonlyRequest)
	case "home.property.get":
		result, err = client.RunHomePropertyGet(ctx, readonlyRequest)
	case "home.stat.get":
		result, err = client.RunHomeStatGet(ctx, readonlyRequest)
	case "geo_area.children.list":
		result, err = client.RunGeoAreaChildrenList(ctx, readonlyRequest)
	case "geo_area.search":
		result, err = client.RunGeoAreaSearch(ctx, readonlyRequest)
	case "group.structure.list":
		result, err = client.RunGroupStructureList(ctx, readonlyRequest)
	case "group.list":
		result, err = client.RunGroupList(ctx, readonlyRequest)
	case "group.search":
		result, err = client.RunGroupSearch(ctx, readonlyRequest)
	case "group.detail.get":
		result, err = client.RunGroupDetailGet(ctx, readonlyRequest)
	case "group.complex.get":
		result, err = client.RunGroupComplexGet(ctx, readonlyRequest)
	case "scene.detail.get":
		result, err = client.RunSceneDetailGet(ctx, readonlyRequest)
	case "scene.list":
		result, err = client.RunSceneList(ctx, readonlyRequest)
	case "scene.scoped.list":
		result, err = client.RunSceneScopedList(ctx, readonlyRequest)
	case "scene.search":
		result, err = client.RunSceneSearch(ctx, readonlyRequest)
	case "automation.list":
		result, err = client.RunAutomationList(ctx, readonlyRequest)
	case "automation.supported.list":
		result, err = client.RunAutomationSupportedList(ctx, readonlyRequest, false)
	case "automation.supported.v2.list":
		result, err = client.RunAutomationSupportedList(ctx, readonlyRequest, true)
	case "automation.rule.list":
		result, err = client.RunAutomationRuleList(ctx, readonlyRequest)
	case "automation.list.page":
		result, err = client.RunAutomationListPage(ctx, readonlyRequest)
	case "automation.detail.get":
		result, err = client.RunAutomationDetailGet(ctx, readonlyRequest)
	case "schedule_job.list":
		result, err = client.RunScheduleJobList(ctx, readonlyRequest)
	case "message.list":
		result, err = client.RunMessageList(ctx, readonlyRequest)
	case "sensor.list":
		result, err = client.RunSensorList(ctx, readonlyRequest)
	case "sensor.event.list":
		result, err = client.RunSensorEventList(ctx, readonlyRequest)
	case "device.energy.summary":
		result, err = client.RunDeviceEnergySummary(ctx, readonlyRequest)
	case "device.weather.get":
		result, err = client.RunDeviceWeatherGet(ctx, readonlyRequest)
	case "device.virtual_count.get":
		result, err = client.RunDeviceVirtualCountGet(ctx, readonlyRequest)
	case "meshgroup.detail.get":
		result, err = client.RunMeshgroupDetailGet(ctx, readonlyRequest)
	case "node.sorted_device.list":
		result, err = client.RunNodeSortedDeviceList(ctx, readonlyRequest)
	case "gateway.detail.get":
		result, err = client.RunGatewayDetailGet(ctx, readonlyRequest)
	case "gateway.list":
		result, err = client.RunGatewayList(ctx, readonlyRequest)
	case "gateway.thread.get":
		result, err = client.RunGatewayThreadGet(ctx, readonlyRequest)
	case "gateway.stats.list":
		result, err = client.RunGatewayStatsList(ctx, readonlyRequest)
	case "gateway.scene_relation.list":
		result, err = client.RunGatewaySceneRelationList(ctx, readonlyRequest)
	case "panel.get":
		result, err = client.RunPanelGet(ctx, readonlyRequest)
	case "panel.list":
		result, err = client.RunPanelList(ctx, readonlyRequest)
	case "panel.button.type.get":
		result, err = client.RunPanelButtonTypeGet(ctx, readonlyRequest)
	case "screen.control.list":
		result, err = client.RunScreenControlList(ctx, readonlyRequest)
	case "knob.get":
		result, err = client.RunKnobGet(ctx, readonlyRequest)
	case "upgrade.file.list":
		result, err = client.RunUpgradeFileList(ctx, readonlyRequest)
	case "upgrade.progress.get":
		result, err = client.RunUpgradeProgressGet(ctx, readonlyRequest)
	case "upgrade.file.batch_list":
		result, err = client.RunUpgradeFileBatchList(ctx, readonlyRequest)
	case "progress.get":
		result, err = client.RunProgressGet(ctx, readonlyRequest)
	case "app_upgrade.latest.get":
		result, err = client.RunAppUpgradeLatestGet(ctx, readonlyRequest)
	case "ota.version_file.batch_list":
		result, err = client.RunOTAVersionFileBatchList(ctx, readonlyRequest)
	case "node.property_config.get":
		result, err = client.RunNodePropertyConfigGet(ctx, readonlyRequest)
	case "thing.schema.list":
		result, err = client.RunThingSchemaList(ctx, readonlyRequest)
	case "thing.schema.detail.list":
		result, err = client.RunThingSchemaDetailList(ctx, readonlyRequest)
	case "thing.schema.get":
		result, err = client.RunThingSchemaGet(ctx, readonlyRequest)
	case "thing.schema.event.list":
		result, err = client.RunThingSchemaEventList(ctx, readonlyRequest)
	case "thing.product.info.batch_get":
		result, err = client.RunThingProductInfoBatchGet(ctx, readonlyRequest)
	case "thing.product.info.v3.batch_get":
		result, err = client.RunThingProductInfoV3BatchGet(ctx, readonlyRequest)
	case "thing.product.list.v3":
		result, err = client.RunThingProductListV3(ctx, readonlyRequest)
	case "product.pedia.search":
		result, err = client.RunProductPediaSearch(ctx, readonlyRequest)
	case "thing.product_domain.list":
		result, err = client.RunProductDomainList(ctx, readonlyRequest)
	case "thing.product_faq.list":
		result, err = client.RunProductFAQList(ctx, readonlyRequest)
	case "thing.product_faq.detail.get":
		result, err = client.RunProductFAQDetailGet(ctx, readonlyRequest)
	case "thing.product_faq.type.list":
		result, err = client.RunProductFAQTypeList(ctx, readonlyRequest)
	case "thing.product_faq.item_type.list":
		result, err = client.RunProductFAQItemTypeList(ctx, readonlyRequest)
	case "thing.product_faq.locale.list":
		result, err = client.RunProductFAQLocaleList(ctx, readonlyRequest)
	case "thing.product_faq.page.list":
		result, err = client.RunProductFAQPageList(ctx, readonlyRequest)
	case "thing.product_faq.page_detail.list":
		result, err = client.RunProductFAQPageDetailList(ctx, readonlyRequest)
	case "thing.category.list":
		result, err = client.RunThingCategoryList(ctx, readonlyRequest)
	case "thing.component.list":
		result, err = client.RunThingComponentList(ctx, readonlyRequest)
	case "thing.component.get":
		result, err = client.RunThingComponentGet(ctx, readonlyRequest)
	case "thing.property.list":
		result, err = client.RunThingPropertyList(ctx, readonlyRequest)
	case "favorite.list":
		result, err = client.RunFavoriteList(ctx, readonlyRequest)
	case "home.sort.list":
		result, err = client.RunHomeSortList(ctx, readonlyRequest)
	case "ai_voice.product.list":
		result, err = client.RunAIVoiceProductList(ctx, readonlyRequest)
	default:
		return metadataReadonlyProjectionResponse(request, spec, endpoint.Region, houseID, nil, noAPIMetrics()), nil
	}
	if err != nil {
		return contract.Response{}, err
	}
	result.APICalls += resolutionCalls
	return metadataCloudReadonlyResponse(request, spec, result), nil
}

func (app *app) enrichMetadataReadonlyTarget(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Request, int, *contract.Response, error) {
	if !metadataReadonlyIntentNeedsEntityID(request.Intent) {
		return request, 0, nil, nil
	}
	target := entityGetTargetFromRequest(request)
	if target.entityType == "" {
		return request, 0, nil, nil
	}
	if target.id != "" {
		return requestWithResolvedMetadataTarget(request, target.entityType, target.id), 0, nil, nil
	}
	if target.name == "" {
		return request, 0, nil, nil
	}
	if target.entityType == "gateway" {
		match, candidates, calls, fallbackErr := resolveGatewayFromReadonlyList(ctx, endpoint, houseID, authorization, clientID, target.name)
		if fallbackErr != nil {
			return request, calls, nil, fallbackErr
		}
		if match.ID != "" {
			return requestWithResolvedMetadataTarget(request, "gateway", match.ID), calls, nil, nil
		}
		if len(candidates) > 0 {
			response := entityGetClarificationResponse(request, "ambiguous_target", target, candidates, calls)
			return request, calls, &response, nil
		}
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return request, 0, nil, err
	}
	if resolved.Match.ID == "" {
		response := entityGetClarificationResponse(request, "entity_not_found", target, resolved.Candidates, entityListAPICalls(resolved.Entities))
		return request, entityListAPICalls(resolved.Entities), &response, nil
	}
	if len(resolved.Candidates) > 1 {
		response := entityGetClarificationResponse(request, "ambiguous_target", target, resolved.Candidates, entityListAPICalls(resolved.Entities))
		return request, entityListAPICalls(resolved.Entities), &response, nil
	}
	entityType := firstNonEmptyString(target.entityType, resolved.Match.Type)
	return requestWithResolvedMetadataTarget(request, entityType, resolved.Match.ID), entityListAPICalls(resolved.Entities), nil, nil
}

func (app *app) enrichHomeSortListTarget(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Request, int, *contract.Response, error) {
	if strings.TrimSpace(houseID) == "" {
		return request, 0, nil, nil
	}
	parameters := copyRequestParameters(request.Parameters)
	totalCalls := 0
	for _, target := range []struct {
		entityType string
		idField    string
		nameFields []string
	}{
		{entityType: "room", idField: semantic.FieldRoomID, nameFields: []string{semantic.FieldRoomName, semantic.FieldTargetRoomName}},
		{entityType: "area", idField: semantic.FieldAreaID, nameFields: []string{semantic.FieldAreaName}},
		{entityType: "device", idField: semantic.FieldDeviceID, nameFields: []string{semantic.FieldDeviceName}},
	} {
		if firstRequestString(parameters, target.idField) != "" {
			continue
		}
		name := firstRequestString(parameters, target.nameFields...)
		if name == "" {
			continue
		}
		entityTarget := entityGetTarget{name: name, entityType: target.entityType}
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, entityTarget)
		if err != nil {
			return request, totalCalls, nil, err
		}
		calls := entityListAPICalls(resolved.Entities)
		totalCalls += calls
		if resolved.Match.ID == "" {
			response := entityGetClarificationResponse(request, "entity_not_found", entityTarget, resolved.Candidates, calls)
			return request, totalCalls, &response, nil
		}
		if len(resolved.Candidates) > 1 {
			response := entityGetClarificationResponse(request, "ambiguous_target", entityTarget, resolved.Candidates, calls)
			return request, totalCalls, &response, nil
		}
		parameters[target.idField] = resolved.Match.ID
	}
	request.Parameters = parameters
	return request, totalCalls, nil, nil
}

func resolveGatewayFromReadonlyList(ctx context.Context, endpoint api.Endpoint, houseID string, authorization string, clientID string, name string) (api.EntitySummary, []api.EntitySummary, int, error) {
	if strings.TrimSpace(houseID) == "" || strings.TrimSpace(name) == "" {
		return api.EntitySummary{}, nil, 0, nil
	}
	result, err := api.NewMetadataReadonlyClient(endpoint, nil).RunGatewayList(ctx, api.MetadataReadonlyRequest{
		HouseID: houseID,
		Credentials: api.MetadataReadonlyCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return api.EntitySummary{}, nil, result.APICalls, err
	}
	gateways := gatewayEntitiesFromReadonlyResult(result)
	ranked := semantic.RankNameMatches(name, gateways, func(entity api.EntitySummary) string {
		return entity.Name
	})
	if len(ranked) == 0 {
		return api.EntitySummary{}, nil, result.APICalls, nil
	}
	if ranked[0].Match.Kind == "name" {
		exact := entityMatchCandidatesByKind(ranked, "name")
		if len(exact) == 1 {
			return exact[0], exact, result.APICalls, nil
		}
		return api.EntitySummary{}, exact, result.APICalls, nil
	}
	second := semantic.NameMatch{}
	if len(ranked) > 1 {
		second = ranked[1].Match
	}
	if semantic.NameMatchAutoAccept(ranked[0].Match, second) {
		return ranked[0].Value, []api.EntitySummary{ranked[0].Value}, result.APICalls, nil
	}
	return api.EntitySummary{}, entityMatchCandidates(ranked), result.APICalls, nil
}

func gatewayEntitiesFromReadonlyResult(result api.MetadataReadonlyResult) []api.EntitySummary {
	data, ok := result.Data.(map[string]any)
	if !ok {
		return nil
	}
	rows, ok := data[semantic.FieldGateways].([]any)
	if !ok {
		return nil
	}
	gateways := make([]api.EntitySummary, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(requestString(item[semantic.FieldID]))
		name := strings.TrimSpace(requestString(item[semantic.FieldName]))
		if id == "" || name == "" {
			continue
		}
		gateways = append(gateways, api.EntitySummary{
			Type:    "gateway",
			ID:      id,
			Name:    name,
			HouseID: strings.TrimSpace(requestString(item[semantic.FieldHouseID])),
			RoomID:  strings.TrimSpace(requestString(item[semantic.FieldRoomID])),
		})
	}
	return gateways
}

func metadataReadonlyIntentNeedsEntityID(intent string) bool {
	switch intent {
	case "device.detail.get",
		"device.attr.list",
		"device.energy.summary",
		"device.weather.get",
		"room.detail.get",
		"scene.scoped.list",
		"area.detail.get",
		"group.detail.get",
		"scene.detail.get",
		"automation.detail.get",
		"meshgroup.detail.get",
		"gateway.detail.get",
		"gateway.thread.get",
		"gateway.scene_relation.list",
		"panel.get",
		"panel.button.type.get",
		"screen.control.list",
		"knob.get",
		"node.sorted_device.list",
		"node.property_config.get",
		"upgrade.file.list",
		"upgrade.progress.get",
		"upgrade.file.batch_list":
		return true
	default:
		return false
	}
}

func requestWithResolvedMetadataTarget(request contract.Request, entityType string, entityID string) contract.Request {
	if strings.TrimSpace(entityID) == "" {
		return request
	}
	parameters := copyRequestParameters(request.Parameters)
	parameters[semantic.FieldID] = entityID
	parameters[semantic.FieldEntityID] = entityID
	parameters[semantic.FieldTargetID] = entityID
	parameters[semantic.FieldTargetType] = entityType
	parameters[semantic.FieldNodeID] = entityID
	switch entityType {
	case "device":
		parameters[semantic.FieldDeviceID] = entityID
	case "room":
		parameters[semantic.FieldRoomID] = entityID
	case "area":
		parameters[semantic.FieldAreaID] = entityID
	case "group":
		parameters[semantic.FieldGroupID] = entityID
	case "scene":
		parameters[semantic.FieldSceneID] = entityID
	case "automation":
		parameters[semantic.FieldAutomationID] = entityID
	case "gateway":
		parameters[semantic.FieldGatewayID] = entityID
		parameters[semantic.FieldDeviceID] = entityID
	}
	request.Parameters = parameters
	return request
}

func metadataCloudReadonlyResponse(request contract.Request, spec metadataReadonlySpec, result api.MetadataReadonlyResult) contract.Response {
	status := "success"
	traceID := spec.traceID
	if result.Partial {
		status = "partial"
		if traceID == "" || !strings.HasSuffix(traceID, "-partial") {
			traceID = result.Capability + "-partial"
		}
	}
	payload := map[string]any{
		semantic.FieldRegion:      result.Region,
		semantic.FieldCapability:  result.Capability,
		semantic.FieldSource:      "cloud_read",
		semantic.FieldCloudWrites: false,
	}
	if result.HouseID != "" {
		payload[semantic.FieldHouseID] = result.HouseID
	}
	if result.DeviceID != "" && metadataReadonlyUsesDeviceID(result.Capability) {
		payload[semantic.FieldDeviceID] = result.DeviceID
	}
	if result.DeviceID != "" && strings.HasPrefix(result.Capability, "gateway.") {
		payload[semantic.FieldGatewayID] = result.DeviceID
	}
	if result.Data != nil {
		payload[semantic.FieldData] = result.Data
	}
	if len(result.Warnings) > 0 {
		payload[semantic.FieldUnknownEvidence] = result.Warnings
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     spec.message,
		Result:          payload,
		Warnings:        result.Warnings,
		TraceID:         traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  result.APICalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func metadataReadonlyUsesDeviceID(capability string) bool {
	switch {
	case strings.HasPrefix(capability, "device."):
		return true
	case strings.HasPrefix(capability, "panel."):
		return true
	case strings.HasPrefix(capability, "knob."):
		return true
	case strings.HasPrefix(capability, "screen."):
		return true
	case strings.HasPrefix(capability, "upgrade."):
		return true
	case strings.HasPrefix(capability, "node."):
		return true
	default:
		return false
	}
}

func metadataReadonlyProjectionResponse(request contract.Request, spec metadataReadonlySpec, region string, houseID string, evidence []any, metrics map[string]any) contract.Response {
	result := map[string]any{
		semantic.FieldRegion:          region,
		semantic.FieldHouseID:         houseID,
		semantic.FieldCapability:      spec.capability,
		semantic.FieldSource:          spec.source,
		semantic.FieldUnknownEvidence: spec.unknownEvidence,
		semantic.FieldCloudWrites:     false,
	}
	if len(evidence) > 0 {
		result[semantic.FieldEntityEvidence] = evidence
	}
	if len(spec.guidance) > 0 {
		result[semantic.FieldGuidance] = spec.guidance
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          spec.status,
		UserMessage:     spec.message,
		Result:          result,
		Warnings:        spec.unknownEvidence,
		TraceID:         spec.traceID,
		Metrics:         metrics,
	}
}

func (app *app) prepareMetadataLocal(request contract.Request, houseID string, spec metadataReadonlySpec) (contract.Response, error) {
	if requestRunsHouseIndependent(request) {
		houseID = ""
	} else if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     spec.message,
		Result: map[string]any{
			semantic.FieldHouseID:          houseID,
			semantic.FieldCapability:       spec.capability,
			semantic.FieldPlanType:         "local_non_persistent_guidance",
			semantic.FieldPersistentWrites: false,
			semantic.FieldCloudWrites:      false,
			semantic.FieldGuidance:         spec.guidance,
			semantic.FieldUnknownEvidence:  spec.unknownEvidence,
		},
		Warnings: spec.unknownEvidence,
		TraceID:  spec.traceID,
		Metrics:  noAPIMetrics(),
	}, nil
}

type metadataReadonlySpec struct {
	capability      string
	status          string
	message         string
	traceID         string
	source          string
	entityTypes     []string
	limit           int
	unknownEvidence []string
	guidance        []string
}

func homeMemberListSpec() metadataReadonlySpec {
	return metadataPartialSpec("home.member.list", "home-member-list-partial", "已读取家庭成员的脱敏只读信息。", "house_context_missing", nil)
}

func metadataDetailReadonlySpec(capability string, message string) metadataReadonlySpec {
	return metadataReadonlySpec{
		capability:      capability,
		status:          "success",
		message:         message,
		traceID:         strings.ReplaceAll(capability, ".", "-") + "-readonly",
		source:          "cloud_read",
		unknownEvidence: []string{},
	}
}

func panelGetSpec() metadataReadonlySpec {
	return metadataPartialSpec("panel.get", "panel-get-partial", "已读取面板详情和按键配置的安全摘要。", "device_context_missing", []string{"device"})
}

func knobGetSpec() metadataReadonlySpec {
	return metadataPartialSpec("knob.get", "knob-get-partial", "已读取旋钮配置的安全摘要。", "device_context_missing", []string{"device"})
}

func deviceStorageGetSpec() metadataReadonlySpec {
	return metadataPartialSpec("device.storage.get", "device-storage-get-partial", "已返回设备素材/存储相关的保守只读证据，专项读取能力尚未启用。", "device_storage_read_unavailable", []string{"device"})
}

func aiVoiceListSpec() metadataReadonlySpec {
	spec := metadataPartialSpec("ai_voice.list", "ai-voice-list-partial", "AI 语音账号能力涉及第三方凭据，当前仅返回安全边界说明。", "ai_voice_credential_policy_requires_review", nil)
	spec.guidance = []string{"不要在对话中粘贴第三方语音账号、密码或 token。", "需要绑定或修改时使用官方 App 或后续本地批准流程。"}
	return spec
}

func automationCapabilitiesSpec() metadataReadonlySpec {
	return metadataReadonlySpec{
		capability:      "automation.capabilities",
		status:          "success",
		message:         "已返回当前 Runtime 的自动化能力边界。",
		traceID:         "automation-capabilities-local",
		source:          "runtime_policy",
		unknownEvidence: []string{},
		guidance: []string{
			"automation.create 会先生成执行预览，直接执行时 Runtime 会重新校验后创建并按名称读回验证。",
			"automation.explain 和 diagnose.automation 可做只读解释和诊断。",
			"automation.update/enable/disable/delete 仍需 owner review 或本地批准。",
		},
	}
}

func favoriteListSpec() metadataReadonlySpec {
	return metadataPartialSpec("favorite.list", "favorite-list-partial", "已读取家庭收藏配置的只读摘要。", "house_context_missing", nil)
}

func favoritePlanSpec() metadataReadonlySpec {
	return metadataReadonlySpec{
		capability:      "favorite.plan",
		status:          "success",
		message:         "已生成本地收藏整理建议；不会修改云端首页或收藏配置。",
		traceID:         "favorite-plan-local",
		source:          "local_guidance_only",
		unknownEvidence: []string{"favorite_current_order_unavailable"},
		guidance:        []string{"先读取当前房间、设备、情景和自动化实体。", "按高频设备、关键情景、房间维度整理收藏候选。", "真实收藏写入仍需后续执行预览与用户确认。"},
	}
}

func homeSortListSpec() metadataReadonlySpec {
	return metadataReadonlySpec{
		capability:      "home.sort.list",
		status:          "success",
		message:         "已读取家庭排序配置的只读摘要。",
		traceID:         "home-sort-list-readonly",
		source:          "cloud_read",
		unknownEvidence: []string{},
	}
}

func metadataPartialSpec(capability string, traceID string, message string, unknown string, entityTypes []string) metadataReadonlySpec {
	source := "entity_projection_and_policy"
	if len(entityTypes) == 0 {
		source = "local_policy_only"
	}
	return metadataReadonlySpec{
		capability:      capability,
		status:          "partial",
		message:         message,
		traceID:         traceID,
		source:          source,
		entityTypes:     entityTypes,
		limit:           8,
		unknownEvidence: []string{unknown},
	}
}

func metadataEntityEvidence(entities []api.EntitySummary, entityTypes []string, limit int) []any {
	if limit <= 0 {
		limit = 8
	}
	allowed := map[string]bool{}
	for _, entityType := range entityTypes {
		allowed[entityType] = true
	}
	items := []any{}
	for _, entity := range entities {
		if !allowed[entity.Type] {
			continue
		}
		items = append(items, entitySummaryMap(entity))
		if len(items) >= limit {
			break
		}
	}
	return items
}

func unsupportedMetadataIntentResponse(request contract.Request) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "not_supported",
		UserMessage:     fmt.Sprintf("当前 Runtime 尚未支持 %s。", request.Intent),
		Warnings:        []string{"metadata_intent_not_supported"},
		TraceID:         "metadata-intent-not-supported",
		Metrics:         noAPIMetrics(),
	}
}
