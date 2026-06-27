package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HomeMemberKind string

const (
	HomeMemberInvite    HomeMemberKind = "home.member.invite"
	HomeMemberAccept    HomeMemberKind = "home.member.accept_share"
	HomeMemberConfigure HomeMemberKind = "home.member.configure"
	HomeMemberRemove    HomeMemberKind = "home.member.remove"
	HomeMemberTransfer  HomeMemberKind = "home.member.transfer"
	HomeMemberQuit      HomeMemberKind = "home.member.quit"
)

type HomeMemberCredentials struct {
	Authorization string
	ClientID      string
}

type HomeMemberRequest struct {
	Kind           HomeMemberKind
	HouseID        string
	Payload        map[string]any
	VerifyAttempts int
	VerifyInterval time.Duration
	Credentials    HomeMemberCredentials
}

type HomeMemberResult struct {
	Region     string `json:"region"`
	HouseID    string `json:"houseId"`
	Capability string `json:"capability"`
	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
	APICalls   int    `json:"apiCalls"`
	Data       any    `json:"data,omitempty"`
}

type HomeMemberClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewHomeMemberClient(endpoint Endpoint, client *http.Client) HomeMemberClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return HomeMemberClient{endpoint: endpoint, client: client}
}

func (client HomeMemberClient) Run(ctx context.Context, request HomeMemberRequest) (HomeMemberResult, error) {
	houseID := strings.TrimSpace(request.HouseID)
	if houseID == "" {
		return HomeMemberResult{}, fmt.Errorf("house id is required")
	}
	credentials := requestCredentials{Authorization: request.Credentials.Authorization, ClientID: request.Credentials.ClientID, HouseID: houseID}
	if strings.TrimSpace(credentials.Authorization) == "" {
		return HomeMemberResult{}, fmt.Errorf("missing token; run auth login --qr or set YEELIGHT_HOME_ACCESS_TOKEN")
	}
	apiCalls := 0
	calls, err := client.preflight(ctx, request.Kind, houseID, request.Payload, credentials)
	apiCalls += calls
	if err != nil {
		return HomeMemberResult{}, err
	}
	data, calls, err := client.write(ctx, request.Kind, houseID, request.Payload, credentials)
	apiCalls += calls
	if err != nil {
		return HomeMemberResult{}, err
	}
	ok, calls, err := client.verifyAfterWrite(ctx, request.Kind, houseID, request.Payload, credentials, request.VerifyAttempts, request.VerifyInterval)
	apiCalls += calls
	if err != nil {
		return HomeMemberResult{}, err
	}
	if !ok {
		return HomeMemberResult{}, fmt.Errorf("%s write verification mismatch", request.Kind)
	}
	return HomeMemberResult{
		Region:     client.endpoint.Region,
		HouseID:    houseID,
		Capability: string(request.Kind),
		Verified:   true,
		VerifiedBy: homeMemberVerifyWith(request.Kind),
		APICalls:   apiCalls,
		Data:       data,
	}, nil
}

func (client HomeMemberClient) preflight(ctx context.Context, kind HomeMemberKind, houseID string, payload map[string]any, credentials requestCredentials) (int, error) {
	switch kind {
	case HomeMemberInvite:
		_, calls, err := client.readMembers(ctx, houseID, credentials)
		return calls, err
	case HomeMemberAccept:
		if strings.TrimSpace(stringFromAny(payload["shareId"])) == "" {
			return 0, fmt.Errorf("share id is required")
		}
		if strings.TrimSpace(stringFromAny(payload["createTime"])) == "" {
			return 0, fmt.Errorf("share createTime is required")
		}
		if strings.TrimSpace(stringFromAny(payload["toUid"])) == "" {
			return 0, fmt.Errorf("recipient uid is required")
		}
		return 0, nil
	case HomeMemberConfigure:
		memberID := strings.TrimSpace(stringFromAny(payload["memberId"]))
		if memberID == "" {
			return 0, fmt.Errorf("member id is required")
		}
		if !validConfigurableHomeRole(payload["userRole"]) {
			return 0, fmt.Errorf("home member userRole must be 0 or 2")
		}
		if strings.TrimSpace(stringFromAny(payload["uid"])) == "" {
			return 0, fmt.Errorf("operator uid is required")
		}
		return client.requireMember(ctx, houseID, memberID, credentials)
	case HomeMemberRemove, HomeMemberTransfer:
		memberID := strings.TrimSpace(stringFromAny(payload["memberId"]))
		if memberID == "" {
			return 0, fmt.Errorf("member id is required")
		}
		return client.requireMutableMember(ctx, houseID, memberID, credentials)
	case HomeMemberQuit:
		uid := strings.TrimSpace(stringFromAny(payload["uid"]))
		if uid == "" {
			return 0, fmt.Errorf("member uid is required")
		}
		return client.requireMutableMember(ctx, houseID, uid, credentials)
	default:
		return 0, fmt.Errorf("unsupported home member kind %q", kind)
	}
}

