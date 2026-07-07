package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/operation"
	localruntime "github.com/yeelight/yeelight-home/internal/runtime"
	"github.com/yeelight/yeelight-home/internal/semantic"
)

func (app *app) runInvoke(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil || !invokeFlagsAllowed(flags) || !flags.bool("stdin") {
		_, _ = fmt.Fprintln(stderr, "usage: yeelight-home invoke --stdin [--profile <name>] [--region <region>] [--house-id <id>] [--dry-run|--preview-only]")
		return exitInvalidInput
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read stdin: %v\n", err)
		return exitInternalError
	}
	request, err := contract.DecodeRequest(data)
	if err != nil {
		if response, ok := invalidSkillRequestResponse(data, err); ok {
			encoded, encodeErr := contract.EncodeResponse(response)
			if encodeErr != nil {
				_, _ = fmt.Fprintf(stderr, "encode SkillResponse: %v\n", encodeErr)
				return exitInternalError
			}
			if _, writeErr := stdout.Write(encoded); writeErr != nil {
				_, _ = fmt.Fprintf(stderr, "write stdout: %v\n", writeErr)
				return exitInternalError
			}
			return exitOK
		}
		_, _ = fmt.Fprintf(stderr, "invalid SkillRequest: %v\n", err)
		return exitInvalidInput
	}
	response, err := app.invokeWithFlags(context.Background(), request, flags)
	if err != nil {
		response = invokeErrorResponse(request, err)
		encoded, encodeErr := contract.EncodeResponse(response)
		if encodeErr != nil {
			_, _ = fmt.Fprintf(stderr, "encode SkillResponse: %v\n", encodeErr)
			return exitInternalError
		}
		if _, writeErr := stdout.Write(encoded); writeErr != nil {
			_, _ = fmt.Fprintf(stderr, "write stdout: %v\n", writeErr)
			return exitInternalError
		}
		_, _ = fmt.Fprintf(stderr, "invoke: %v\n", err)
		return exitInternalError
	}
	encoded, err := contract.EncodeResponse(response)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode SkillResponse: %v\n", err)
		return exitInternalError
	}
	if _, err := stdout.Write(encoded); err != nil {
		_, _ = fmt.Fprintf(stderr, "write stdout: %v\n", err)
		return exitInternalError
	}
	return exitOK
}

func invalidSkillRequestResponse(data []byte, err error) (contract.Response, bool) {
	message := strings.TrimSpace(fmt.Sprint(err))
	if !strings.Contains(message, "unsupported intent") {
		return contract.Response{}, false
	}
	request, decodeErr := contract.DecodeRequestEnvelope(data)
	if decodeErr != nil || strings.TrimSpace(request.RequestID) == "" {
		return contract.Response{}, false
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "not_supported",
		UserMessage:     "当前 yeelight-home Runtime 不支持这个 intent。请改用 Skill 随附 intent-catalog.json 中的已支持意图，或先用 intent.explain 查询目标意图的公开契约。",
		Result: map[string]any{
			semantic.FieldIntent:      request.Intent,
			semantic.FieldSafeToRetry: false,
			semantic.FieldNextAction:  "use_supported_intent_from_catalog_or_call_intent_explain",
		},
		Warnings: []string{"unsupported_intent_returned_as_skill_response"},
		TraceID:  "invoke-unsupported-intent",
		Metrics:  noAPIMetrics(),
		Error: &contract.Error{
			Code:    "unsupported_intent",
			Message: message,
		},
	}, true
}

func invokeErrorResponse(request contract.Request, err error) contract.Response {
	message := strings.TrimSpace(fmt.Sprint(err))
	if message == "" {
		message = "unknown invoke error"
	}
	safeToRetry, nextAction := invokeErrorRetryPolicy(message)
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "error",
		UserMessage:     "Runtime 执行失败，已返回可解析错误；调用方可以根据 error.code、error.message 和原始语义请求继续修正或重试。",
		Result: map[string]any{
			semantic.FieldIntent:      request.Intent,
			semantic.FieldSafeToRetry: safeToRetry,
			semantic.FieldNextAction:  nextAction,
		},
		Warnings: []string{"runtime_error_returned_as_skill_response"},
		TraceID:  "invoke-error",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
		Error: &contract.Error{
			Code:    "invoke_failed",
			Message: message,
		},
	}
}

func invokeErrorRetryPolicy(message string) (bool, string) {
	normalized := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(normalized, "returned non-success business response") {
		return false, "report_backend_failure_do_not_retry_same_payload"
	}
	if strings.Contains(normalized, "unsupported") || strings.Contains(normalized, "not supported") {
		return false, "use_supported_alternative_or_ask_user"
	}
	return true, "fix_request_or_retry_after_backend_recovery"
}

