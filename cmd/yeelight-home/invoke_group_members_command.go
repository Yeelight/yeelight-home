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

func (app *app) prepareGroupMembersUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, profile string, region string, houseID string, authorization string, clientID string) (contract.Response, error) {
	if requestHouseID := requestHouseID(request); requestHouseID != "" {
		houseID = requestHouseID
	}
	if strings.TrimSpace(houseID) == "" {
		return groupMembersClarificationResponse(request, "missing_house_id"), nil
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
	payload, preconditions, summary, preview, extraAPICalls, reason := buildGroupMembersUpdatePayload(ctx, request, endpoint, houseID, authorization, clientID, entities)
	if reason != "" {
		return groupMembersClarificationResponse(request, reason), nil
	}
	record, err := operation.NewPrepared(profile, region, houseID, request.Intent, request.RequestID, summary, payload, preconditions, time.Now())
	if err != nil {
		return contract.Response{}, err
	}
	app.preparedOperation = &record
	return executionPreviewResponseWithDetails(request, record, entities, preview, extraAPICalls), nil
}

func buildGroupMembersUpdatePayload(ctx context.Context, request contract.Request, endpoint api.Endpoint, houseID string, authorization string, clientID string, entities api.EntityListResult) (map[string]any, []string, string, map[string]any, int, string) {
	groupID, reason := resolveGroupMemberTarget(request, entities)
	if reason != "" {
		return nil, nil, "", nil, 0, reason
	}
	addIDs, removeIDs, desiredIDs, reason := resolveGroupMemberDevices(request.Parameters, entities)
	if reason != "" {
		return nil, nil, "", nil, 0, reason
	}
	snapshot, detailCalls, err := api.NewGroupMembersClient(endpoint, nil).Snapshot(ctx, houseID, groupID, api.SpaceOrganizationCredentials{
		Authorization: authorization,
		ClientID:      clientID,
	})
	if err != nil {
		return nil, nil, "", nil, detailCalls, "group_detail_unavailable"
	}
	if len(desiredIDs) > 0 {
		addIDs, removeIDs = diffGroupMemberIDs(snapshot.DeviceIDs, desiredIDs)
	}
	if duplicated := intersectStringIDs(addIDs, removeIDs); len(duplicated) > 0 {
		return nil, nil, "", nil, detailCalls, "group_member_delta_conflict"
	}
	if len(addIDs) == 0 && len(removeIDs) == 0 && !hasGroupMemberSelection(request.Parameters) {
		return nil, nil, "", nil, detailCalls, "invalid_group_members_payload"
	}
	finalIDs := applyGroupMemberDelta(snapshot.DeviceIDs, addIDs, removeIDs)
	if len(finalIDs) > groupDeviceLimit {
		return nil, nil, "", nil, detailCalls, "group_member_limit_exceeded"
	}
	payload := map[string]any{
		semantic.FieldHouseID:         requestNumberOrString(houseID),
		semantic.FieldGroupID:         groupID,
		semantic.FieldAddDeviceIDs:    addIDs,
		semantic.FieldRemoveDeviceIDs: removeIDs,
	}
	if len(desiredIDs) > 0 {
		payload[semantic.FieldDeviceIDs] = desiredIDs
	}
	preview := map[string]any{
		semantic.FieldGroupID:            groupID,
		semantic.FieldName:               snapshot.Name,
		semantic.FieldCurrentItems:       snapshot.DeviceIDs,
		semantic.FieldAddDeviceIDs:       addIDs,
		semantic.FieldRemoveDeviceIDs:    removeIDs,
		semantic.FieldDeviceIDs:          finalIDs,
		semantic.FieldDeviceCount:        len(finalIDs),
		semantic.FieldCurrentRoomID:      snapshot.RoomID,
		semantic.FieldProductComponentID: snapshot.ComponentID,
	}
	return payload, []string{
		"提交前读取家庭实体和设备组详情",
		"新增和移除的设备必须属于当前家庭",
		"提交后通过设备组详情验证成员列表",
	}, fmt.Sprintf("更新设备组 %s 的成员", firstNonEmptyString(snapshot.Name, groupID)), preview, detailCalls, ""
}

func resolveGroupMemberTarget(request contract.Request, entities api.EntityListResult) (string, string) {
	groupID := firstRequestString(request.Parameters, semantic.FieldGroupID, semantic.FieldID, semantic.FieldEntityID)
	if groupID != "" {
		if !entityExists(entities, "group", groupID) {
			return "", "invalid_group_reference"
		}
		return groupID, ""
	}
	name := firstRequestString(request.Parameters, semantic.FieldGroupName, semantic.FieldCurrentName, semantic.FieldEntityName, semantic.FieldTargetName, semantic.FieldName)
	if name == "" {
		return "", "invalid_group_reference"
	}
	match, ambiguous := findUniqueEntityByName(entities, "group", name)
	if ambiguous {
		return "", "ambiguous_group_reference"
	}
	if match.ID == "" {
		return "", "invalid_group_reference"
	}
	return match.ID, ""
}

func resolveGroupMemberDevices(parameters map[string]any, entities api.EntityListResult) ([]string, []string, []string, string) {
	desiredIDs, reason := resolveDeviceReferenceList(entities, valueIDList(parameters[semantic.FieldDeviceIDs]), requestStringList(parameters[semantic.FieldDeviceNames]))
	if reason != "" {
		return nil, nil, nil, reason
	}
	addIDs, reason := resolveDeviceReferenceList(entities, append(valueIDList(parameters[semantic.FieldAddDeviceIDs]), valueIDList(parameters[semantic.FieldAddDeviceList])...), requestStringList(parameters[semantic.FieldAddDeviceNames]))
	if reason != "" {
		return nil, nil, nil, reason
	}
	removeIDs, reason := resolveDeviceReferenceList(entities, append(valueIDList(parameters[semantic.FieldRemoveDeviceIDs]), valueIDList(parameters[semantic.FieldRemoveDeviceList])...), requestStringList(parameters[semantic.FieldRemoveDeviceNames]))
	if reason != "" {
		return nil, nil, nil, reason
	}
	return addIDs, removeIDs, desiredIDs, ""
}

func resolveDeviceReferenceList(entities api.EntityListResult, ids []string, names []string) ([]string, string) {
	result := append([]string{}, ids...)
	for _, name := range names {
		match, ambiguous := findUniqueEntityByName(entities, "device", name)
		if ambiguous {
			return nil, "ambiguous_device_reference"
		}
		if match.ID == "" {
			return nil, "invalid_device_reference"
		}
		result = append(result, match.ID)
	}
	result = uniqueStringIDs(result)
	for _, id := range result {
		if !entityExists(entities, "device", id) {
			return nil, "invalid_device_reference"
		}
	}
	return result, ""
}

func hasGroupMemberSelection(parameters map[string]any) bool {
	for _, key := range []string{
		semantic.FieldDeviceIDs,
		semantic.FieldDeviceNames,
		semantic.FieldAddDeviceIDs,
		semantic.FieldAddDeviceNames,
		semantic.FieldAddDeviceList,
		semantic.FieldRemoveDeviceIDs,
		semantic.FieldRemoveDeviceNames,
		semantic.FieldRemoveDeviceList,
	} {
		if _, ok := parameters[key]; ok {
			return true
		}
	}
	return false
}

func diffGroupMemberIDs(currentIDs []string, desiredIDs []string) ([]string, []string) {
	current := stringIDSet(currentIDs)
	desired := stringIDSet(desiredIDs)
	addIDs := []string{}
	removeIDs := []string{}
	for id := range desired {
		if !current[id] {
			addIDs = append(addIDs, id)
		}
	}
	for id := range current {
		if !desired[id] {
			removeIDs = append(removeIDs, id)
		}
	}
	return uniqueStringIDs(addIDs), uniqueStringIDs(removeIDs)
}

func applyGroupMemberDelta(currentIDs []string, addIDs []string, removeIDs []string) []string {
	set := stringIDSet(currentIDs)
	for _, id := range addIDs {
		set[id] = true
	}
	for _, id := range removeIDs {
		delete(set, id)
	}
	result := make([]string, 0, len(set))
	for id := range set {
		result = append(result, id)
	}
	return uniqueStringIDs(result)
}

func uniqueStringIDs(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

func intersectStringIDs(left []string, right []string) []string {
	rightSet := stringIDSet(right)
	result := []string{}
	for _, id := range left {
		if rightSet[id] {
			result = append(result, id)
		}
	}
	return result
}

func stringIDSet(ids []string) map[string]bool {
	result := map[string]bool{}
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			result[trimmed] = true
		}
	}
	return result
}

