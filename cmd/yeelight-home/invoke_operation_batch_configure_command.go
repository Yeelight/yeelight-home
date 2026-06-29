package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
)

const maxOperationBatchConfigureSteps = 20

type operationBatchStep struct {
	Intent     string
	HouseID    string
	Payload    map[string]any
	Summary    string
	Preview    map[string]any
	APICalls   int
	StepNumber int
}

func (app *app) prepareOperationBatchConfigure(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return configureClarificationResponse(request, "missing_house_id", []string{"parameters.houseId", "homeRef.id", "local profile houseId"}), nil
	}
	rawSteps, ok := requestMapList(firstNonNil(request.Parameters["operations"], request.Parameters["steps"]))
	if !ok || len(rawSteps) == 0 || len(rawSteps) > maxOperationBatchConfigureSteps {
		return operationBatchConfigureClarificationResponse(request, "invalid_operation_batch_payload"), nil
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
	steps := make([]operationBatchStep, 0, len(rawSteps))
	extraAPICalls := 0
	for index, raw := range rawSteps {
		step, reason, err := app.buildOperationBatchConfigureStep(ctx, request, endpoint, profile, region, houseID, authorization, clientID, entities, raw, index+1)
		if err != nil {
			return contract.Response{}, err
		}
		if reason != "" {
			return operationBatchConfigureClarificationResponse(request, reason), nil
		}
		steps = append(steps, step)
		extraAPICalls += step.APICalls
	}
	payload := map[string]any{
		"houseId": requestNumberOrString(houseID),
		"steps":   operationBatchStepsPayload(steps),
	}
	record, err := operation.NewPrepared(profile, region, houseID, "operation.batch.configure", request.RequestID, fmt.Sprintf("批量配置%d个添加或修改操作", len(steps)), payload, []string{
		"执行前重新校验 profile、region、家庭和请求载荷",
		"批量配置只允许 Runtime 白名单内的新增、更新、排序、收藏、面板、旋钮、自动化状态和照明设计导入操作",
		"删除、解绑、成员移除/转移、家庭删除、网关删除、全屋锁定/解锁等高影响动作必须使用独立语义请求",
		"Runtime 根据 operations 构建受控批量 payload",
		"执行时按步骤顺序运行，并复用每个单项能力自己的云端写入和读后验证逻辑",
	}, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	preview := map[string]any{
		"mode":        "direct_batch_configure",
		"stepCount":   len(steps),
		"steps":       operationBatchStepsPreview(steps),
		"exclusions":  operationBatchConfigureExclusions(),
		"writePolicy": "direct_execute_allowlisted_add_update_configure_steps",
	}
	return executionPreviewResponseWithDetails(request, record, entities, preview, extraAPICalls), nil
}

func operationBatchConfigureClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, []string{"parameters.operations[].intent", "parameters.operations[].parameters"}, operationBatchConfigurePayloadGuide())
}

func (app *app) buildOperationBatchConfigureStep(ctx context.Context, parent contract.Request, endpoint api.Endpoint, profile string, region string, defaultHouseID string, authorization string, clientID string, entities api.EntityListResult, raw map[string]any, stepNumber int) (operationBatchStep, string, error) {
	intent := strings.TrimSpace(requestString(raw["intent"]))
	if intent == "home.create" {
		return operationBatchStep{}, "operation_batch_contains_account_scoped_intent", nil
	}
	if !operationBatchConfigureAllowedIntent(intent) {
		if operationBatchConfigureStrictIntent(intent) {
			return operationBatchStep{}, "operation_batch_contains_strict_or_destructive_intent", nil
		}
		return operationBatchStep{}, "operation_batch_contains_unsupported_intent", nil
	}
	parameters, ok := raw["parameters"].(map[string]any)
	if !ok {
		parameters = map[string]any{}
	}
	houseID := defaultHouseID
	if value := requestHouseID(contract.Request{Parameters: parameters}); value != "" {
		houseID = value
	}
	if strings.TrimSpace(houseID) == "" {
		return operationBatchStep{}, "missing_house_id", nil
	}
	if houseID != defaultHouseID {
		return operationBatchStep{}, "operation_batch_cross_house_not_allowed", nil
	}
	stepRequest := contract.Request{
		ContractVersion: parent.ContractVersion,
		RequestID:       fmt.Sprintf("%s-step-%02d", parent.RequestID, stepNumber),
		SessionID:       parent.SessionID,
		Locale:          parent.Locale,
		Utterance:       parent.Utterance,
		Intent:          intent,
		HomeRef:         parent.HomeRef,
		Targets:         requestTargets(raw, parent.Targets),
		Parameters:      copyRequestMap(parameters),
		Context:         parent.Context,
		Options:         parent.Options,
	}
	if stepRequest.Parameters == nil {
		stepRequest.Parameters = map[string]any{}
	}
	if _, exists := stepRequest.Parameters["houseId"]; !exists {
		stepRequest.Parameters["houseId"] = houseID
	}
	payload, preconditions, summary, preview, calls, reason, err := app.buildOperationBatchStepPlanPayload(ctx, stepRequest, endpoint, profile, region, houseID, authorization, clientID, entities)
	if err != nil || reason != "" {
		return operationBatchStep{}, reason, err
	}
	_ = preconditions
	return operationBatchStep{
		Intent:     intent,
		HouseID:    houseID,
		Payload:    payload,
		Summary:    summary,
		Preview:    preview,
		APICalls:   calls,
		StepNumber: stepNumber,
	}, "", nil
}