func invokeFlagsAllowed(flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "stdin", "profile", "region", "house-id", "dry-run", "preview-only":
		default:
			return false
		}
	}
	return true
}

func (app *app) invoke(ctx context.Context, request contract.Request) (contract.Response, error) {
	flags := cliFlags{values: map[string]string{}}
	return app.invokeWithFlags(ctx, request, flags)
}

func (app *app) invokeWithFlags(ctx context.Context, request contract.Request, flags cliFlags) (response contract.Response, err error) {
	return app.invokeWithFlagsDirect(ctx, request, flags)
}

func (app *app) invokeWithFlagsDirect(ctx context.Context, request contract.Request, flags cliFlags) (contract.Response, error) {
	originalPrepared := app.preparedOperation
	app.preparedOperation = nil
	defer func() {
		app.preparedOperation = originalPrepared
	}()
	response, err := app.invokeWithFlagsRaw(ctx, request, flags)
	if err != nil {
		return contract.Response{}, err
	}
	if shouldReturnPreviewOnly(request, flags) {
		if isPreparedExecutionResponse(response) {
			return previewOnlyResponse(response), nil
		}
		if isDirectNoWritePreviewResponse(response) {
			return response, nil
		}
		if response.Status == "success" || response.Status == "partial" {
			response.Warnings = appendWarning(response.Warnings, "dry_run_no_cloud_write_not_available_for_direct_execution")
		}
		return response, nil
	}
	if !isPreparedExecutionResponse(response) {
		return response, nil
	}
	record := app.preparedOperation
	if record == nil {
		return response, nil
	}
	if record.Risk == operation.RiskR3 && !requestBool(request.Parameters, semantic.FieldConfirmed) {
		return r3ConfirmationRequiredResponse(request, response), nil
	}
	executed, err := app.executeTransientPreparedOperation(ctx, request, flags, *record)
	if err != nil {
		return contract.Response{}, err
	}
	mergePreviewCacheHits(executed.Metrics, response.Metrics)
	executed.Warnings = appendWarning(executed.Warnings, "runtime_direct_execution_after_validation")
	if executed.Execution == nil {
		executed.Execution = map[string]any{}
	}
	executed.Execution[semantic.FieldExecutionModel] = "direct"
	return executed, nil
}

func r3ConfirmationRequiredResponse(request contract.Request, previewResponse contract.Response) contract.Response {
	preview := copyRequestMap(requestMap(previewResponse.Result[semantic.FieldPreview]))
	clarification := map[string]any{
		semantic.FieldReason: "explicit_confirmation_required",
		semantic.FieldAcceptedFields: []string{
			semantic.ParameterPath(semantic.FieldConfirmed),
		},
		semantic.FieldNextStep: "Ask the user for explicit natural-language confirmation, then resend the same request with parameters.confirmed=true. Do not execute destructive or permission-sensitive operations without that flag.",
	}
	if len(preview) > 0 {
		clarification[semantic.FieldPreview] = preview
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "这是高影响操作，需要用户明确确认后才会执行。",
		Clarification:   clarification,
		Warnings:        []string{"r3_explicit_confirmation_required"},
		TraceID:         "r3-confirmation-required",
		Metrics:         previewResponse.Metrics,
	}
}

func isDirectNoWritePreviewResponse(response contract.Response) bool {
	if response.TraceID != "direct-write-preview" && response.TraceID != "lighting-experience-apply-preview" {
		return false
	}
	if response.Result == nil {
		return false
	}
	return requestBool(response.Result, semantic.FieldDryRun)
}

func mergePreviewCacheHits(target map[string]any, preview map[string]any) {
	if target == nil || preview == nil {
		return
	}
	targetHits := metricInt(target[semantic.FieldCacheHits])
	previewHits := metricInt(preview[semantic.FieldCacheHits])
	if previewHits <= 0 {
		return
	}
	target[semantic.FieldCacheHits] = targetHits + previewHits
}

func metricInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	default:
		return 0
	}
}

