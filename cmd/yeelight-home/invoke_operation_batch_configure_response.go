package main

import (
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func operationBatchStepResult(index int, step operationBatchStep, response contract.Response) map[string]any {
	result := map[string]any{
		"index":   index,
		"intent":  step.Intent,
		"status":  response.Status,
		"traceId": response.TraceID,
	}
	if response.Result != nil {
		result["result"] = response.Result
	}
	if len(response.Warnings) > 0 {
		result["warnings"] = response.Warnings
	}
	return result
}

func operationBatchCommitResponse(request contract.Request, record plan.Record, results []any, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已按一次授权提交并验证批量配置计划。",
		Result: map[string]any{
			"region":           record.Region,
			"houseId":          record.HouseID,
			"capability":       "operation.batch.configure",
			"stepCount":        len(results),
			"steps":            results,
			"persistentWrites": true,
		},
		Execution: map[string]any{
			"planId":   record.ID,
			"planHash": record.Hash,
			"intent":   record.Intent,
			"status":   "committed",
		},
		Warnings: []string{},
		TraceID:  "operation-batch-configure-commit",
		Metrics: map[string]any{
			"apiCalls":  apiCalls,
			"cacheHits": 0,
		},
	}
}

func operationBatchPartialResponse(request contract.Request, record plan.Record, completed []any, failed operationBatchStep, response contract.Response, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "partial",
		UserMessage:     "批量配置计划已部分执行；其中一个步骤未成功，后续步骤未继续执行。",
		Result: map[string]any{
			"region":         record.Region,
			"houseId":        record.HouseID,
			"capability":     "operation.batch.configure",
			"completedSteps": completed,
			"failedStep": map[string]any{
				"index":         failed.StepNumber,
				"intent":        failed.Intent,
				"status":        response.Status,
				"userMessage":   response.UserMessage,
				"clarification": response.Clarification,
				"error":         response.Error,
			},
			"persistentWrites": true,
		},
		Execution: map[string]any{
			"planId":   record.ID,
			"planHash": record.Hash,
			"intent":   record.Intent,
			"status":   "partial",
		},
		Warnings: []string{"operation_batch_partial_commit"},
		TraceID:  "operation-batch-configure-partial",
		Metrics: map[string]any{
			"apiCalls":  apiCalls + responseMetricInt(response, "apiCalls"),
			"cacheHits": 0,
		},
	}
}

func responseMetricInt(response contract.Response, key string) int {
	if response.Metrics == nil {
		return 0
	}
	return anyInt(response.Metrics[key])
}

func anyInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