func (app *app) executeOperationBatchConfigure(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	steps, reason := operationBatchStepsFromPlan(record.Payload)
	if reason != "" {
		return executionBlockedResponse(request, reason, "批量配置载荷无效，未执行。"), nil
	}
	results := make([]any, 0, len(steps))
	totalAPICalls := 0
	for index, step := range steps {
		stepRecord := record
		stepRecord.Intent = step.Intent
		stepRecord.HouseID = step.HouseID
		stepRecord.Summary = step.Summary
		stepRecord.Payload = step.Payload
		stepRecord.Preconditions = nil
		stepRequest := request
		stepRequest.Intent = step.Intent
		stepRequest.RequestID = fmt.Sprintf("%s-step-%02d", request.RequestID, index+1)
		response, err := app.executePreparedExecution(ctx, stepRequest, endpoint, stepRecord, authorization, clientID)
		if err != nil {
			return contract.Response{}, err
		}
		if response.Status != "success" {
			return operationBatchPartialResponse(request, record, results, step, response, totalAPICalls), nil
		}
		results = append(results, operationBatchStepResult(index+1, step, response))
		totalAPICalls += responseMetricInt(response, "apiCalls")
	}
	return operationBatchExecuteResponse(request, record, results, totalAPICalls), nil
}

func operationBatchStepsPayload(steps []operationBatchStep) []any {
	result := make([]any, 0, len(steps))
	for _, step := range steps {
		result = append(result, map[string]any{
			"intent":  step.Intent,
			"houseId": step.HouseID,
			"summary": step.Summary,
			"payload": step.Payload,
			"preview": step.Preview,
		})
	}
	return result
}

func operationBatchStepsPreview(steps []operationBatchStep) []any {
	result := make([]any, 0, len(steps))
	for _, step := range steps {
		item := map[string]any{
			"index":   step.StepNumber,
			"intent":  step.Intent,
			"summary": step.Summary,
		}
		if len(step.Preview) > 0 {
			item["preview"] = step.Preview
		}
		result = append(result, item)
	}
	return result
}

func operationBatchStepsFromPlan(payload map[string]any) ([]operationBatchStep, string) {
	rawSteps, ok := payload["steps"].([]any)
	if !ok || len(rawSteps) == 0 || len(rawSteps) > maxOperationBatchConfigureSteps {
		return nil, "invalid_operation_batch_payload"
	}
	steps := make([]operationBatchStep, 0, len(rawSteps))
	for index, raw := range rawSteps {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, "invalid_operation_batch_step"
		}
		intent := strings.TrimSpace(requestString(item["intent"]))
		if !operationBatchConfigureAllowedIntent(intent) {
			return nil, "operation_batch_contains_unsupported_intent"
		}
		stepPayload, ok := item["payload"].(map[string]any)
		if !ok || len(stepPayload) == 0 {
			return nil, "invalid_operation_batch_step_payload"
		}
		houseID := firstNonEmptyString(requestString(item["houseId"]), requestString(payload["houseId"]))
		if strings.TrimSpace(houseID) == "" {
			return nil, "missing_house_id"
		}
		steps = append(steps, operationBatchStep{
			Intent:     intent,
			HouseID:    houseID,
			Payload:    stepPayload,
			Summary:    requestString(item["summary"]),
			StepNumber: index + 1,
		})
	}
	return steps, ""
}

func operationBatchConfigureAllowedIntent(intent string) bool {
	switch intent {
	case "home.update",
		"home.sort.configure",
		"favorite.add",
		"favorite.update",
		"favorite.batch_add",
		"favorite.batch_update",
		"room.create",
		"room.rename",
		"room.update",
		"room.batch_create",
		"room.batch_update",
		"room.area.configure",
		"area.create",
		"area.update",
		"device.rename",
		"device.move",
		"device.move_room.batch",
		"entity.rename.batch",
		"group.create",
		"group.update",
		"scene.create",
		"scene.update",
		"automation.create",
		"automation.update",
		"automation.enable",
		"automation.disable",
		"gateway.configure",
		"panel.button.configure",
		"panel.button_event.update",
		"panel.button_event.batch_update",
		"knob.configure",
		"lighting.design.import",
		"device.slot.create":
		return true
	default:
		return false
	}
}

func operationBatchConfigureStrictIntent(intent string) bool {
	switch intent {
	case "favorite.delete",
		"favorite.batch_delete",
		"room.delete",
		"room.batch_delete",
		"area.delete",
		"area.batch_delete",
		"group.delete",
		"group.batch_delete",
		"scene.delete",
		"scene.batch_delete",
		"automation.delete",
		"automation.batch_delete",
		"device.remove",
		"device.unbind",
		"gateway.delete",
		"home.delete",
		"home.member.remove",
		"home.member.transfer",
		"home.member.quit",
		"home.lock_all",
		"home.unlock_all",
		"panel.button_event.reset",
		"knob.reset":
		return true
	default:
		return false
	}
}

func operationBatchConfigureExclusions() []string {
	return []string{
		"delete",
		"batch_delete",
		"device.remove",
		"device.unbind",
		"gateway.delete",
		"home.delete",
		"member.remove",
		"member.transfer",
		"home.lock_all",
		"home.unlock_all",
		"reset",
	}
}

func requestTargets(raw map[string]any, fallback []map[string]any) []map[string]any {
	rows, ok := raw["targets"].([]any)
	if !ok {
		return fallback
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if ok {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}

func copyRequestMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