func (app *app) executeTransientPreparedOperation(ctx context.Context, request contract.Request, flags cliFlags, record operation.Prepared) (contract.Response, error) {
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return contract.Response{}, err
	}
	if record.Profile != context.Profile {
		return executionBlockedResponse(request, "profile_mismatch", "内部执行载荷不属于当前本地 profile。"), nil
	}
	if record.Region != context.Region {
		return executionBlockedResponse(request, "region_mismatch", "内部执行载荷环境与当前 Runtime 环境不一致。"), nil
	}
	if err := record.Verify(time.Now()); err != nil {
		return executionVerifyBlockedResponse(request, record, err), nil
	}
	response, err := app.executePreparedExecution(ctx, request, context.Endpoint, record, context.AccessToken, context.ClientID)
	if err != nil {
		return contract.Response{}, err
	}
	response = app.refreshTopologyCacheAfterWrite(ctx, context.Endpoint, record, context.AccessToken, context.ClientID, response)
	return response, nil
}

func previewOnlyResponse(response contract.Response) contract.Response {
	preview := copyRequestMap(requestMap(response.Result[semantic.FieldPreview]))
	response.Status = "success"
	response.UserMessage = "已生成执行预览；调用方确认后可重新发送同一语义请求直接执行。"
	response.Result = map[string]any{
		semantic.FieldDryRun:         true,
		semantic.FieldPreview:        preview,
		semantic.FieldExecutionModel: "resend_same_intent_after_user_confirmation",
	}
	response.TraceID = "invoke-preview"
	response.Warnings = appendWarning(response.Warnings, "dry_run_no_cloud_write")
	return response
}

func isPreparedExecutionResponse(response contract.Response) bool {
	preview := requestMap(response.Result[semantic.FieldPreview])
	return preview != nil && requestString(preview[semantic.FieldExecutionModel]) == "ordinary_invoke_executes_directly"
}

func shouldReturnPreviewOnly(request contract.Request, flags cliFlags) bool {
	if flags.bool("dry-run") || flags.bool("preview-only") {
		return true
	}
	for _, source := range []map[string]any{request.Options, request.Parameters} {
		if requestBool(source, semantic.FieldPreviewOnly, semantic.FieldDryRun) {
			return true
		}
	}
	return false
}

