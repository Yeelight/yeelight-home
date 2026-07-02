package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

type AccountInfoCredentials struct {
	Authorization string
	ClientID      string
}

type AccountInfoResult struct {
	Region   string         `json:"region"`
	Summary  map[string]any `json:"summary"`
	RawShape string         `json:"rawShape"`
	APICalls int            `json:"apiCalls"`
}

type AccountInfoClient struct {
	endpoint Endpoint
	client   *http.Client
}

func NewAccountInfoClient(endpoint Endpoint, client *http.Client) AccountInfoClient {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return AccountInfoClient{endpoint: endpoint, client: client}
}

func (client AccountInfoClient) Run(ctx context.Context, credentials AccountInfoCredentials) (AccountInfoResult, error) {
	response, err := callJSON(ctx, client.client, http.MethodGet, client.endpoint.AccountBaseURL()+"/apis/account/user/info", nil, requestCredentials{
		Authorization: credentials.Authorization,
		ClientID:      credentials.ClientID,
	})
	if err != nil {
		return AccountInfoResult{}, err
	}
	if !isBusinessOK(response) {
		return AccountInfoResult{}, fmt.Errorf("account info returned non-success business response: code=%s message=%s dataType=%s", responseScalar(response, "code"), responseScalar(response, "message", "msg"), responseDataType(response))
	}
	return AccountInfoResult{
		Region:   client.endpoint.Region,
		Summary:  redactedAccountSummary(response["data"]),
		RawShape: responseDataType(response),
		APICalls: 1,
	}, nil
}

func redactedAccountSummary(data any) map[string]any {
	item, ok := data.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	summary := map[string]any{}
	for _, key := range semantic.AccountIDFields() {
		if value := firstAnyString(item, key); value != "" {
			summary[semantic.FieldID] = maskIdentifier(value)
			break
		}
	}
	for _, key := range semantic.AccountDisplayNameFields() {
		if value := firstAnyString(item, key); value != "" {
			summary[semantic.FieldDisplayName] = value
			break
		}
	}
	for _, key := range semantic.AccountPhoneFields() {
		if value := firstAnyString(item, key); value != "" {
			summary[semantic.FieldPhoneMasked] = maskTail(value, 4)
			break
		}
	}
	for _, key := range semantic.AccountEmailFields() {
		if value := firstAnyString(item, key); value != "" {
			summary[semantic.FieldEmailMasked] = maskEmail(value)
			break
		}
	}
	return summary
}

func maskIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 6 {
		return "***"
	}
	return trimmed[:3] + "***" + trimmed[len(trimmed)-3:]
}

func maskTail(value string, tail int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= tail {
		return "***"
	}
	return "***" + trimmed[len(trimmed)-tail:]
}

func maskEmail(value string) string {
	trimmed := strings.TrimSpace(value)
	parts := strings.SplitN(trimmed, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}
