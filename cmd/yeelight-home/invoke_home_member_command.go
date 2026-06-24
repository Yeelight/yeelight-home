package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
	"github.com/yeelight/yeelight-home/internal/contract"
	"github.com/yeelight/yeelight-home/internal/plan"
)

func (app *app) invokeHomeMemberPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" && request.Intent != "home.member.accept_share" {
		return configureClarificationResponse(request, "missing_house_id", homeMemberAcceptedFields(request.Intent)), nil
	}
	client := api.NewHomeMemberClient(endpoint, nil)
	currentUID, accountCalls, err := client.CurrentUserID(ctx, api.HomeMemberCredentials{Authorization: authorization, ClientID: clientID})
	if err != nil {
		return contract.Response{}, err
	}
	members := []map[string]any{}
	memberCalls := 0
	if strings.TrimSpace(houseID) != "" && request.Intent != "home.member.accept_share" {
		members, memberCalls, err = client.ProbeMembers(ctx, houseID, api.HomeMemberCredentials{Authorization: authorization, ClientID: clientID})
		if err != nil {
			return contract.Response{}, err
		}
	}
	payload, preconditions, summary, risk, challenge, err := buildHomeMemberPayload(request, houseID, currentUID, members)
	if err != nil {
		return configureClarificationResponse(request, err.Error(), homeMemberAcceptedFields(request.Intent)), nil
	}
	record, err := plan.NewRecordWithRisk(profile, region, houseID, request.Intent, request.RequestID, summary, risk, challenge, payload, preconditions, time.Now(), pendingPlanTTL)
	if err != nil {
		return contract.Response{}, err
	}
	if err := app.planStore.Save(record); err != nil {
		return contract.Response{}, err
	}
	preview := homeMemberPreview(request.Intent, payload, members)
	return pendingPlanResponseWithPreview(request, record, api.EntityListResult{Region: endpoint.Region, HouseID: houseID, APICalls: memberCalls}, preview, accountCalls), nil
}

func buildHomeMemberPayload(request contract.Request, houseID string, currentUID string, members []map[string]any) (map[string]any, []string, string, string, string, error) {
	switch request.Intent {
	case "home.member.invite":
		expiredTime, ok := valueInt(firstNonNil(request.Parameters["expiredTime"], request.Parameters["expiresAt"]))
		if !ok || expiredTime <= 0 {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_invite_payload")
		}
		userRole, ok := normalizedHomeMemberRole(firstNonNil(request.Parameters["userRole"], request.Parameters["role"]))
		if !ok {
			userRole = 0
		}
		reuseBarcode := requestBoolDefault(request.Parameters["reuseBarcode"], true)
		return map[string]any{
				"houseId":      requestNumberOrString(houseID),
				"expiredTime":  expiredTime,
				"userRole":     userRole,
				"reuseBarcode": reuseBarcode,
			}, []string{
				"提交前重新读取家庭成员列表",
				"只有当前账号具备家庭分享权限时云端才会生成分享码",
				"生成结果只返回脱敏后的分享证据，不暴露 Token 或原始接口信息",
			}, "生成家庭分享邀请码", plan.RiskR2, "", nil
	case "home.member.accept_share":
		shareID := firstValueIDString(request.Parameters, "shareId", "id")
		createTime, ok := valueInt(request.Parameters["createTime"])
		acceptedHouseID := firstValueIDString(request.Parameters, "houseId", "resId")
		if acceptedHouseID == "" {
			acceptedHouseID = houseID
		}
		if shareID == "" || !ok || createTime <= 0 || acceptedHouseID == "" || currentUID == "" {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_accept_share_payload")
		}
		return map[string]any{
				"houseId":    requestNumberOrString(acceptedHouseID),
				"shareId":    requestNumberOrString(shareID),
				"createTime": createTime,
				"toUid":      requestNumberOrString(currentUID),
			}, []string{
				"Runtime 使用当前本地登录账号作为接受人，不接受模型传入 toUid",
				"分享码必须提供结构化 shareId 与 createTime",
				"plan.commit 只接受 planId，忽略提交时附带的分享字段",
				"提交后通过 home.summary 验证新家庭在当前账号下可见",
			}, "接受家庭分享", plan.RiskR2, "", nil
	case "home.member.configure":
		memberID := firstValueIDString(request.Parameters, "memberId", "uid", "userId", "id")
		userRole, ok := normalizedHomeMemberRole(firstNonNil(request.Parameters["userRole"], request.Parameters["role"]))
		if memberID == "" || !ok || userRole == 1 || !homeMemberExists(members, memberID) {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_configure_payload")
		}
		return map[string]any{
				"houseId":  requestNumberOrString(houseID),
				"uid":      requestNumberOrString(currentUID),
				"memberId": requestNumberOrString(memberID),
				"userRole": userRole,
			}, []string{
				"提交前重新读取家庭成员列表",
				"仅允许在普通成员与管理员之间切换角色",
				"plan.commit 只接受 planId，忽略提交时附带的角色字段",
				"提交后通过 home.member.list 验证目标成员角色",
			}, "更新家庭成员角色", plan.RiskR2, "", nil
	case "home.member.remove":
		memberID := firstValueIDString(request.Parameters, "memberId", "uid", "userId", "id")
		if memberID == "" || !homeMemberExists(members, memberID) || homeMemberIsMaster(members, memberID) {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_remove_payload")
		}
		challenge := "REMOVE home.member " + memberID
		return map[string]any{
			"houseId":  requestNumberOrString(houseID),
			"memberId": requestNumberOrString(memberID),
		}, homeMemberR3Preconditions("移除家庭成员"), "移除家庭成员", plan.RiskR3, challenge, nil
	case "home.member.transfer":
		memberID := firstValueIDString(request.Parameters, "memberId", "uid", "userId", "id")
		if memberID == "" || !homeMemberExists(members, memberID) || homeMemberIsMaster(members, memberID) {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_transfer_payload")
		}
		challenge := "TRANSFER home.member " + memberID
		return map[string]any{
			"houseId":  requestNumberOrString(houseID),
			"memberId": requestNumberOrString(memberID),
		}, homeMemberR3Preconditions("转移家庭管理员权限"), "转移家庭管理员权限", plan.RiskR3, challenge, nil
	case "home.member.quit":
		uid := firstValueIDString(request.Parameters, "uid", "memberId", "userId", "id")
		if uid == "" {
			uid = currentUID
		}
		if uid == "" || !homeMemberExists(members, uid) || homeMemberIsMaster(members, uid) {
			return nil, nil, "", "", "", fmt.Errorf("invalid_home_member_quit_payload")
		}
		challenge := "QUIT home.member " + uid
		return map[string]any{
			"houseId": requestNumberOrString(houseID),
			"uid":     requestNumberOrString(uid),
		}, homeMemberR3Preconditions("退出分享家庭"), "退出分享家庭", plan.RiskR3, challenge, nil
	default:
		return nil, nil, "", "", "", fmt.Errorf("unsupported_home_member_intent")
	}
}

