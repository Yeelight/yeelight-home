package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

type nodePropertyTarget struct {
	entityType string
	nodeID     string
	name       string
	roomID     string
	roomName   string
}

func (app *app) invokeNodePropertySet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	propertyID := devicePropertySetPropertyName(request)
	if propertyID == "" {
		return nodePropertySetClarificationResponse(request, "missing_property", target, nil, 0), nil
	}
	if semantic.PropertySensitive(propertyID) {
		return devicePropertySetSensitivePropertyResponse(request, propertyID), nil
	}
	value, ok := request.Parameters[semantic.FieldValue]
	if !ok {
		return nodePropertySetClarificationResponse(request, "missing_value", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if direct, ok := directNodePropertyTarget(request, houseID, target); ok {
		execution, err := runNodePropertySet(ctx, endpoint, direct, houseID, propertyID, value, request, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID, Warnings: []string{}}
		return nodePropertySetResponse(request, entities, entitySummaryFromNodeTarget(direct, houseID), execution, value, 0, semantic.PropertyName, "已设置 %s 的%s。", "node-property-set-command"), nil
	}
	if target.id == "" && target.name == "" {
		return nodePropertySetClarificationResponse(request, "missing_target", target, nil, 0), nil
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return contract.Response{}, err
	}
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	if match.ID == "" {
		return nodePropertySetClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
	}
	if len(candidates) > 1 && target.id == "" {
		return nodePropertySetClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
	}
	if !nodePropertySetEntityTypeSupported(match.Type) {
		return nodePropertySetClarificationResponse(request, "target_not_supported_node", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
	}
	execution, err := runNodePropertySet(ctx, endpoint, nodePropertyTarget{
		entityType: match.Type,
		nodeID:     match.ID,
		name:       match.Name,
		roomID:     match.RoomID,
		roomName:   target.roomName,
	}, houseID, propertyID, value, request, authorization, clientID)
	if err != nil {
		return contract.Response{}, err
	}
	return nodePropertySetResponse(request, entities, match, execution, value, entityListAPICalls(entities), semantic.PropertyName, "已设置 %s 的%s。", "node-property-set-command"), nil
}

func (app *app) invokeNodePropertyToggle(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	propertyID := devicePropertySetPropertyName(request)
	if propertyID == "" {
		return nodePropertySetClarificationResponse(request, "missing_property", target, nil, 0), nil
	}
	if semantic.PropertySensitive(propertyID) {
		return devicePropertySetSensitivePropertyResponse(request, propertyID), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	nodeTarget, entities, entity, resolutionAPICalls, handled, err := app.resolveNodeControlTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil || handled {
		return entities.Response, err
	}
	execution, err := api.NewNodeControlClient(endpoint, nil).RunToggle(ctx, api.NodeControlRequest{
		HouseID:      houseID,
		NodeType:     nodeTarget.entityType,
		NodeID:       nodeTarget.nodeID,
		PropertyName: propertyID,
		Payload:      controlPayloadFromRequest(request),
		Duration:     request.Parameters[semantic.FieldDuration],
		Delay:        request.Parameters[semantic.FieldDelay],
		Credentials: api.NodeControlCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return nodeControlResponse(request, entities.Entities, entity, execution, resolutionAPICalls, "已切换 %s 的%s。", "node-property-toggle-command"), nil
}

func (app *app) invokeNodeActionExecute(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	actionName := firstRequestString(request.Parameters, semantic.FieldActionName, semantic.FieldAction, semantic.FieldName)
	if actionName == "" {
		return nodeActionClarificationResponse(request, "missing_action_name", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	nodeTarget, entities, entity, resolutionAPICalls, handled, err := app.resolveNodeControlTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil || handled {
		return entities.Response, err
	}
	execution, err := api.NewNodeControlClient(endpoint, nil).RunAction(ctx, api.NodeControlRequest{
		HouseID:    houseID,
		NodeType:   nodeTarget.entityType,
		NodeID:     nodeTarget.nodeID,
		ActionName: actionName,
		Payload:    controlPayloadFromRequest(request),
		Duration:   request.Parameters[semantic.FieldDuration],
		Delay:      request.Parameters[semantic.FieldDelay],
		Credentials: api.NodeControlCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return nodeControlResponse(request, entities.Entities, entity, execution, resolutionAPICalls, "已执行 %s 的动作。", "node-action-execute-command"), nil
}

func (app *app) invokeLightingFlowExecute(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	if firstNonNil(request.Parameters[semantic.FieldFlow], request.Parameters[semantic.FieldPayload]) == nil {
		return nodeActionClarificationResponse(request, "missing_flow_payload", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	nodeTarget, entities, entity, resolutionAPICalls, handled, err := app.resolveNodeControlTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil || handled {
		return entities.Response, err
	}
	execution, err := api.NewNodeControlClient(endpoint, nil).RunFlow(ctx, api.NodeControlRequest{
		HouseID:  houseID,
		NodeType: nodeTarget.entityType,
		NodeID:   nodeTarget.nodeID,
		Flow:     request.Parameters[semantic.FieldFlow],
		Payload:  controlPayloadFromRequest(request),
		Duration: request.Parameters[semantic.FieldDuration],
		Delay:    request.Parameters[semantic.FieldDelay],
		Credentials: api.NodeControlCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return nodeControlResponse(request, entities.Entities, entity, execution, resolutionAPICalls, "已应用 %s 的高级灯效。", "lighting-flow-execute-command"), nil
}

func (app *app) invokeNodePropertiesSet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	target := entityGetTargetFromRequest(request)
	properties := nodePropertiesFromRequest(request)
	if len(properties) == 0 {
		return nodePropertySetClarificationResponse(request, "missing_properties", target, nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	nodeTarget, entities, entity, resolutionAPICalls, handled, err := app.resolveNodeControlTarget(ctx, request, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil || handled {
		return entities.Response, err
	}
	execution, err := api.NewNodeControlClient(endpoint, nil).RunPropertiesSet(ctx, api.NodePropertiesSetRequest{
		HouseID:    houseID,
		NodeType:   nodeTarget.entityType,
		NodeID:     nodeTarget.nodeID,
		Properties: properties,
		Credentials: api.NodeControlCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return nodeControlResponse(request, entities.Entities, entity, execution, resolutionAPICalls, "已更新 %s 的多个状态。", "node-properties-set-command"), nil
}

func (app *app) invokeNodePropertyBatchSet(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	propertyID := devicePropertySetPropertyName(request)
	if propertyID == "" {
		return nodePropertySetClarificationResponse(request, "missing_property", entityGetTargetFromRequest(request), nil, 0), nil
	}
	if semantic.PropertySensitive(propertyID) {
		return devicePropertySetSensitivePropertyResponse(request, propertyID), nil
	}
	value, ok := request.Parameters[semantic.FieldValue]
	if !ok {
		return nodePropertySetClarificationResponse(request, "missing_value", entityGetTargetFromRequest(request), nil, 0), nil
	}
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	nodeType := api.NormalizeNodeType(firstRequestString(request.Parameters, semantic.FieldNodeType, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldType))
	nodeIDs := nodeBatchIDsFromRequest(request, nodeType, houseID)
	if nodeType == "" || len(nodeIDs) == 0 {
		return nodePropertySetClarificationResponse(request, "missing_batch_target", entityGetTargetFromRequest(request), nil, 0), nil
	}
	execution, err := api.NewNodeControlClient(endpoint, nil).RunPropertyBatchSet(ctx, api.NodePropertyBatchSetRequest{
		HouseID:      houseID,
		NodeType:     nodeType,
		NodeIDs:      nodeIDs,
		PropertyName: propertyID,
		Value:        value,
		Credentials: api.NodeControlCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	entity := api.EntitySummary{Type: nodeType, ID: strings.Join(nodeIDs, ","), Name: fmt.Sprintf("%d 个对象", len(nodeIDs)), HouseID: houseID}
	entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID, Warnings: []string{}}
	return nodeControlResponse(request, entities, entity, execution, 0, "已批量更新 %s 的%s。", "node-property-batch-set-command"), nil
}

func (app *app) invokeStateBatchQuery(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	items := stateBatchItemsFromRequest(request, houseID)
	if len(items) == 0 {
		return stateBatchQueryClarificationResponse(request, "missing_batch_targets"), nil
	}
	results := make([]any, 0, len(items))
	warnings := []string{}
	apiCalls := 0
	failures := 0
	stateClient := api.NewStateQueryClient(endpoint, nil)
	for _, item := range items {
		state, err := stateClient.Run(ctx, api.StateQueryRequest{
			HouseID:      houseID,
			NodeType:     item.nodeType,
			NodeID:       item.nodeID,
			DeviceID:     item.deviceID,
			PropertyName: item.propertyName,
			PropertySet:  item.propertySet,
			Credentials: api.StateQueryCredentials{
				Authorization: authorization,
				ClientID:      clientID,
			},
		})
		if err != nil {
			failures++
			results = append(results, map[string]any{
				semantic.FieldNodeType: item.nodeType,
				semantic.FieldNodeID:   item.nodeID,
				semantic.FieldDeviceID: item.deviceID,
				semantic.FieldProperty: semantic.PropertyName(item.propertyName),
				semantic.FieldError:    "state_query_failed",
			})
			continue
		}
		apiCalls += stateQueryAPICalls(state)
		results = append(results, stateBatchResult(item, state))
	}
	status := "success"
	message := "已读取批量状态。"
	traceID := "state-batch-query-readonly"
	if failures > 0 {
		status = "partial"
		message = "已读取部分批量状态。"
		traceID = "state-batch-query-partial"
		warnings = append(warnings, "state_batch_partial")
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     message,
		Result: map[string]any{
			semantic.FieldRegion:  endpoint.Region,
			semantic.FieldHouseID: houseID,
			semantic.FieldResults: results,
			semantic.FieldCount:   len(results),
		},
		Warnings: warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}, nil
}

type resolvedNodeControlTarget struct {
	Entities api.EntityListResult
	Response contract.Response
}

func (app *app) resolveNodeControlTarget(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string, target entityGetTarget) (nodePropertyTarget, resolvedNodeControlTarget, api.EntitySummary, int, bool, error) {
	if direct, ok := directNodePropertyTarget(request, houseID, target); ok {
		entities := api.EntityListResult{Region: endpoint.Region, HouseID: houseID, Warnings: []string{}}
		return direct, resolvedNodeControlTarget{Entities: entities}, entitySummaryFromNodeTarget(direct, houseID), 0, false, nil
	}
	if target.id == "" && target.name == "" {
		return nodePropertyTarget{}, resolvedNodeControlTarget{Response: nodePropertySetClarificationResponse(request, "missing_target", target, nil, 0)}, api.EntitySummary{}, 0, true, nil
	}
	resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, authorization, clientID, target)
	if err != nil {
		return nodePropertyTarget{}, resolvedNodeControlTarget{}, api.EntitySummary{}, 0, false, err
	}
	entities := resolved.Entities
	match := resolved.Match
	candidates := resolved.Candidates
	apiCalls := entityListAPICalls(entities)
	if match.ID == "" {
		return nodePropertyTarget{}, resolvedNodeControlTarget{Entities: entities, Response: nodePropertySetClarificationResponse(request, "entity_not_found", target, candidates, apiCalls)}, api.EntitySummary{}, apiCalls, true, nil
	}
	if len(candidates) > 1 && target.id == "" {
		return nodePropertyTarget{}, resolvedNodeControlTarget{Entities: entities, Response: nodePropertySetClarificationResponse(request, "ambiguous_target", target, candidates, apiCalls)}, api.EntitySummary{}, apiCalls, true, nil
	}
	if !nodePropertySetEntityTypeSupported(match.Type) {
		return nodePropertyTarget{}, resolvedNodeControlTarget{Entities: entities, Response: nodePropertySetClarificationResponse(request, "target_not_supported_node", target, []api.EntitySummary{match}, apiCalls)}, api.EntitySummary{}, apiCalls, true, nil
	}
	return nodePropertyTarget{
		entityType: match.Type,
		nodeID:     match.ID,
		name:       match.Name,
		roomID:     match.RoomID,
		roomName:   target.roomName,
	}, resolvedNodeControlTarget{Entities: entities}, match, apiCalls, false, nil
}

func directNodePropertyTarget(request contract.Request, houseID string, target entityGetTarget) (nodePropertyTarget, bool) {
	nodeType := firstNodePropertyString(request.Parameters, semantic.FieldNodeType, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldType)
	if nodeType == "" {
		nodeType = target.entityType
	}
	nodeType = api.NormalizeNodeType(nodeType)
	nodeID := firstNodePropertyString(request.Parameters, nodeIDKeysForType(nodeType)...)
	nodeID = firstNonEmptyString(nodeID, target.id)
	name := firstNodePropertyString(request.Parameters, nodeNameKeysForType(nodeType)...)
	name = firstNonEmptyString(name, target.name)
	if nodeType == "home" {
		nodeID = firstNonEmptyString(nodeID, houseID)
		name = firstNonEmptyString(name, "全屋")
	}
	if nodeType == "" || nodeID == "" {
		return nodePropertyTarget{}, false
	}
	if _, ok := api.NodeTypeID(nodeType); !ok {
		return nodePropertyTarget{}, false
	}
	return nodePropertyTarget{
		entityType: nodeType,
		nodeID:     nodeID,
		name:       name,
		roomID:     target.roomID,
		roomName:   target.roomName,
	}, true
}

func controlPayloadFromRequest(request contract.Request) map[string]any {
	if payload := requestMap(request.Parameters[semantic.FieldPayload]); payload != nil {
		return payload
	}
	if payload := requestMap(request.Parameters[semantic.FieldParameters]); payload != nil {
		return payload
	}
	return map[string]any{}
}

func nodePropertiesFromRequest(request contract.Request) map[string]any {
	if properties := requestMap(request.Parameters[semantic.FieldProperties]); properties != nil {
		return properties
	}
	if set := requestMap(request.Parameters[semantic.FieldSet]); set != nil {
		return set
	}
	propertyID := devicePropertySetPropertyName(request)
	value, ok := request.Parameters[semantic.FieldValue]
	if propertyID == "" || !ok || semantic.PropertySensitive(propertyID) {
		return map[string]any{}
	}
	return map[string]any{propertyID: value}
}

func nodeBatchIDsFromRequest(request contract.Request, nodeType string, houseID string) []string {
	if nodeType == "home" {
		return requestStringSetValues(firstNonEmptyString(firstRequestString(request.Parameters, semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldHouseID, semantic.FieldID), houseID))
	}
	keys := append(nodeIDKeysForType(nodeType), semantic.FieldNodeIDs, semantic.FieldIDs)
	values := make([]any, 0, len(keys)+3)
	for _, key := range keys {
		values = append(values, request.Parameters[key])
	}
	values = append(values, request.Parameters[semantic.FieldDeviceIDs], request.Parameters[semantic.FieldRoomIDs], request.Parameters[semantic.FieldAreaIDs], request.Parameters[semantic.FieldGroupIDs])
	return requestStringSetValues(values...)
}

func requestStringSetValues(values ...any) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		for _, item := range requestStringList(value) {
			for _, part := range strings.Split(item, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed == "" || seen[trimmed] {
					continue
				}
				seen[trimmed] = true
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func runNodePropertySet(ctx context.Context, endpoint api.Endpoint, target nodePropertyTarget, houseID string, propertyID string, value any, request contract.Request, authorization string, clientID string) (api.NodePropertySetResult, error) {
	return api.NewNodePropertySetClient(endpoint, nil).Run(ctx, api.NodePropertySetRequest{
		HouseID:      houseID,
		NodeType:     target.entityType,
		NodeID:       target.nodeID,
		PropertyName: propertyID,
		Value:        value,
		Command:      firstNodePropertyString(request.Parameters, semantic.FieldCommand),
		Duration:     request.Parameters[semantic.FieldDuration],
		Delay:        request.Parameters[semantic.FieldDelay],
		Index:        request.Parameters[semantic.FieldIndex],
		Category:     request.Parameters[semantic.FieldCategory],
		Credentials: api.NodePropertySetCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
}

func runNodePropertyAdjust(ctx context.Context, endpoint api.Endpoint, target nodePropertyTarget, houseID string, propertyID string, value any, authorization string, clientID string) (api.NodePropertyAdjustResult, error) {
	return api.NewNodePropertyAdjustClient(endpoint, nil).Run(ctx, api.NodePropertyAdjustRequest{
		HouseID:      houseID,
		NodeType:     target.entityType,
		NodeID:       target.nodeID,
		PropertyName: propertyID,
		Value:        value,
		Credentials: api.NodePropertyAdjustCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
}

func nodeControlResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.NodeControlResult, resolutionAPICalls int, messageTemplate string, traceID string) contract.Response {
	entityName := strings.TrimSpace(entity.Name)
	if entityName == "" {
		entityName = firstNonEmptyString(execution.NodeID, entity.ID)
	}
	result := map[string]any{
		semantic.FieldRegion:   firstNonEmptyString(entities.Region, execution.Region),
		semantic.FieldHouseID:  firstNonEmptyString(entities.HouseID, execution.HouseID),
		semantic.FieldEntity:   entitySummaryMap(entity),
		semantic.FieldNodeType: execution.NodeType,
		semantic.FieldNodeID:   execution.NodeID,
		semantic.FieldCommand:  execution.Command,
		semantic.FieldSource:   execution.Source,
		semantic.FieldRawShape: execution.RawShape,
	}
	if execution.PropertyName != "" {
		result[semantic.FieldProperty] = semantic.PropertyName(execution.PropertyName)
	}
	if execution.ActionName != "" {
		result[semantic.FieldActionName] = execution.ActionName
	}
	if len(execution.PropertySet) > 0 {
		result[semantic.FieldProperties] = execution.PropertySet
	}
	if execution.TargetCount > 0 {
		result[semantic.FieldCount] = execution.TargetCount
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     nodePropertySetUserMessage(messageTemplate, entityName, semantic.PropertyName(execution.PropertyName)),
		Result:          result,
		Warnings:        entities.Warnings,
		TraceID:         traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  resolutionAPICalls + nodeControlAPICalls(execution),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func nodeControlAPICalls(execution api.NodeControlResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}

func nodePropertySetResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.NodePropertySetResult, expected any, resolutionAPICalls int, propertyName func(string) string, messageTemplate string, traceID string) contract.Response {
	entityName := strings.TrimSpace(entity.Name)
	if entityName == "" {
		entityName = execution.NodeID
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     nodePropertySetUserMessage(messageTemplate, entityName, propertyName(execution.PropertyName)),
		Result: map[string]any{
			semantic.FieldRegion:        firstNonEmptyString(entities.Region, execution.Region),
			semantic.FieldHouseID:       firstNonEmptyString(entities.HouseID, execution.HouseID),
			semantic.FieldEntity:        entitySummaryMap(entity),
			semantic.FieldNodeType:      execution.NodeType,
			semantic.FieldNodeID:        execution.NodeID,
			semantic.FieldProperty:      propertyName(execution.PropertyName),
			semantic.FieldCommand:       execution.Command,
			semantic.FieldSource:        execution.Source,
			semantic.FieldExpectedValue: expected,
			semantic.FieldVerified:      false,
		},
		Warnings: entities.Warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  resolutionAPICalls + nodePropertySetAPICalls(execution),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func nodePropertyAdjustResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.NodePropertyAdjustResult, delta int, resolutionAPICalls int, propertyName func(string) string, messageTemplate string, traceID string) contract.Response {
	entityName := strings.TrimSpace(entity.Name)
	if entityName == "" {
		entityName = execution.NodeID
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     nodePropertySetUserMessage(messageTemplate, entityName, propertyName(execution.PropertyName)),
		Result: map[string]any{
			semantic.FieldRegion:   firstNonEmptyString(entities.Region, execution.Region),
			semantic.FieldHouseID:  firstNonEmptyString(entities.HouseID, execution.HouseID),
			semantic.FieldEntity:   entitySummaryMap(entity),
			semantic.FieldNodeType: execution.NodeType,
			semantic.FieldNodeID:   execution.NodeID,
			semantic.FieldProperty: propertyName(execution.PropertyName),
			semantic.FieldCommand:  execution.Command,
			semantic.FieldSource:   execution.Source,
			semantic.FieldDelta:    delta,
			semantic.FieldVerified: false,
		},
		Warnings: entities.Warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  resolutionAPICalls + nodePropertyAdjustAPICalls(execution),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func nodePropertySetUserMessage(messageTemplate string, entityName string, propertyName string) string {
	if strings.Count(messageTemplate, "%s") >= 2 {
		return fmt.Sprintf(messageTemplate, entityName, propertyName)
	}
	return fmt.Sprintf(messageTemplate, entityName)
}

func nodePropertySetClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
	preview := make([]any, 0, len(candidates))
	for index, candidate := range candidates {
		if index >= 5 {
			break
		}
		preview = append(preview, entitySummaryMap(candidate))
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请明确要控制的对象、属性和值。",
		Clarification: map[string]any{
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"home", "room", "area", "group", "device"},
			semantic.FieldAcceptedFields: []string{
				semantic.ParameterPath(semantic.FieldNodeType),
				semantic.ParameterPath(semantic.FieldNodeID),
				semantic.ParameterPath(semantic.FieldTargetType),
				semantic.ParameterPath(semantic.FieldTargetID),
				semantic.ParameterPath(semantic.FieldEntityType),
				semantic.ParameterPath(semantic.FieldEntityID),
				semantic.ParameterPath(semantic.FieldRoomID),
				semantic.ParameterPath(semantic.FieldRoomName),
				semantic.ParameterPath(semantic.FieldAreaID),
				semantic.ParameterPath(semantic.FieldAreaName),
				semantic.ParameterPath(semantic.FieldGroupID),
				semantic.ParameterPath(semantic.FieldGroupName),
				semantic.ParameterPath(semantic.FieldDeviceID),
				semantic.ParameterPath(semantic.FieldDeviceName),
				semantic.ParameterPath(semantic.FieldProperty),
				semantic.ParameterPath(semantic.FieldValue),
			},
			semantic.FieldPayloadGuide: payloadGuideForIntent(request.Intent),
		},
		Warnings: []string{},
		TraceID:  "node-property-set-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func nodeActionClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
	response := nodePropertySetClarificationResponse(request, reason, target, candidates, apiCalls)
	if response.Clarification != nil {
		response.Clarification[semantic.FieldAcceptedFields] = []string{
			semantic.ParameterPath(semantic.FieldNodeType),
			semantic.ParameterPath(semantic.FieldNodeID),
			semantic.ParameterPath(semantic.FieldTargetType),
			semantic.ParameterPath(semantic.FieldTargetID),
			semantic.ParameterPath(semantic.FieldEntityType),
			semantic.ParameterPath(semantic.FieldEntityID),
			semantic.ParameterPath(semantic.FieldActionName),
			semantic.ParameterPath(semantic.FieldAction),
			semantic.ParameterPath(semantic.FieldPayload),
			semantic.ParameterPath(semantic.FieldFlow),
		}
	}
	return response
}

type stateBatchItem struct {
	nodeType     string
	nodeID       string
	deviceID     string
	propertyName string
	propertySet  []string
}

func stateBatchItemsFromRequest(request contract.Request, houseID string) []stateBatchItem {
	defaultNodeType := api.NormalizeNodeType(firstRequestString(request.Parameters, semantic.FieldNodeType, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldType))
	defaultProperty := stateQueryPropertyName(request)
	defaultPropertySet := requestStringSetValues(request.Parameters[semantic.FieldProperties], request.Parameters["propertySet"])
	if rows, ok := requestMapList(request.Parameters[semantic.FieldItems]); ok {
		items := make([]stateBatchItem, 0, len(rows))
		for _, row := range rows {
			if item, ok := stateBatchItemFromMap(row, houseID, defaultNodeType, defaultProperty, defaultPropertySet); ok {
				items = append(items, item)
			}
		}
		return items
	}
	nodeType := defaultNodeType
	if nodeType == "" {
		nodeType = "device"
	}
	items := []stateBatchItem{}
	for _, id := range nodeBatchIDsFromRequest(request, nodeType, houseID) {
		item := stateBatchItem{nodeType: nodeType, nodeID: id, propertyName: defaultProperty, propertySet: defaultPropertySet}
		if nodeType == "device" {
			item.deviceID = id
		}
		items = append(items, item)
	}
	return items
}

func stateBatchItemFromMap(row map[string]any, houseID string, defaultNodeType string, defaultProperty string, defaultPropertySet []string) (stateBatchItem, bool) {
	nodeType := api.NormalizeNodeType(firstNonEmptyString(firstRequestString(row, semantic.FieldNodeType, semantic.FieldTargetType, semantic.FieldEntityType, semantic.FieldType), defaultNodeType))
	deviceID := firstRequestString(row, semantic.FieldDeviceID)
	nodeID := firstNonEmptyString(firstRequestString(row, semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldID), deviceID)
	if nodeType == "" && deviceID != "" {
		nodeType = "device"
	}
	if nodeType == "home" && nodeID == "" {
		nodeID = houseID
	}
	if nodeType == "" || nodeID == "" {
		return stateBatchItem{}, false
	}
	property := firstNonEmptyString(firstRequestString(row, semantic.FieldProperty, semantic.FieldPropertyName), defaultProperty)
	if id, ok := semantic.PropertyID(property); ok {
		property = id
	}
	propertySet := requestStringSetValues(row[semantic.FieldProperties], row["propertySet"])
	if len(propertySet) == 0 {
		propertySet = defaultPropertySet
	}
	item := stateBatchItem{nodeType: nodeType, nodeID: nodeID, propertyName: property, propertySet: propertySet}
	if nodeType == "device" {
		item.deviceID = firstNonEmptyString(deviceID, nodeID)
	}
	return item, true
}

func stateBatchResult(item stateBatchItem, state api.StateQueryResult) map[string]any {
	result := map[string]any{
		semantic.FieldNodeType:   firstNonEmptyString(state.NodeType, item.nodeType),
		semantic.FieldNodeID:     firstNonEmptyString(state.NodeID, item.nodeID),
		semantic.FieldDeviceID:   firstNonEmptyString(state.DeviceID, item.deviceID),
		semantic.FieldSource:     state.Source,
		semantic.FieldQueryScope: state.QueryScope,
	}
	if state.PropertyName != "" {
		result[semantic.FieldProperty] = semantic.PropertyName(state.PropertyName)
		result[semantic.FieldValue] = state.Value
	} else {
		result[semantic.FieldProperties] = stateQueryPublicProperties(state.Properties)
	}
	if len(state.Skipped) > 0 {
		result[semantic.FieldSkippedProperties] = stateQueryPublicSkippedProperties(state.Skipped)
	}
	return result
}

func stateBatchQueryClarificationResponse(request contract.Request, reason string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请提供要批量读取状态的对象。",
		Clarification: map[string]any{
			semantic.FieldReason: reason,
			semantic.FieldAcceptedFields: []string{
				semantic.ParameterPath(semantic.FieldItems),
				semantic.ParameterPath(semantic.FieldNodeType),
				semantic.ParameterPath(semantic.FieldNodeIDs),
				semantic.ParameterPath(semantic.FieldDeviceIDs),
				semantic.ParameterPath(semantic.FieldProperty),
				semantic.ParameterPath(semantic.FieldProperties),
			},
			semantic.FieldPayloadGuide: payloadGuideForIntent(request.Intent),
		},
		Warnings: []string{},
		TraceID:  "state-batch-query-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
}

func entitySummaryFromNodeTarget(target nodePropertyTarget, houseID string) api.EntitySummary {
	name := strings.TrimSpace(target.name)
	if name == "" {
		name = target.nodeID
	}
	return api.EntitySummary{
		Type:    target.entityType,
		ID:      target.nodeID,
		Name:    name,
		HouseID: houseID,
		RoomID:  target.roomID,
	}
}

func nodePropertySetEntityTypeSupported(entityType string) bool {
	_, ok := api.NodeTypeID(entityType)
	return ok
}

func nodePropertySetAPICalls(execution api.NodePropertySetResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}

func nodePropertyAdjustAPICalls(execution api.NodePropertyAdjustResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}

func firstNodePropertyString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := requestString(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func nodeIDKeysForType(entityType string) []string {
	switch api.NormalizeNodeType(entityType) {
	case "home":
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldHouseID, semantic.FieldEntityID, semantic.FieldID}
	case "room":
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldRoomID, semantic.FieldEntityID, semantic.FieldID}
	case "area":
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldAreaID, semantic.FieldEntityID, semantic.FieldID}
	case "group":
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldGroupID, semantic.FieldMeshGroupID, semantic.FieldEntityID, semantic.FieldID}
	case "device":
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldDeviceID, semantic.FieldGatewayID, semantic.FieldPanelID, semantic.FieldKnobID, semantic.FieldEntityID, semantic.FieldID}
	default:
		return []string{semantic.FieldNodeID, semantic.FieldTargetID, semantic.FieldEntityID, semantic.FieldID}
	}
}

func nodeNameKeysForType(entityType string) []string {
	switch api.NormalizeNodeType(entityType) {
	case "home":
		return []string{semantic.FieldHomeName, semantic.FieldHouseName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	case "room":
		return []string{semantic.FieldRoomName, semantic.FieldTargetRoomName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	case "area":
		return []string{semantic.FieldAreaName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	case "group":
		return []string{semantic.FieldGroupName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	case "device":
		return []string{semantic.FieldDeviceName, semantic.FieldGatewayName, semantic.FieldPanelName, semantic.FieldKnobName, semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	default:
		return []string{semantic.FieldTargetName, semantic.FieldEntityName, semantic.FieldName}
	}
}