func (client HomeMemberClient) write(ctx context.Context, kind HomeMemberKind, houseID string, payload map[string]any, credentials requestCredentials) (any, int, error) {
	switch kind {
	case HomeMemberInvite:
		body := mapWithoutKeys(payload, "capability")
		body["houseId"] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/share/r/housesharebarcode", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.invite returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	case HomeMemberAccept:
		body := map[string]any{
			"shareId":    payload["shareId"],
			"createTime": payload["createTime"],
			"toUid":      payload["toUid"],
		}
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/share/w/acceptbarcodeshare", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.accept_share returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	case HomeMemberConfigure:
		body := mapWithoutKeys(payload, "capability")
		body["houseId"] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/w/updateUserRole", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.configure returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	case HomeMemberRemove:
		body := mapWithoutKeys(payload, "capability")
		body["houseId"] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/w/remove", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.remove returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	case HomeMemberTransfer:
		body := mapWithoutKeys(payload, "capability")
		body["houseId"] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/w/transfer", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.transfer returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	case HomeMemberQuit:
		body := mapWithoutKeys(payload, "capability")
		body["houseId"] = requestNumberOrStringForAPI(houseID)
		response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/w/quit", body, credentials)
		if err != nil {
			return nil, 1, err
		}
		if !isBusinessOK(response) {
			return nil, 1, fmt.Errorf("home.member.quit returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
		}
		return sanitizeCloudData(response["data"]), 1, nil
	default:
		return nil, 0, fmt.Errorf("unsupported home member kind %q", kind)
	}
}

func (client HomeMemberClient) verifyAfterWrite(ctx context.Context, kind HomeMemberKind, houseID string, payload map[string]any, credentials requestCredentials, attempts int, interval time.Duration) (bool, int, error) {
	if attempts <= 0 {
		attempts = 3
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	calls := 0
	for attempt := 0; attempt < attempts; attempt++ {
		var ok bool
		var readCalls int
		var err error
		switch kind {
		case HomeMemberInvite:
			ok, readCalls, err = client.verifyInviteResult(ctx, houseID, payload, credentials)
		case HomeMemberAccept:
			ok, readCalls, err = client.verifyAcceptResult(ctx, houseID, payload, credentials)
		case HomeMemberConfigure:
			ok, readCalls, err = client.verifyMemberRole(ctx, houseID, strings.TrimSpace(stringFromAny(payload["memberId"])), payload["userRole"], credentials)
		case HomeMemberTransfer:
			ok, readCalls, err = client.verifyMemberRole(ctx, houseID, strings.TrimSpace(stringFromAny(payload["memberId"])), float64(1), credentials)
		case HomeMemberRemove:
			ok, readCalls, err = client.verifyMemberMissing(ctx, houseID, strings.TrimSpace(stringFromAny(payload["memberId"])), credentials)
		case HomeMemberQuit:
			ok, readCalls, err = client.verifyMemberMissing(ctx, houseID, strings.TrimSpace(stringFromAny(payload["uid"])), credentials)
		default:
			return false, calls, fmt.Errorf("unsupported home member kind %q", kind)
		}
		calls += readCalls
		if err != nil || ok || attempt == attempts-1 {
			return ok, calls, err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false, calls, ctx.Err()
		case <-timer.C:
		}
	}
	return false, calls, nil
}

func (client HomeMemberClient) readMembers(ctx context.Context, houseID string, credentials requestCredentials) ([]map[string]any, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodPost, strings.TrimRight(client.endpoint.BaseURL, "/")+"/v1/house/r/memberlistV2", map[string]any{"houseId": requestNumberOrStringForAPI(houseID)}, credentials)
	if err != nil {
		return nil, 1, err
	}
	if !isBusinessOK(response) {
		return nil, 1, fmt.Errorf("home.member.list returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	rows := rowsFromData(response["data"])
	members := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if item, ok := row.(map[string]any); ok {
			members = append(members, item)
		}
	}
	return members, 1, nil
}

func (client HomeMemberClient) requireMember(ctx context.Context, houseID string, uid string, credentials requestCredentials) (int, error) {
	members, calls, err := client.readMembers(ctx, houseID, credentials)
	if err != nil {
		return calls, err
	}
	for _, member := range members {
		if homeMemberUID(member) == uid {
			return calls, nil
		}
	}
	return calls, fmt.Errorf("member %s not found before write", uid)
}

func (client HomeMemberClient) requireMutableMember(ctx context.Context, houseID string, uid string, credentials requestCredentials) (int, error) {
	members, calls, err := client.readMembers(ctx, houseID, credentials)
	if err != nil {
		return calls, err
	}
	for _, member := range members {
		if homeMemberUID(member) == uid {
			if homeMemberRole(member) == "1" {
				return calls, fmt.Errorf("home master member cannot be removed, transferred to itself, or quit through this intent")
			}
			return calls, nil
		}
	}
	return calls, fmt.Errorf("member %s not found before write", uid)
}

func (client HomeMemberClient) verifyMemberMissing(ctx context.Context, houseID string, uid string, credentials requestCredentials) (bool, int, error) {
	members, calls, err := client.readMembers(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	for _, member := range members {
		if homeMemberUID(member) == uid {
			return false, calls, nil
		}
	}
	return true, calls, nil
}

func (client HomeMemberClient) verifyMemberRole(ctx context.Context, houseID string, uid string, expectedRole any, credentials requestCredentials) (bool, int, error) {
	members, calls, err := client.readMembers(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	expected := strings.TrimSpace(stringFromAny(expectedRole))
	for _, member := range members {
		if homeMemberUID(member) == uid {
			return homeMemberRole(member) == expected, calls, nil
		}
	}
	return false, calls, nil
}

func (client HomeMemberClient) verifyInviteResult(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	_, calls, err := client.readMembers(ctx, houseID, credentials)
	if err != nil {
		return false, calls, err
	}
	return strings.TrimSpace(stringFromAny(payload["expiredTime"])) != "", calls, nil
}

func (client HomeMemberClient) verifyAcceptResult(ctx context.Context, houseID string, payload map[string]any, credentials requestCredentials) (bool, int, error) {
	summary, err := NewHomeSummaryClient(client.endpoint, client.client).Run(ctx, HomeSummaryCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
	if err != nil {
		return false, 1, err
	}
	for _, house := range summary.Houses {
		if strings.TrimSpace(house.ID) == houseID {
			return true, 1, nil
		}
	}
	return false, 1, nil
}

func (client HomeMemberClient) CurrentUserID(ctx context.Context, credentials HomeMemberCredentials) (string, int, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, client.endpoint.AccountBaseURL()+"/apis/account/user/info", nil, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
	if err != nil {
		return "", 1, err
	}
	if !isBusinessOK(response) {
		return "", 1, fmt.Errorf("account.info returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	data, _ := response["data"].(map[string]any)
	uid := firstAnyString(data, "uid", "userId", "id", "accountId")
	if uid == "" {
		return "", 1, fmt.Errorf("account.info did not include a user id")
	}
	return uid, 1, nil
}

func (client HomeMemberClient) ProbeMembers(ctx context.Context, houseID string, credentials HomeMemberCredentials) ([]map[string]any, int, error) {
	return client.readMembers(ctx, houseID, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
}

func homeMemberUID(member map[string]any) string {
	return firstAnyString(member, "uid", "userId", "memberId", "id")
}

func homeMemberRole(member map[string]any) string {
	return firstAnyString(member, "role", "userRole", "memberRole")
}

func validConfigurableHomeRole(value any) bool {
	role := strings.TrimSpace(stringFromAny(value))
	return role == "0" || role == "2"
}

func homeMemberVerifyWith(kind HomeMemberKind) string {
	switch kind {
	case HomeMemberInvite:
		return "home.member.list"
	case HomeMemberAccept:
		return "home.summary"
	case HomeMemberConfigure:
		return "home.member.list"
	case HomeMemberRemove:
		return "home.member.list"
	case HomeMemberTransfer:
		return "home.member.list"
	case HomeMemberQuit:
		return "home.member.list"
	default:
		return "write_after_read"
	}
}