func (app *app) invokeWithFlagsRaw(ctx context.Context, request contract.Request, flags cliFlags) (response contract.Response, err error) {
	if flags.values == nil {
		flags.values = map[string]string{}
	}
	if request.Options == nil {
		request.Options = map[string]any{}
	}
	if flags.bool("dry-run") {
		request.Options[semantic.FieldDryRun] = true
	}
	if flags.bool("preview-only") {
		request.Options[semantic.FieldPreviewOnly] = true
	}
	if region := requestString(request.Parameters[semantic.FieldRegion]); region != "" {
		if flags.string("region", "") == "" {
			flags.values[semantic.FieldRegion] = region
		}
	}
	if houseID := requestHouseID(request); houseID != "" {
		if flags.string("house-id", "") == "" {
			flags.values["house-id"] = houseID
		}
	}
	context, err := app.resolveRuntimeContext(flags)
	if err != nil {
		return contract.Response{}, err
	}
	profile := context.Profile
	if !context.TokenPresent && !isLocalOnlyInvokeIntent(request.Intent) {
		return localruntime.NewEngine(false).Invoke(request), nil
	}
	if !isImplementedInvokeIntent(request.Intent) {
		return localruntime.NewEngine(true).Invoke(request), nil
	}
	region := context.Region
	clientID := context.ClientID
	houseID := context.HouseID
	houseIndependent := requestRunsHouseIndependent(request)
	if houseIndependent {
		houseID = ""
	}
	endpoint := context.Endpoint
	accessToken := context.AccessToken
	if !houseIndependent && !isLocalOnlyInvokeIntent(request.Intent) && requestHouseID(request) == "" && !skipImplicitHomeResolution(request) {
		resolvedHouseID, resolveErr := app.resolveRequestHouseID(ctx, request, endpoint, accessToken, clientID)
		if resolveErr != nil {
			return contract.Response{}, resolveErr
		}
		if resolvedHouseID != "" {
			houseID = resolvedHouseID
		} else if strings.TrimSpace(firstRequestString(request.HomeRef, semantic.FieldName, semantic.FieldHouseName)) != "" {
			return configureClarificationResponse(request, "home_name_not_found_or_ambiguous", []string{
				semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldName),
				semantic.FieldPath(semantic.FieldHomeRef, semantic.FieldID),
				semantic.ParameterPath(semantic.FieldHouseID),
			}), nil
		}
	}
	if shouldReturnPreviewOnly(request, flags) {
		if preview, handled, err := app.previewDirectWriteIntent(ctx, request, endpoint, profile, region, houseID, accessToken, clientID); handled {
			return preview, err
		}
	}
	defer func() {
		if err != nil {
			return
		}
		if observeErr := app.observeMemorySignal(request, profile, region, houseID, response); observeErr != nil {
			err = observeErr
		}
	}()
	switch request.Intent {
	case "intent.explain":
		return invokeIntentExplain(request), nil
	case "account.info":
		return app.invokeAccountInfo(ctx, request, endpoint, accessToken, clientID)
	case "home.member.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, homeMemberListSpec())
	case "home.member.current.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.member.current.get", "已读取当前家庭成员的脱敏只读信息。"))
	case "device.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.detail.get", "已读取设备详情的安全摘要。"))
	case "device.attr.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.attr.list", "已读取设备属性的安全摘要。"))
	case "device.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.list", "已读取家庭设备候选列表。"))
	case "room.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.detail.get", "已读取房间详情的安全摘要。"))
	case "room.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.list", "已读取家庭房间列表。"))
	case "room.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("room.search", "已搜索家庭房间候选。"))
	case "area.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("area.detail.get", "已读取区域详情的安全摘要。"))
	case "home.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.detail.get", "已读取家庭详情的安全摘要。"))
	case "home.stat.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("home.stat.get", "已读取家庭统计的安全摘要。"))
	case "geo_area.children.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("geo_area.children.list", "已读取地理区域下级城市候选。"))
	case "geo_area.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("geo_area.search", "已按名称搜索地理区域候选。"))
	case "group.structure.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.structure.list", "已读取设备组结构的安全摘要。"))
	case "group.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.list", "已读取设备组列表。"))
	case "group.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.search", "已搜索设备组候选。"))
	case "group.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("group.detail.get", "已读取设备组详情的安全摘要。"))
	case "scene.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.detail.get", "已读取情景详情的安全摘要。"))
	case "scene.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.list", "已读取家庭情景列表。"))
	case "scene.scoped.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.scoped.list", "已按家庭或房间读取情景列表。"))
	case "scene.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("scene.search", "已搜索家庭情景。"))
	case "automation.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.list", "已读取家庭自动化列表。"))
	case "automation.supported.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.supported.list", "已读取自动化支持能力。"))
	case "automation.supported.v2.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.supported.v2.list", "已读取自动化 V2 支持能力。"))
	case "automation.rule.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.rule.list", "已读取家庭规则列表。"))
	case "automation.list.page":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.list.page", "已分页读取自动化列表。"))
	case "automation.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("automation.detail.get", "已读取自动化详情的安全摘要。"))
	case "schedule_job.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("schedule_job.list", "已读取家庭定时任务列表。"))
	case "message.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("message.list", "已读取当前账号消息列表的安全摘要。"))
	case "sensor.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("sensor.list", "已读取家庭传感器列表的安全摘要。"))
	case "sensor.event.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("sensor.event.list", "已读取传感器事件列表的安全摘要。"))
	case "device.energy.summary":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.energy.summary", "已读取设备用电摘要。"))
	case "device.weather.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.weather.get", "已读取设备天气上下文。"))
	case "device.virtual_count.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("device.virtual_count.get", "已读取家庭虚拟设备数量。"))
	case "meshgroup.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("meshgroup.detail.get", "已读取 Mesh 组详情的安全摘要。"))
	case "node.sorted_device.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("node.sorted_device.list", "已读取节点下设备排序列表。"))
	case "gateway.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.detail.get", "已读取网关详情的安全摘要。"))
	case "gateway.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.list", "已读取家庭网关列表的安全摘要。"))
	case "gateway.thread.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.thread.get", "已读取网关 Thread 信息的安全摘要。"))
	case "gateway.stats.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.stats.list", "已读取家庭网关统计的安全摘要。"))
	case "gateway.scene_relation.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("gateway.scene_relation.list", "已读取网关关联情景的安全摘要。"))
	case "product.pedia.search":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("product.pedia.search", "已搜索产品百科资料和候选说明书/FAQ 资源。"))
	case "home.summary":
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunList(ctx, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		})
		if err != nil {
			return contract.Response{}, err
		}
		return homeSummaryResponse(request, summary), nil
	case "home.list":
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunList(ctx, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		})
		if err != nil {
			return contract.Response{}, err
		}
		return homeListResponse(request, summary, "home-list-readonly", "已读取账号下家庭列表。"), nil
	case "home.search":
		if strings.TrimSpace(firstNonEmptyString(
			requestString(request.Parameters[semantic.FieldFuzzyName]),
			requestString(request.Parameters[semantic.FieldName]),
			requestString(request.Parameters[semantic.FieldKeyword]),
			requestString(request.Parameters[semantic.FieldQuery]),
		)) == "" {
			return homeSearchClarificationResponse(request), nil
		}
		summary, err := api.NewHomeSummaryClient(endpoint, nil).RunSearch(ctx, request.Parameters, api.HomeSummaryCredentials{
			Authorization: accessToken,
			ClientID:      clientID,
		})
		if err != nil {
			return contract.Response{}, err
		}
		return homeListResponse(request, summary, "home-search-readonly", "已搜索账号下匹配家庭。"), nil
	case "entity.list":
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		entities, err := app.loadEntities(ctx, endpoint, profile, region, houseID, accessToken, clientID, entityLoadOptions{Refresh: true})
		if err != nil {
			return contract.Response{}, err
		}
		return entityListResponse(request, entities), nil
	case "entity.get":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return entityGetClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, accessToken, clientID, target)
		if err != nil {
			return contract.Response{}, err
		}
		entities := resolved.Entities
		match := resolved.Match
		candidates := resolved.Candidates
		if len(candidates) > 1 && target.id == "" {
			return entityGetClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.ID == "" {
			return entityGetClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		return entityGetResponse(request, entities, match, resolved.MatchedBy), nil
	case "entity.capabilities":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return entityCapabilitiesClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, accessToken, clientID, target)
		if err != nil {
			return contract.Response{}, err
		}
		entities := resolved.Entities
		match := resolved.Match
		candidates := resolved.Candidates
		if len(candidates) > 1 && target.id == "" {
			return entityCapabilitiesClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.ID == "" {
			return entityCapabilitiesClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.Type == "device" {
			capabilities, err := api.NewDeviceCapabilitiesClient(endpoint, nil).Run(ctx, api.DeviceCapabilitiesRequest{
				HouseID:  houseID,
				DeviceID: match.ID,
				Credentials: api.DeviceCapabilitiesCredentials{
					Authorization: accessToken,
					ClientID:      clientID,
				},
			})
			if err == nil {
				return entityDeviceCapabilitiesResponse(request, entities, match, capabilities), nil
			}
			return entityCapabilitiesFallbackResponse(request, entities, match, "设备实例级 schema 读取失败，已降级为保守能力边界。"), nil
		}
		return entityCapabilitiesResponse(request, entities, match), nil
	case "state.query":
		target := entityGetTargetFromRequest(request)
		if target.id == "" && target.name == "" {
			return stateQueryClarificationResponse(request, "missing_target", target, nil, 0), nil
		}
		if requestHouseID := requestHouseID(request); requestHouseID != "" {
			houseID = requestHouseID
		}
		resolved, err := app.resolveEntity(ctx, endpoint, profile, region, houseID, accessToken, clientID, target)
		if err != nil {
			return contract.Response{}, err
		}
		entities := resolved.Entities
		match := resolved.Match
		candidates := resolved.Candidates
		if match.ID == "" {
			return stateQueryClarificationResponse(request, "entity_not_found", target, candidates, entityListAPICalls(entities)), nil
		}
		if len(candidates) > 1 && target.id == "" {
			return stateQueryClarificationResponse(request, "ambiguous_target", target, candidates, entityListAPICalls(entities)), nil
		}
		if match.Type != "device" {
			return stateQueryClarificationResponse(request, "target_not_device", target, []api.EntitySummary{match}, entityListAPICalls(entities)), nil
		}
		propertyID := stateQueryPropertyName(request)
		if propertyID != "" && semantic.PropertySensitive(propertyID) {
			return stateQuerySensitivePropertyResponse(request, propertyID), nil
		}
		propertySet := []string{}
		if propertyID == "" {
			if capabilities, err := api.NewDeviceCapabilitiesClient(endpoint, nil).Run(ctx, api.DeviceCapabilitiesRequest{
				HouseID:  houseID,
				DeviceID: match.ID,
				Credentials: api.DeviceCapabilitiesCredentials{
					Authorization: accessToken,
					ClientID:      clientID,
				},
			}); err == nil {
				propertySet = stateQueryPropertySet(capabilities.Device)
			}
		}
		state, err := api.NewStateQueryClient(endpoint, nil).Run(ctx, api.StateQueryRequest{
			DeviceID:     match.ID,
			PropertyName: propertyID,
			PropertySet:  propertySet,
			Credentials: api.StateQueryCredentials{
				Authorization: accessToken,
				ClientID:      clientID,
			},
		})
		if err != nil {
			return contract.Response{}, err
		}
		return stateQueryResponse(request, entities, match, state), nil
	case "scene.execute":
		return app.invokeSceneExecute(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "scene.test":
		return app.invokeSceneTest(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "light.power.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightPowerSpec())
	case "light.brightness.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightBrightnessSpec())
	case "light.brightness.adjust":
		return app.invokeLightPropertyAdjust(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightBrightnessAdjustSpec())
	case "light.color_temperature.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightColorTemperatureSpec())
	case "light.color_temperature.adjust":
		return app.invokeLightPropertyAdjust(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightColorTemperatureAdjustSpec())
	case "light.color.set":
		return app.invokeLightPropertySet(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, lightColorSpec())
	case "lighting.experience.apply":
		return app.invokeLightingExperienceApply(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "diagnose.device":
		return app.invokeDiagnoseDevice(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.gateway":
		return app.invokeDiagnoseGateway(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.scene":
		return app.invokeDiagnoseScene(ctx, request, endpoint, houseID, accessToken, clientID)
	case "diagnose.automation":
		return app.invokeDiagnoseAutomation(ctx, request, endpoint, houseID, accessToken, clientID)
	case "automation.explain":
		return app.invokeAutomationExplain(ctx, request, endpoint, houseID, accessToken, clientID)
	case "automation.capabilities":
		return app.prepareMetadataLocal(request, houseID, automationCapabilitiesSpec())
	case "panel.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, panelGetSpec())
	case "panel.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("panel.list", "已读取家庭下面板列表的安全摘要。"))
	case "panel.button.type.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("panel.button.type.get", "已按类型读取面板按键配置的安全摘要。"))
	case "screen.control.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("screen.control.list", "已读取屏控制设备列表的安全摘要。"))
	case "knob.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, knobGetSpec())
	case "upgrade.file.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.file.list", "已读取设备可用升级文件的安全摘要。"))
	case "upgrade.progress.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.progress.get", "已读取设备升级进度的安全摘要。"))
	case "upgrade.file.batch_list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("upgrade.file.batch_list", "已批量读取设备可用升级文件的安全摘要。"))
	case "progress.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("progress.get", "已读取任务进度的安全摘要。"))
	case "app_upgrade.latest.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("app_upgrade.latest.get", "已读取 App 最新升级版本的安全摘要。"))
	case "ota.version_file.batch_list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("ota.version_file.batch_list", "已按版本批量读取 OTA 升级文件的安全摘要。"))
	case "node.property_config.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("node.property_config.get", "已读取节点属性配置的安全摘要。"))
	case "thing.schema.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.list", "已读取可见产品物模型列表的安全摘要。"))
	case "thing.schema.detail.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.detail.list", "已读取可见产品物模型详情列表的安全摘要。"))
	case "thing.schema.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.get", "已读取产品物模型详情的安全摘要。"))
	case "thing.schema.event.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.schema.event.list", "已读取产品事件物模型的安全摘要。"))
	case "thing.product.info.batch_get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.info.batch_get", "已批量读取产品定义的安全摘要。"))
	case "thing.product.info.v3.batch_get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.info.v3.batch_get", "已按版本批量读取产品定义的安全摘要。"))
	case "thing.product.list.v3":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product.list.v3", "已读取版本化产品列表的安全摘要。"))
	case "thing.product_domain.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_domain.list", "已读取产品域目录的安全摘要。"))
	case "thing.product_faq.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.list", "已读取产品帮助 FAQ 列表的安全摘要。"))
	case "thing.product_faq.detail.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.detail.get", "已读取产品帮助 FAQ 详情的安全摘要。"))
	case "thing.product_faq.type.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.type.list", "已读取产品帮助 FAQ 类型列表。"))
	case "thing.product_faq.item_type.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.item_type.list", "已读取产品帮助 FAQ 项类型列表。"))
	case "thing.product_faq.locale.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.locale.list", "已读取产品帮助 FAQ 支持语言列表。"))
	case "thing.product_faq.page.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.page.list", "已分页读取产品帮助 FAQ 列表的安全摘要。"))
	case "thing.product_faq.page_detail.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.product_faq.page_detail.list", "已分页读取产品帮助 FAQ 详情列表的安全摘要。"))
	case "thing.category.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.category.list", "已读取物模型品类列表的安全摘要。"))
	case "thing.component.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.component.list", "已读取物模型组件列表的安全摘要。"))
	case "thing.component.get":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.component.get", "已读取物模型组件详情的安全摘要。"))
	case "thing.property.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("thing.property.list", "已读取物模型属性列表的安全摘要。"))
	case "device.storage.get":
		return app.invokeMetadataReadonlyProjection(ctx, request, endpoint, houseID, accessToken, clientID, deviceStorageGetSpec())
	case "ai_voice.list":
		return app.invokeMetadataReadonlyProjection(ctx, request, endpoint, houseID, accessToken, clientID, aiVoiceListSpec())
	case "ai_voice.product.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, metadataDetailReadonlySpec("ai_voice.product.list", "已读取支持 AI 语音能力的产品列表安全摘要。"))
	case "favorite.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, favoriteListSpec())
	case "favorite.plan":
		return app.prepareMetadataLocal(request, houseID, favoritePlanSpec())
	case "home.sort.list":
		return app.invokeMetadataCloudReadonly(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, homeSortListSpec())
	case "home.sort.configure", "favorite.add", "favorite.update", "favorite.delete", "favorite.batch_add", "favorite.batch_update", "favorite.batch_delete":
		return app.prepareHomeOrganization(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.member.invite", "home.member.accept_share", "home.member.configure", "home.member.remove", "home.member.transfer", "home.member.quit":
		return app.prepareHomeMember(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.lock_all", "home.unlock_all":
		return app.prepareHomeLock(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "home.create":
		return app.prepareHomeCreate(ctx, request, endpoint, profile, region, accessToken, clientID)
	case "home.update", "room.batch_create", "room.batch_update", "room.area.configure":
		return app.prepareHomeSpaceConfiguration(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.rename", "room.update", "area.update", "device.rename", "device.move", "group.update":
		return app.prepareSpaceOrganization(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.move_room.batch":
		return app.prepareSpaceBatchOrganization(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.remove", "gateway.delete", "home.delete":
		return app.prepareDestructiveDelete(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "device.unbind":
		return app.prepareDeviceUnbind(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "gateway.configure":
		return app.prepareGatewayConfiguration(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "entity.rename.batch":
		return app.prepareEntityBatchRename(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.delete", "area.delete", "group.delete", "scene.delete", "automation.delete":
		return app.prepareMetadataDelete(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.batch_delete", "area.batch_delete", "group.batch_delete", "scene.batch_delete", "automation.batch_delete":
		return app.prepareMetadataBatchDelete(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "panel.button.configure", "panel.button_event.update", "panel.button_event.batch_update", "panel.button_event.reset", "knob.configure", "knob.reset":
		return app.preparePanelConfiguration(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "lighting.design.plan":
		return app.prepareLightingDesign(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "lighting.design.apply":
		return app.prepareLightingDesignApply(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "lighting.design.import", "device.slot.create":
		return app.prepareLightingDesignImport(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "room.create":
		return app.prepareRoomCreate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "area.create":
		return app.prepareMetadataCreate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, areaCreateSpec())
	case "group.create":
		return app.prepareMetadataCreate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, groupCreateSpec())
	case "scene.create":
		return app.prepareMetadataCreate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, sceneCreateSpec())
	case "scene.update":
		return app.prepareSceneUpdate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "automation.create":
		return app.prepareMetadataCreate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID, automationCreateSpec())
	case "automation.update":
		return app.prepareAutomationUpdate(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "automation.enable", "automation.disable":
		return app.prepareAutomationStatus(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	case "memory.remember":
		return app.invokeMemoryRemember(request, profile, region, houseID)
	case "memory.list":
		return app.invokeMemoryList(request, profile, region, houseID)
	case "memory.pause":
		return app.invokeMemoryPauseResume(request, profile, region, houseID, true)
	case "memory.resume":
		return app.invokeMemoryPauseResume(request, profile, region, houseID, false)
	case "memory.forget":
		return app.invokeMemoryForget(request, profile, region, houseID)
	case "recommendation.list":
		return app.invokeRecommendationList(request, profile, region, houseID)
	case "recommendation.record":
		return app.invokeRecommendationRecord(request, profile, region, houseID)
	case "recommendation.feedback":
		return app.invokeRecommendationFeedback(request, profile, region, houseID)
	case "operation.lesson.record":
		return app.invokeOperationLessonRecord(request, profile, region, houseID)
	case "operation.lesson.list":
		return app.invokeOperationLessonList(request, profile, region, houseID)
	case "operation.batch.configure":
		return app.prepareOperationBatchConfigure(ctx, request, endpoint, profile, region, houseID, accessToken, clientID)
	default:
		return localruntime.NewEngine(true).Invoke(request), nil
	}
}

func skipImplicitHomeResolution(request contract.Request) bool {
	if request.Intent != "lighting.design.import" {
		return false
	}
	if requestHouseID(request) != "" {
		return false
	}
	if requestBool(request.HomeRef, semantic.FieldUseCurrent) {
		return false
	}
	return true
}

func requestRunsHouseIndependent(request contract.Request) bool {
	if !isHouseIndependentInvokeIntent(request.Intent) {
		return false
	}
	if metadataReadonlyIntentNeedsEntityID(request.Intent) && requestHouseID(request) != "" {
		return false
	}
	return true
}

func (app *app) resolveRequestHouseID(ctx context.Context, request contract.Request, endpoint api.Endpoint, accessToken string, clientID string) (string, error) {
	if houseID := requestHouseID(request); houseID != "" {
		return houseID, nil
	}
	homeName := strings.TrimSpace(firstRequestString(request.HomeRef, semantic.FieldName, semantic.FieldHouseName))
	if homeName == "" {
		return "", nil
	}
	summary, err := api.NewHomeSummaryClient(endpoint, nil).RunList(ctx, api.HomeSummaryCredentials{
		Authorization: accessToken,
		ClientID:      clientID,
	})
	if err != nil {
		return "", err
	}
	ranked := semantic.RankNameMatches(homeName, summary.Houses, func(house api.HouseSummary) string {
		return house.Name
	})
	if len(ranked) == 0 {
		return "", nil
	}
	if ranked[0].Match.Kind == "name" {
		exact := 0
		matchID := ""
		for _, match := range ranked {
			if match.Match.Kind == "name" {
				exact++
				matchID = match.Value.ID
			}
		}
		if exact == 1 {
			return matchID, nil
		}
		return "", nil
	}
	second := semantic.NameMatch{}
	if len(ranked) > 1 {
		second = ranked[1].Match
	}
	if semantic.NameMatchAutoAccept(ranked[0].Match, second) {
		return ranked[0].Value.ID, nil
	}
	return "", nil
}

func homeSummaryResponse(request contract.Request, summary api.HomeSummaryResult) contract.Response {
	houses := make([]any, 0, len(summary.Houses))
	for _, house := range summary.Houses {
		houses = append(houses, map[string]any{
			semantic.FieldHouseID: house.ID,
			semantic.FieldID:      house.ID,
			semantic.FieldName:    house.Name,
		})
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("已找到 %d 个家庭。", summary.HouseCount),
		Result: map[string]any{
			semantic.FieldRegion:     summary.Region,
			semantic.FieldHouseCount: summary.HouseCount,
			semantic.FieldHouses:     houses,
			semantic.FieldSource:     summary.Source,
		},
		Warnings: []string{},
		TraceID:  "home-summary-readonly",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  firstPositive(summary.APICalls, 1),
			semantic.FieldCacheHits: 0,
		},
	}
}

func homeListResponse(request contract.Request, summary api.HomeSummaryResult, traceID string, message string) contract.Response {
	houses := make([]any, 0, len(summary.Houses))
	for _, house := range summary.Houses {
		item := map[string]any{
			semantic.FieldHouseID: house.ID,
			semantic.FieldID:      house.ID,
			semantic.FieldName:    house.Name,
		}
		for key, value := range map[string]string{
			semantic.FieldIcon:        house.Icon,
			semantic.FieldDescription: house.Desc,
			semantic.FieldAreaCode:    house.AreaCode,
			semantic.FieldAreaName:    house.AreaName,
		} {
			if strings.TrimSpace(value) != "" {
				item[key] = value
			}
		}
		if len(house.Counts) > 0 {
			item[semantic.FieldCounts] = house.Counts
		}
		houses = append(houses, item)
	}
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "success",
		UserMessage:     fmt.Sprintf("%s共 %d 个候选家庭。", message, summary.HouseCount),
		Result: map[string]any{
			semantic.FieldRegion:     summary.Region,
			semantic.FieldHouseCount: summary.HouseCount,
			semantic.FieldHouses:     houses,
			semantic.FieldSource:     summary.Source,
		},
		Warnings: []string{},
		TraceID:  traceID,
		Metrics: map[string]any{
			semantic.FieldAPICalls:  firstPositive(summary.APICalls, 1),
			semantic.FieldCacheHits: 0,
		},
	}
}

func homeSearchClarificationResponse(request contract.Request) contract.Response {
	return contract.Response{
		ContractVersion: contract.Version,
		RequestID:       request.RequestID,
		Status:          "clarification_required",
		UserMessage:     "请提供要搜索的家庭名称关键词。",
		Clarification: map[string]any{
			semantic.FieldReason:         "home_search_keyword_missing",
			semantic.FieldRequiredFields: []string{semantic.ParameterPath(semantic.FieldName)},
		},
		Warnings: []string{},
		TraceID:  "home-search-clarification",
		Metrics: map[string]any{
			semantic.FieldAPICalls:  0,
			semantic.FieldCacheHits: 0,
		},
	}
}

func firstPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
