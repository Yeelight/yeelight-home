package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

const entityRenameBatchLimit = 20

func (app *app) prepareEntityBatchRename(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{
			semantic.ParameterPath(semantic.FieldHouseID),
			semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
			"local profile houseId",
		}), nil
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
	items, reason := normalizeEntityBatchRenameItems(request, entities)
	if reason != "" {
		return entityBatchRenameClarificationResponse(request, reason), nil
	}
	payload := map[string]any{
		semantic.FieldHouseID: requestNumberOrString(houseID),
		semantic.FieldItems:   items,
	}
	preview := entityBatchRenamePreview(items, entities)
	now := time.Now()
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, fmt.Sprintf("批量重命名 %d 个设备或情景", len(items)), payload, []string{
		"提交前重新读取家庭实体列表",
		"每个目标必须是当前家庭下的设备或情景",
		"单次计划最多重命名 20 个目标",
		"提交后通过 entity.list 验证每个目标名称",
	}, now)
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, 0), nil
}

func normalizeEntityBatchRenameItems(request contract.Request, entities api.EntityListResult) ([]any, string) {
	rawItems, ok := requestMapList(request.Parameters[semantic.FieldItems])
	if !ok || len(rawItems) == 0 || len(rawItems) > entityRenameBatchLimit {
		return nil, "invalid_entity_rename_batch_payload"
	}
	result := make([]any, 0, len(rawItems))
	seen := map[string]bool{}
	for _, raw := range rawItems {
		entityType := firstRequestString(raw, semantic.FieldEntityType)
		typeID, typeReason := entityRenameTypeID(raw, entityType)
		if typeReason != "" {
			return nil, typeReason
		}
		entityType, _ = entityTypeForRenameType(typeID)
		id := firstRequestString(raw, semantic.FieldID, semantic.FieldEntityID)
		currentName := firstRequestString(raw, semantic.FieldCurrentName)
		if id == "" && currentName != "" {
			resolved, reason := resolveEntityRenameTargetByName(entities, entityType, currentName)
			if reason != "" {
				return nil, reason
			}
			id = resolved.ID
		}
		if id == "" {
			return nil, "missing_entity_rename_target"
		}
		current, ok := findEntitySummary(entities, entityType, id)
		if !ok {
			return nil, "invalid_entity_rename_reference"
		}
		newName := firstRequestString(raw, semantic.FieldNewName, semantic.FieldName)
		if newName == "" || newName == current.Name {
			return nil, "invalid_entity_rename_name"
		}
		for _, entity := range entities.Entities {
			if entity.Type == entityType && entity.ID != id && entity.Name == newName {
				return nil, "entity_rename_name_already_exists"
			}
		}
		key := fmt.Sprintf("%s:%s", entityType, id)
		if seen[key] {
			return nil, "duplicate_entity_rename_target"
		}
		seen[key] = true
		item := map[string]any{
			semantic.FieldID:       id,
			semantic.FieldEntityID: id,
			semantic.InternalField(semantic.DomainAction, semantic.FieldTargetType): typeID,
			semantic.FieldEntityType: entityType,
			semantic.FieldName:       newName,
		}
		if index, ok := requestInt(raw[semantic.FieldIndex]); ok {
			item[semantic.FieldIndex] = index
		}
		result = append(result, item)
	}
	return result, ""
}

func entityRenameTypeID(item map[string]any, entityType string) (int, string) {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case "device":
		return 2, ""
	case "scene":
		return 6, ""
	default:
		return 0, "invalid_entity_rename_resource_type"
	}
}

func entityTypeForRenameType(typeID int) (string, bool) {
	switch typeID {
	case groupTypeDevice:
		return "device", true
	case groupTypeScene:
		return "scene", true
	default:
		return "", false
	}
}

func resolveEntityRenameTargetByName(entities api.EntityListResult, entityType string, name string) (api.EntitySummary, string) {
	match, candidates, _ := findEntity(entityGetTarget{name: name, entityType: entityType}, entities.Entities)
	if match.ID != "" {
		return match, ""
	}
	if len(candidates) > 0 {
		return api.EntitySummary{}, "ambiguous_entity_rename_target"
	}
	return api.EntitySummary{}, "entity_rename_target_not_found"
}

func entityBatchRenamePreview(items []any, entities api.EntityListResult) map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entityType := requestString(item[semantic.FieldEntityType])
		entityID := requestString(item[semantic.FieldID])
		row := map[string]any{
			semantic.FieldEntityType: entityType,
			semantic.FieldID:         entityID,
			semantic.FieldName:       item[semantic.FieldName],
		}
		if current, ok := findEntitySummary(entities, entityType, entityID); ok {
			row[semantic.FieldCurrentName] = current.Name
			row[semantic.FieldRoomID] = current.RoomID
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(left, right int) bool {
		if rows[left][semantic.FieldEntityType] == rows[right][semantic.FieldEntityType] {
			return requestString(rows[left][semantic.FieldID]) < requestString(rows[right][semantic.FieldID])
		}
		return requestString(rows[left][semantic.FieldEntityType]) < requestString(rows[right][semantic.FieldEntityType])
	})
	return map[string]any{
		semantic.FieldItemCount: len(rows),
		semantic.FieldItems:     rows,
	}
}

func entityBatchRenameAcceptedFields() []string {
	return []string{
		semantic.ParameterPath(semantic.FieldHouseID),
		semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityType),
		semanticParameterArrayPath(semantic.FieldItems, semantic.FieldID),
		semanticParameterArrayPath(semantic.FieldItems, semantic.FieldEntityID),
		semanticParameterArrayPath(semantic.FieldItems, semantic.FieldCurrentName),
		semanticParameterArrayPath(semantic.FieldItems, semantic.FieldNewName),
	}
}

func entityBatchRenameClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, entityBatchRenameAcceptedFields(), entityRenameBatchPayloadGuide())
}

func (app *app) executeEntityBatchRename(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewEntityBatchRenameClient(endpoint, nil).Run(ctx, api.EntityBatchRenameRequest{
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.SpaceOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return entityBatchRenameExecuteResponse(request, record, result), nil
}
