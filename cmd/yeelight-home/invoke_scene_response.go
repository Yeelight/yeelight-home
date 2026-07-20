package main

import (
	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/i18n"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func sceneExecuteResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.SceneExecuteResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     i18n.Text(request.Locale, i18n.SceneExecuted, entity.Name),
		Result: map[string]any{
			semantic.FieldRegion:  entities.Region,
			semantic.FieldHouseID: entities.HouseID,
			semantic.FieldEntity:  entitySummaryMap(entity),
			semantic.FieldSource:  execution.Source,
		},
		Warnings: entities.Warnings,
		TraceID:  "scene-execute-command",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(entities) + sceneExecuteAPICalls(execution),
			semantic.FieldCacheHits: topologyCacheHits(entities),
		},
	}
}

func sceneExecuteClarificationResponse(request contract.Request, reason string, target entityGetTarget, candidates []api.EntitySummary, apiCalls int) contract.Response {
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
		UserMessage:     i18n.Text(request.Locale, i18n.SceneClarification),
		Clarification: map[string]any{
			semantic.FieldReason:               reason,
			semantic.FieldTarget:               target.toMap(),
			semantic.FieldCandidates:           preview,
			semantic.FieldSupportedEntityTypes: []string{"scene"},
		},
		Warnings: []string{},
		TraceID:  "scene-execute-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func sceneExecuteBlockedResponse(request contract.Request, result api.EntityListResult, entity api.EntitySummary, code string, message string) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "blocked",
		UserMessage:     message,
		Result: map[string]any{
			semantic.FieldRegion:      result.Region,
			semantic.FieldHouseID:     result.HouseID,
			semantic.FieldEntity:      entitySummaryMap(entity),
			semantic.FieldSafeToRetry: false,
			semantic.FieldNextAction:  "view_scene_detail_or_add_valid_gateway",
		},
		Warnings: append([]string{}, result.Warnings...),
		TraceID:  "scene-execute-blocked",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  entityListAPICalls(result) + 1,
			semantic.FieldCacheHits: topologyCacheHits(result),
		},
		Error: &contract.Error{
			Code:    code,
			Message: message,
		},
	}
}

func sceneExecuteAPICalls(execution api.SceneExecuteResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}