func homeMemberR3Preconditions(action string) []string {
	return []string{
		"这是 R3 高影响成员操作计划，普通 plan.commit 会被阻断",
		"必须先在本机终端运行 approveCommand 完成一次性审批",
		"plan.commit 只接受 planId，忽略提交时附带的成员字段",
		"提交前 Runtime 会重新读取成员并确认目标仍属于当前家庭",
		"提交后 Runtime 会通过 home.member.list 验证结果",
		action + "可能影响家庭访问权限，需要用户明确确认",
	}
}

func normalizedHomeMemberRole(value any) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(requestString(value))) {
	case "0", "normal", "member", "普通用户":
		return 0, true
	case "2", "admin", "管理员":
		return 2, true
	default:
		return 0, false
	}
}

func requestBoolDefault(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func homeMemberPreview(intent string, payload map[string]any, members []map[string]any) map[string]any {
	preview := map[string]any{"planned": pendingPlanPayloadPreview(plan.Record{HouseID: requestString(payload["houseId"]), Payload: payload})}
	memberID := firstNonEmptyString(valueIDString(payload["memberId"]), valueIDString(payload["uid"]))
	if memberID != "" {
		if member, ok := findHomeMember(members, memberID); ok {
			preview["targetMember"] = map[string]any{
				"memberIdMasked": maskLocalIdentifier(memberID),
				"displayName":    firstAnyMemberString(member, "nickname", "nickName", "name", "displayName", "remark"),
				"role":           firstAnyMemberString(member, "role", "userRole", "memberRole"),
			}
		}
	}
	if intent == "home.member.transfer" || intent == "home.member.remove" || intent == "home.member.quit" {
		preview["impact"] = map[string]any{
			"mode":                 "r3_home_member_mutation",
			"requiresLocalApprove": true,
		}
	}
	return preview
}

func homeMemberAcceptedFields(intent string) []string {
	switch intent {
	case "home.member.invite":
		return []string{"parameters.houseId", "parameters.expiredTime", "parameters.userRole", "parameters.reuseBarcode"}
	case "home.member.accept_share":
		return []string{"parameters.houseId", "parameters.shareId", "parameters.createTime"}
	case "home.member.configure":
		return []string{"parameters.houseId", "parameters.memberId", "parameters.userRole"}
	case "home.member.remove", "home.member.transfer":
		return []string{"parameters.houseId", "parameters.memberId"}
	case "home.member.quit":
		return []string{"parameters.houseId", "parameters.uid"}
	default:
		return []string{"parameters.houseId"}
	}
}

func (app *app) commitHomeMemberPlan(ctx context.Context, request contract.Request, endpoint api.Endpoint, record plan.Record, authorization string, clientID string, kind api.HomeMemberKind) (contract.Response, error) {
	result, err := api.NewHomeMemberClient(endpoint, nil).Run(ctx, api.HomeMemberRequest{
		Kind:           kind,
		HouseID:        record.HouseID,
		Payload:        record.Payload,
		VerifyAttempts: 5,
		VerifyInterval: time.Second,
		Credentials: api.HomeMemberCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	if _, err := app.planStore.MarkCommitted(record.ID); err != nil {
		return contract.Response{}, err
	}
	return homeMemberCommitResponse(request, record, result), nil
}

func homeMemberExists(members []map[string]any, uid string) bool {
	_, ok := findHomeMember(members, uid)
	return ok
}

func homeMemberIsMaster(members []map[string]any, uid string) bool {
	member, ok := findHomeMember(members, uid)
	if !ok {
		return false
	}
	return firstAnyMemberString(member, "role", "userRole", "memberRole") == "1"
}

func findHomeMember(members []map[string]any, uid string) (map[string]any, bool) {
	for _, member := range members {
		if firstAnyMemberString(member, "uid", "userId", "memberId", "id") == uid {
			return member, true
		}
	}
	return nil, false
}

func firstAnyMemberString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(requestString(values[key])); value != "" {
			return value
		}
	}
	return ""
}

func maskLocalIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 6 {
		return "***"
	}
	return trimmed[:3] + "***" + trimmed[len(trimmed)-3:]
}