func groupMembersClarificationResponse(request contract.Request, reason string) contract.Response {
	return configureClarificationResponseWithGuide(request, reason, groupMembersAcceptedFields(), payloadGuideForIntent(request.Intent))
}

func groupMembersAcceptedFields() []string {
	return semanticParameterPaths(
		semantic.FieldHouseID,
		semantic.FieldGroupID,
		semantic.FieldGroupName,
		semantic.FieldDeviceIDs,
		semantic.FieldDeviceNames,
		semantic.FieldAddDeviceIDs,
		semantic.FieldAddDeviceNames,
		semantic.FieldRemoveDeviceIDs,
		semantic.FieldRemoveDeviceNames,
	)
}

func (app *app) executeGroupMembersUpdate(ctx context.Context, request contract.Request, endpoint api.Endpoint, record operation.Prepared, authorization string, clientID string) (contract.Response, error) {
	result, err := api.NewGroupMembersClient(endpoint, nil).Run(ctx, api.GroupMembersRequest{
		HouseID:         record.HouseID,
		GroupID:         valueIDString(record.Payload[semantic.FieldGroupID]),
		AddDeviceIDs:    valueIDList(record.Payload[semantic.FieldAddDeviceIDs]),
		RemoveDeviceIDs: valueIDList(record.Payload[semantic.FieldRemoveDeviceIDs]),
		VerifyAttempts:  5,
		VerifyInterval:  time.Second,
		Credentials: api.SpaceOrganizationCredentials{
			Authorization: authorization,
			ClientID:      clientID,
		},
	})
	if err != nil {
		return contract.Response{}, err
	}
	return groupMembersExecuteResponse(request, record, result), nil
}
