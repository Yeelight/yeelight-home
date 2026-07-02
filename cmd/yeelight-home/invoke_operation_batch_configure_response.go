package main

import (
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func operationBatchStepResult(index int, step operationBatchStep, response contract.Response) map[string]any {
	result := map[string]any{
		semantic.FieldIndex:   index,
		semantic.FieldIntent:  step.Intent,
		semantic.FieldStatus:  response.Status,
		semantic.FieldTraceID: response.TraceID,
	}
	if response.Result != nil {
		result[semantic.FieldResult] = response.Result
	}
	if len(response.Warnings) > 0 {
		result[semantic.FieldWarnings] = response.Warnings
	}
	return result
}

func operationBatchExecuteResponse(request contract.Request, record operation.Prepared, results []any, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     "已直接执行并验证批量配置。",
		Result: map[string]any{
			semantic.FieldRegion:           record.Region,
			semantic.FieldHouseID:          record.HouseID,
			semantic.FieldCapability:       "operation.batch.configure",
			semantic.FieldStepCount:        len(results),
			semantic.FieldSteps:            results,
			semantic.FieldPersistentWrites: true,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "executed",
		},
		Warnings: []string{},
		TraceID:  "operation-batch-configure-execute",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls,
			semantic.FieldCacheHits: 0,
		},
	}
}

func operationBatchPartialResponse(request contract.Request, record operation.Prepared, completed []any, failed operationBatchStep, response contract.Response, apiCalls int) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "partial",
		UserMessage:     "批量配置已部分执行；其中一个步骤未成功，后续步骤未继续执行。",
		Result: map[string]any{
			semantic.FieldRegion:         record.Region,
			semantic.FieldHouseID:        record.HouseID,
			semantic.FieldCapability:     "operation.batch.configure",
			semantic.FieldCompletedSteps: completed,
			semantic.FieldFailedStep: map[string]any{
				semantic.FieldIndex:         failed.StepNumber,
				semantic.FieldIntent:        failed.Intent,
				semantic.FieldStatus:        response.Status,
				semantic.FieldUserMessage:   response.UserMessage,
				semantic.FieldClarification: response.Clarification,
				semantic.FieldError:         response.Error,
			},
			semantic.FieldPersistentWrites: true,
		},
		Execution: map[string]any{
			semantic.FieldIntent: record.Intent,
			semantic.FieldStatus: "partial",
		},
		Warnings: []string{"operation_batch_partial_execute"},
		TraceID:  "operation-batch-configure-partial",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  apiCalls + responseMetricInt(response, semantic.FieldAPICalls),
			semantic.FieldCacheHits: 0,
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
