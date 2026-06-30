package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

func configureClarificationResponse(request contract.Request, reason string, acceptedFields []string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, acceptedFields, nil)
}

func configureClarificationResponseWithGuide(request contract.Request, reason string, acceptedFields []string, guide map[string]any) contract.Response {
	clarification := map[string]any{
		"reason":         reason,
		"acceptedFields": acceptedFields,
	}
	for key, value := range guide {
		clarification[key] = value
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请补充要配置的必要信息。",
		Clarification:   clarification,
		Warnings:        []string{},
		TraceID:         "configure-clarification",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
		},
	}
}

func responseWithVerifiedTopology(response contract.Response, entities api.EntityListResult) contract.Response {
	if entities.Total == 0 {
		return response
	}
	if response.Internal == nil {
		response.Internal = map[string]any{}
	}
	response.Internal["verifiedTopology"] = entities
	return response
}

func executionPreviewResponse(request contract.Request, record operation.Prepared, entities api.EntityListResult) contract.Response {
	return executionPreviewResponseWithDetails(request, record, entities, nil, 0)
}

func executionPreviewResponseWithDetails(request contract.Request, record operation.Prepared, entities api.EntityListResult, preview map[string]any, extraAPICalls int) contract.Response {
	payloadPreview := executionPayloadPreview(record)
	if len(preview) > 0 {
		payloadPreview["semanticPreview"] = preview
	}
	previewPayload := map[string]any{
		"risk":           record.Risk,
		"intent":         record.Intent,
		"summary":        record.Summary,
		"executionModel": "ordinary_invoke_executes_directly",
		"preconditions":  record.Preconditions,
		"payloadPreview": payloadPreview,
	}
	if record.Risk == operation.RiskR3 {
		previewPayload["destructive"] = true
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成语义校验。dry-run 只返回预览；普通 invoke 会直接执行。",
		Result: map[string]any{
			"preparedForDirectExecution": true,
			"preview":                    previewPayload,
		},
		Warnings: []string{},
		TraceID:  "invoke-preview",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + extraAPICalls,
			"cacheHits": 0,
		},
	}
}

func executionPayloadPreview(record operation.Prepared) map[string]any {
	preview := map[string]any{}
	if operation.IsAccountScope(record.HouseID) {
		preview["scope"] = "account"
	} else {
		preview["houseId"] = record.HouseID
	}
	for _, key := range []string{"name", "typeId", "resId", "rank", "favoriteId", "type", "target", "roomId", "deviceId", "sceneId", "automationId", "buttonEventId", "index"} {
		if value, ok := record.Payload[key]; ok {
			preview[key] = value
		}
	}
	if items, ok := record.Payload["items"]; ok {
		preview["items"] = previewList(items, 20)
	}
	if buttons, ok := record.Payload["buttons"]; ok {
		preview["buttons"] = previewList(buttons, 20)
	}
	if buttonEvents, ok := record.Payload["buttonEvents"]; ok {
		preview["buttonEvents"] = previewList(buttonEvents, 20)
	}
	if details, ok := record.Payload["details"]; ok {
		preview["details"] = previewList(details, 20)
	}
	if actions, ok := record.Payload["actions"]; ok {
		preview["actions"] = previewList(actions, 20)
	}
	return preview
}

func previewList(value any, limit int) any {
	items, ok := value.([]any)
	if !ok || limit <= 0 || len(items) <= limit {
		return value
	}
	return map[string]any{
		"count": len(items),
		"items": items[:limit],
	}
}

func homeOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeOrganizationResult) contract.Response {
	userMessage := "已提交并验证首页组织配置。"
	warnings := []string{}
	if !result.Verified {
		userMessage = "已提交首页组织配置，但云端排序读接口当前不可用，未完成读后验证。"
		if result.Warning != "" {
			warnings = append(warnings, result.Warning)
		}
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     userMessage,
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"itemCount":  result.ItemCount,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: warnings,
		TraceID:  "home-organization-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func homeCreateAlreadyExistsResponse(request contract.Request, house api.HouseSummary, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("家庭 %s 已存在，无需创建。", house.Name),
		Result: map[string]any{
			"region":     "",
			"houseId":    house.ID,
			"name":       house.Name,
			"created":    false,
			"verified":   true,
			"verifiedBy": firstNonEmptyString(house.Source, "home.summary"),
		},
		Warnings: []string{},
		TraceID:  "home-create-idempotent",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func homeCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeCreateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证家庭创建操作。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"name":       result.Name,
			"created":    result.Created,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "home-create-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func homeMemberExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeMemberResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成家庭成员操作，并通过成员列表回读验证。",
		Result: map[string]any{
			"intent":           record.Intent,
			"risk":             record.Risk,
			"region":           result.Region,
			"houseId":          result.HouseID,
			"capability":       result.Capability,
			"verified":         result.Verified,
			"verifiedBy":       result.VerifiedBy,
			"resultData":       result.Data,
			"persistentWrites": true,
		},
		Warnings: []string{},
		TraceID:  "home-member-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func homeLockExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeLockResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证整屋重置锁定配置。",
		Result: map[string]any{
			"region":      result.Region,
			"houseId":     result.HouseID,
			"capability":  result.Capability,
			"deviceCount": result.DeviceCount,
			"verified":    result.Verified,
			"verifiedBy":  result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "home-lock-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func entityBatchRenameExecuteResponse(request contract.Request, record operation.Prepared, result api.EntityBatchRenameResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量重命名。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"itemCount":  result.ItemCount,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "entity-batch-rename-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func homeSpaceConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.HomeSpaceConfigurationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证家庭空间配置。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"itemCount":  result.ItemCount,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "home-space-configuration-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func lightingDesignApplyExecuteResponse(request contract.Request, record operation.Prepared, entities api.EntityListResult, results []any, apiCalls int) contract.Response {
	allVerified := true
	for _, item := range results {
		row, ok := item.(map[string]any)
		if !ok || row["verified"] != true {
			allVerified = false
			break
		}
	}
	status := "success"
	traceID := "lighting-design-apply-execute"
	warnings := append([]string{}, entities.Warnings...)
	var responseError *contract.Error
	if !allVerified {
		status = "partial"
		traceID = "lighting-design-apply-verification-mismatch"
		warnings = append(warnings, "write_verification_mismatch")
		responseError = &contract.Error{
			Code:    "write_verification_mismatch",
			Message: "one or more lighting design actions did not match expected values after write",
		}
	}
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          status,
		UserMessage:     "已提交受限照明设计应用操作，并完成设备状态验证。",
		Result: map[string]any{
			"region":           entities.Region,
			"houseId":          record.HouseID,
			"capability":       "lighting.design.apply",
			"persistentWrites": true,
			"createdArtifacts": []string{},
			"actionCount":      len(results),
			"results":          results,
			"verified":         allVerified,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: warnings,
		TraceID:  traceID,
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
		Error: responseError,
	}, entities)
}

func automationStatusExecuteResponse(request contract.Request, record operation.Prepared, result api.AutomationStatusResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证自动化状态。",
		Result: map[string]any{
			"region":       result.Region,
			"houseId":      result.HouseID,
			"automationId": result.AutomationID,
			"name":         result.Name,
			"status":       result.Status,
			"statusLabel":  automationStatusLabel(result.Status),
			"capability":   result.Capability,
			"verified":     result.Verified,
			"verifiedBy":   result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "automation-status-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func automationStatusLabel(status string) string {
	switch status {
	case "1":
		return "enabled"
	case "0":
		return "disabled"
	default:
		return "unknown"
	}
}

func automationUpdateExecuteResponse(request contract.Request, record operation.Prepared, result api.AutomationUpdateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证自动化更新。",
		Result: map[string]any{
			"region":       result.Region,
			"houseId":      result.HouseID,
			"automationId": result.AutomationID,
			"name":         result.Name,
			"status":       result.Status,
			"verified":     result.Verified,
			"verifiedBy":   result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "automation-update-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func sceneUpdateExecuteResponse(request contract.Request, record operation.Prepared, result api.SceneUpdateResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证情景更新。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"sceneId":    result.SceneID,
			"name":       result.Name,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "scene-update-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func metadataDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证删除操作。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"entityType": result.EntityType,
			"entityId":   result.EntityID,
			"name":       result.Name,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-delete-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func metadataBatchDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataBatchDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量删除操作。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"entityType": result.EntityType,
			"itemCount":  result.ItemCount,
			"results":    result.Results,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-batch-delete-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func destructiveDeleteExecuteResponse(request contract.Request, record operation.Prepared, result api.DestructiveDeleteResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证高影响删除操作。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"entityType": result.EntityType,
			"entityId":   result.EntityID,
			"name":       result.Name,
			"risk":       record.Risk,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "destructive-delete-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func deviceUnbindExecuteResponse(request contract.Request, record operation.Prepared, result api.DeviceUnbindResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已完成设备解绑，并通过实体列表回读验证。",
		Result: map[string]any{
			"intent":           record.Intent,
			"risk":             record.Risk,
			"region":           result.Region,
			"houseId":          result.HouseID,
			"deviceId":         result.DeviceID,
			"name":             result.Name,
			"clearMac":         result.ClearMac,
			"unbindRelDevices": result.UnbindRelDevices,
			"verified":         result.Verified,
			"verifiedBy":       result.VerifiedBy,
			"persistentWrites": true,
		},
		Warnings: []string{},
		TraceID:  "device-unbind-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func spaceOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.SpaceOrganizationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证空间组织配置。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"entityType": result.EntityType,
			"entityId":   result.EntityID,
			"name":       result.Name,
			"roomId":     result.RoomID,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "space-organization-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func gatewayConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.GatewayConfigurationResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已更新网关配置，并通过网关详情回读验证。",
		Result: map[string]any{
			"intent":           record.Intent,
			"risk":             record.Risk,
			"region":           result.Region,
			"houseId":          result.HouseID,
			"capability":       result.Capability,
			"gatewayId":        result.GatewayID,
			"name":             result.Name,
			"verified":         result.Verified,
			"verifiedBy":       result.VerifiedBy,
			"persistentWrites": true,
		},
		Warnings: []string{},
		TraceID:  "gateway-configuration-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func spaceBatchOrganizationExecuteResponse(request contract.Request, record operation.Prepared, result api.SpaceBatchOrganizationResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证批量空间组织配置。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"capability": result.Capability,
			"itemCount":  result.ItemCount,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "space-batch-organization-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func panelConfigurationExecuteResponse(request contract.Request, record operation.Prepared, result api.PanelConfigurationResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已提交并验证面板/旋钮配置。",
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"deviceId":   result.DeviceID,
			"capability": result.Capability,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "panel-configuration-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}
}

func roomCreateAlreadyExistsResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("房间 %s 已存在，无需创建。", entity.Name),
		Result: map[string]any{
			"region":   entities.Region,
			"houseId":  entities.HouseID,
			"entity":   entitySummaryMap(entity),
			"created":  false,
			"verified": true,
		},
		Warnings: []string{},
		TraceID:  "room-create-idempotent",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities),
			"cacheHits": 0,
		},
	}
}

func metadataCreateAlreadyExistsResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, label string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("%s %s 已存在，无需创建。", label, entity.Name),
		Result: map[string]any{
			"region":   entities.Region,
			"houseId":  entities.HouseID,
			"entity":   entitySummaryMap(entity),
			"created":  false,
			"verified": true,
		},
		Warnings: []string{},
		TraceID:  "metadata-create-idempotent",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities),
			"cacheHits": 0,
		},
	}
}

func roomCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.RoomCreateResult) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已创建并验证房间 %s。", result.Name),
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"roomId":     result.RoomID,
			"name":       result.Name,
			"created":    result.Created,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "room-create-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func metadataCreateExecuteResponse(request contract.Request, record operation.Prepared, result api.MetadataCreateResult, label string) contract.Response {
	return responseWithVerifiedTopology(contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已创建并验证%s %s。", label, result.Name),
		Result: map[string]any{
			"region":     result.Region,
			"houseId":    result.HouseID,
			"entityType": result.EntityType,
			"entityId":   result.EntityID,
			"name":       result.Name,
			"created":    result.Created,
			"verified":   result.Verified,
			"verifiedBy": result.VerifiedBy,
		},
		Execution: map[string]any{
			"intent": record.Intent,
			"status": "executed",
		},
		Warnings: []string{},
		TraceID:  "metadata-create-execute",
		Metrics: map[string]any{
			"apiCalls":  result.APICalls,
			"cacheHits": 0,
		},
	}, result.VerifiedEntities)
}

func executionBlockedResponse(request contract.Request, code string, message string) contract.Response {
	return executionBlockedResponseWithResult(request, code, message, nil)
}

func executionVerifyBlockedResponse(request contract.Request, record operation.Prepared, verifyErr error) contract.Response {
	code, safeNextStep := executionVerifyCode(verifyErr)
	result := map[string]any{
		"intent": record.Intent,
		"recovery": map[string]any{
			"suggestedIntent": record.Intent,
			"safeNextStep":    safeNextStep,
			"canRegenerate":   true,
			"safeToRetry":     false,
		},
	}
	return executionBlockedResponseWithResult(request, code, verifyErr.Error(), result)
}

func executionBlockedResponseWithResult(request contract.Request, code string, message string, extra map[string]any) contract.Response {
	result := map[string]any{}
	for key, value := range extra {
		result[key] = value
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Result:          result,
		Warnings:        []string{code},
		TraceID:         "execution-blocked",
		Metrics: map[string]any{
			"apiCalls":  0,
			"cacheHits": 0,
		},
		Error: &contract.Error{
			Code:    code,
			Message: message,
		},
	}
}
