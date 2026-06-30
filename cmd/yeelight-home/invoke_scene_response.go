package main

import (
	"fmt"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
)

func sceneExecuteResponse(request contract.Request, entities api.EntityListResult, entity api.EntitySummary, execution api.SceneExecuteResult) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已执行情景：%s。", entity.Name),
		Result: map[string]any{
			"region":   entities.Region,
			"houseId":  entities.HouseID,
			"entity":   entitySummaryMap(entity),
			"source":   execution.Source,
			"rawShape": execution.RawShape,
		},
		Warnings: entities.Warnings,
		TraceID:  "scene-execute-command",
		Metrics: map[string]any{
			"apiCalls":  entityListAPICalls(entities) + sceneExecuteAPICalls(execution),
			"cacheHits": topologyCacheHits(entities),
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
		UserMessage:     "请明确要执行的情景。",
		Clarification: map[string]any{
			"reason":               reason,
			"target":               target.toMap(),
			"candidates":           preview,
			"supportedEntityTypes": []string{"scene"},
		},
		Warnings: []string{},
		TraceID:  "scene-execute-clarification",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func sceneExecuteAPICalls(execution api.SceneExecuteResult) int {
	if execution.APICalls > 0 {
		return execution.APICalls
	}
	return 1
}
