package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type GroupMembersRequest struct {
	HouseID         string
	GroupID         string
	AddDeviceIDs    []string
	RemoveDeviceIDs []string
	VerifyAttempts  int
	VerifyInterval  time.Duration
	Credentials     SpaceOrganizationCredentials
}

type GroupMembersResult struct {
	Region           string           `json:"region"`
	HouseID          string           `json:"houseId"`
	Capability       string           `json:"capability"`
	GroupID          string           `json:"groupId"`
	Name             string           `json:"name,omitempty"`
	AddedDeviceIDs   []string         `json:"addDeviceIds,omitempty"`
	RemovedDeviceIDs []string         `json:"removeDeviceIds,omitempty"`
	CurrentDeviceIDs []string         `json:"deviceIds,omitempty"`
	Verified         bool             `json:"verified"`
	VerifiedBy       string           `json:"verifiedBy,omitempty"`
	APICalls         int              `json:"apiCalls"`
	Warnings         []string         `json:"warnings,omitempty"`
	VerifiedEntities EntityListResult `json:"-"`
}

type GroupMembersSnapshot struct {
	GroupID     string         `json:"groupId"`
	Name        string         `json:"name,omitempty"`
	RoomID      string         `json:"roomId,omitempty"`
	ComponentID string         `json:"componentId,omitempty"`
	DeviceIDs   []string       `json:"deviceIds,omitempty"`
	Detail      map[string]any `json:"detail,omitempty"`
}

type GroupMembersClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewGroupMembersClient(endpoint Endpoint, client *http.Client) GroupMembersClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return GroupMembersClient{endpoint: endpoint, client: client}
}

func (client GroupMembersClient) Snapshot(ctx context.Context, houseID string, groupID string, credentials SpaceOrganizationCredentials) (GroupMembersSnapshot, int, error) {
	houseID = strings.TrimSpace(houseID)
	groupID = strings.TrimSpace(groupID)
	if houseID == "" || groupID == "" {
		return GroupMembersSnapshot{}, 0, fmt.Errorf("house id and group id are required")
	}
	requestCredentials := requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
		HouseID:       houseID,
	}
	response, err := callJSON(ctx, client.client, http.MethodGet, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/group/"+pathSegment(groupID)+"/r/info", nil, requestCredentials)
	if err != nil {
		return GroupMembersSnapshot{}, 1, err
	}
	if !isBusinessOK(response) {
		return GroupMembersSnapshot{}, 1, metadataReadonlyBusinessError("group detail", response)
	}
	return groupMembersSnapshotFromData(response["data"], groupID), 1, nil
}

func (client GroupMembersClient) Run(ctx context.Context, request GroupMembersRequest) (GroupMembersResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	groupID := strings.TrimSpace(request.GroupID)
	if houseID == "" || groupID == "" {
		return GroupMembersResult{}, fmt.Errorf("house id and group id are required")
	}
	addIDs := uniqueNonEmptyIDs(request.AddDeviceIDs)
	removeIDs := uniqueNonEmptyIDs(request.RemoveDeviceIDs)
	if duplicated := intersectIDs(addIDs, removeIDs); len(duplicated) > 0 {
		return GroupMembersResult{}, fmt.Errorf("device ids cannot be both added and removed: %s", strings.Join(duplicated, ","))
	}
	credentials := SpaceOrganizationCredentials{
		Authorization: request.Credentials.Authorization,
		ClientID:      request.Credentials.ClientID,
	}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return GroupMembersResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	warnings := []string{}
	before, calls, err := client.Snapshot(ctx, houseID, groupID, credentials)
	apiCalls += calls
	if err != nil {
		return GroupMembersResult{}, err
	}
	compatWarnings, compatCalls, err := client.validateAddedDevices(ctx, houseID, before.ComponentID, addIDs, credentials)
	apiCalls += compatCalls
	warnings = append(warnings, compatWarnings...)
	if err != nil {
		return GroupMembersResult{}, err
	}
	if len(addIDs) > 0 || len(removeIDs) > 0 {
		if err := client.writeMembers(ctx, houseID, groupID, addIDs, removeIDs, credentials); err != nil {
			return GroupMembersResult{}, err
		}
		apiCalls++
	}
	verified, verifyCalls, err := client.verifyMembers(ctx, houseID, groupID, addIDs, removeIDs, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += verifyCalls
	if err != nil {
		return GroupMembersResult{}, err
	}
	if !groupMembersContainDelta(verified.DeviceIDs, addIDs, removeIDs) {
		return GroupMembersResult{}, fmt.Errorf("group.members.update write verification mismatch")
	}
	return GroupMembersResult{
		Region:           client.endpoint.Region,
		HouseID:          houseID,
		Capability:       "group.members.update",
		GroupID:          groupID,
		Name:             firstNonEmpty(verified.Name, before.Name),
		AddedDeviceIDs:   addIDs,
		RemovedDeviceIDs: removeIDs,
		CurrentDeviceIDs: verified.DeviceIDs,
		Verified:         true,
		VerifiedBy:       "group.detail.get",
		APICalls:         apiCalls,
		Warnings:         warnings,
	}, nil
}

func (client GroupMembersClient) writeMembers(ctx context.Context, houseID string, groupID string, addIDs []string, removeIDs []string, credentials SpaceOrganizationCredentials) error {
	body := map[string]any{
		semantic.FieldHouseID:          requestNumberOrStringForAPI(houseID),
		semantic.FieldGroupID:          requestNumberOrStringForAPI(groupID),
		semantic.FieldAddDeviceList:    apiIDList(addIDs),
		semantic.FieldRemoveDeviceList: apiIDList(removeIDs),
	}
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v2/thing/manage/house/"+pathSegment(houseID)+"/group/"+pathSegment(groupID)+"/w/devices", body, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
		HouseID:       houseID,
	})
	if err != nil {
		return err
	}
	if !isBusinessOK(response) {
		return fmt.Errorf("group.members.update returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return nil
}

func (client GroupMembersClient) verifyMembers(ctx context.Context, houseID string, groupID string, addIDs []string, removeIDs []string, credentials SpaceOrganizationCredentials, attempts int, interval time.Duration) (GroupMembersSnapshot, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		snapshot, readCalls, err := client.Snapshot(ctx, houseID, groupID, credentials)
		calls += readCalls
		if err != nil || groupMembersContainDelta(snapshot.DeviceIDs, addIDs, removeIDs) || attempt == attempts-1 {
			return snapshot, calls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return GroupMembersSnapshot{}, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return GroupMembersSnapshot{}, calls, nil
}

func (client GroupMembersClient) validateAddedDevices(ctx context.Context, houseID string, componentID string, addIDs []string, credentials SpaceOrganizationCredentials) ([]string, int, error) {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" || len(addIDs) == 0 {
		return nil, 0, nil
	}
	warnings := []string{}
	calls := 0
	for _, deviceID := range addIDs {
		capability, err := NewDeviceCapabilitiesClient(client.endpoint, client.client).Run(ctx, DeviceCapabilitiesRequest{
			HouseID:  houseID,
			DeviceID: deviceID,
			Credentials: DeviceCapabilitiesCredentials{
				Authorization: credentials.Authorization,
				ClientID:      credentials.ClientID,
			},
		})
		calls++
		if err != nil {
			warnings = append(warnings, "group_member_schema_unavailable")
			continue
		}
		if !deviceSupportsComponent(capability.Device, componentID) {
			return warnings, calls, fmt.Errorf("device %s does not support group component %s", deviceID, componentID)
		}
	}
	return warnings, calls, nil
}

func groupMembersSnapshotFromData(data any, fallbackGroupID string) GroupMembersSnapshot {
	item := groupDetailItem(data)
	detail := projectGroupDetail(data, fallbackGroupID)
	return GroupMembersSnapshot{
		GroupID:     firstNonEmpty(firstAnyString(item, semantic.FieldID, semantic.FieldGroupID, semantic.MeshGroupIDField(), semantic.FieldMeshGroupID), fallbackGroupID),
		Name:        firstNonEmpty(firstAnyString(item, semantic.FieldName, semantic.FieldGroupName), stringFromAny(detail[semantic.FieldName])),
		RoomID:      firstNonEmpty(firstAnyString(item, semantic.FieldRoomID), stringFromAny(detail[semantic.FieldRoomID])),
		ComponentID: firstAnyString(item, "cid", semantic.InternalCloudComponentIDField(), semantic.FieldComponentID, semantic.FieldProductComponentID),
		DeviceIDs:   groupMemberDeviceIDs(detail),
		Detail:      detail,
	}
}

func groupMemberDeviceIDs(detail map[string]any) []string {
	devices, _ := detail[semantic.FieldDevices].([]any)
	ids := make([]string, 0, len(devices))
	for _, raw := range devices {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if id := firstAnyString(item, semantic.FieldID, semantic.FieldDeviceID); id != "" {
			ids = append(ids, id)
		}
	}
	return uniqueNonEmptyIDs(ids)
}

func groupMembersContainDelta(deviceIDs []string, addIDs []string, removeIDs []string) bool {
	set := idSet(deviceIDs)
	for _, id := range addIDs {
		if !set[id] {
			return false
		}
	}
	for _, id := range removeIDs {
		if set[id] {
			return false
		}
	}
	return true
}

func deviceSupportsComponent(device DeviceCapability, componentID string) bool {
	if strings.TrimSpace(device.ComponentID) == strings.TrimSpace(componentID) {
		return true
	}
	for _, component := range device.Components {
		if strings.TrimSpace(component.ID) == strings.TrimSpace(componentID) {
			return true
		}
	}
	return false
}

func apiIDList(ids []string) []any {
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		result = append(result, requestNumberOrStringForAPI(id))
	}
	return result
}

func uniqueNonEmptyIDs(values []string) []string {
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

func intersectIDs(left []string, right []string) []string {
	rightSet := idSet(right)
	result := []string{}
	for _, id := range left {
		if rightSet[id] {
			result = append(result, id)
		}
	}
	return result
}

func idSet(ids []string) map[string]bool {
	result := map[string]bool{}
	for _, id := range ids {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			result[trimmed] = true
		}
	}
	return result
}
