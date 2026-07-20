package lanmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const defaultTimeout = 15 * time.Second

type Client struct {
	endpoint        string
	protocolVersion string
	httpClient      *http.Client
	headers         http.Header
	nextID          atomic.Int64
}

type Options struct {
	ProtocolVersion string
	Timeout         time.Duration
	HTTPClient      *http.Client
	Headers         http.Header
}

func NewClient(endpoint string, options Options) (*Client, error) {
	normalized, err := NormalizeEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if normalized == "" {
		return nil, fmt.Errorf("LAN MCP endpoint is required")
	}
	protocolVersion := strings.TrimSpace(options.ProtocolVersion)
	if protocolVersion == "" {
		protocolVersion = DefaultProtocolVersion
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		httpClient = &http.Client{Timeout: timeout, CheckRedirect: rejectRedirect, Transport: transport}
	} else {
		copied := *httpClient
		if copied.Timeout <= 0 {
			copied.Timeout = timeout
		}
		if copied.CheckRedirect == nil {
			copied.CheckRedirect = rejectRedirect
		}
		httpClient = &copied
	}
	headers := options.Headers.Clone()
	if headers == nil {
		headers = make(http.Header)
	}
	client := &Client{
		endpoint: normalized, protocolVersion: protocolVersion,
		httpClient: httpClient, headers: headers,
	}
	client.nextID.Store(1)
	return client, nil
}

func (client *Client) Initialize(ctx context.Context) (Session, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		response, headers, err := client.request(ctx, "initialize", rpcRequest{
			JSONRPC: "2.0", ID: client.nextRequestID(), Method: "initialize",
			Params: map[string]any{
				"protocolVersion": client.protocolVersion,
				"capabilities":    map[string]any{},
				"clientInfo":      map[string]any{"name": "yeelight-home", "version": "1"},
			},
		}, Session{})
		if err != nil {
			lastErr = err
			var clientErr *ClientError
			if errors.As(err, &clientErr) && clientErr.StatusCode == http.StatusMisdirectedRequest {
				continue
			}
			return Session{}, err
		}
		var result struct {
			ProtocolVersion string         `json:"protocolVersion"`
			ServerInfo      map[string]any `json:"serverInfo"`
		}
		if len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, &result); err != nil {
				return Session{}, protocolError("initialize", "invalid result", err)
			}
		}
		if !supportedProtocolVersion(result.ProtocolVersion) {
			return Session{}, protocolError("initialize", "server selected an unsupported protocol version", nil)
		}
		session := Session{
			ProtocolVersion: firstNonEmpty(result.ProtocolVersion, client.protocolVersion),
			SessionID:       strings.TrimSpace(headers.Get("Mcp-Session-Id")),
			ServerInfo:      result.ServerInfo,
		}
		session.Stateless = session.SessionID == ""
		if !session.Stateless {
			if _, _, err := client.request(ctx, "notifications/initialized", rpcRequest{
				JSONRPC: "2.0", Method: "notifications/initialized", Params: map[string]any{},
			}, session); err != nil {
				return Session{}, err
			}
		}
		return session, nil
	}
	return Session{}, lastErr
}

func (client *Client) ListTools(ctx context.Context, session Session, cursor string) (ListToolsResult, error) {
	params := map[string]any{}
	if strings.TrimSpace(cursor) != "" {
		params["cursor"] = strings.TrimSpace(cursor)
	}
	response, _, err := client.request(ctx, "tools/list", rpcRequest{
		JSONRPC: "2.0", ID: client.nextRequestID(), Method: "tools/list", Params: params,
	}, session)
	if err != nil {
		return ListToolsResult{}, err
	}
	var result struct {
		Tools      []Tool `json:"tools"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return ListToolsResult{}, protocolError("tools/list", "invalid result", err)
	}
	return ListToolsResult{Session: session, Tools: result.Tools, NextCursor: strings.TrimSpace(result.NextCursor)}, nil
}

func (client *Client) ListAllTools(ctx context.Context) (ListToolsResult, error) {
	session, err := client.Initialize(ctx)
	if err != nil {
		return ListToolsResult{}, err
	}
	result := ListToolsResult{Session: session}
	cursor := ""
	seen := map[string]bool{}
	for page := 0; page < 100; page++ {
		current, err := client.ListTools(ctx, session, cursor)
		if err != nil {
			return result, err
		}
		result.Tools = append(result.Tools, current.Tools...)
		result.NextCursor = current.NextCursor
		if current.NextCursor == "" {
			return result, nil
		}
		if seen[current.NextCursor] {
			return result, protocolError("tools/list", "repeated pagination cursor", nil)
		}
		seen[current.NextCursor] = true
		cursor = current.NextCursor
	}
	return result, protocolError("tools/list", "pagination exceeded 100 pages", nil)
}

func (client *Client) CallTool(ctx context.Context, session Session, name string, arguments map[string]any) (CallResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CallResult{}, fmt.Errorf("tool name is required")
	}
	if arguments == nil {
		arguments = map[string]any{}
	}
	response, _, err := client.request(ctx, "tools/call", rpcRequest{
		JSONRPC: "2.0", ID: client.nextRequestID(), Method: "tools/call",
		Params: map[string]any{"name": name, "arguments": arguments},
	}, session)
	if err != nil {
		return CallResult{}, err
	}
	var result struct {
		IsError           bool            `json:"isError"`
		Content           json.RawMessage `json:"content"`
		StructuredContent any             `json:"structuredContent"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return CallResult{}, protocolError("tools/call", "invalid result", err)
	}
	data := result.StructuredContent
	if data == nil {
		data = parseToolContent(result.Content)
	}
	return CallResult{Session: session, Name: name, IsError: result.IsError, Content: result.Content, Data: data}, nil
}

func (client *Client) request(ctx context.Context, stage string, payload rpcRequest, session Session) (rpcResponse, http.Header, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return rpcResponse{}, nil, protocolError(stage, "encode request", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(data))
	if err != nil {
		return rpcResponse{}, nil, protocolError(stage, "build request", err)
	}
	request.Header = client.headers.Clone()
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("MCP-Protocol-Version", firstNonEmpty(session.ProtocolVersion, client.protocolVersion))
	if session.SessionID != "" {
		request.Header.Set("Mcp-Session-Id", session.SessionID)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		kind := ErrorNetwork
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			kind = ErrorTimeout
		}
		return rpcResponse{}, nil, &ClientError{Kind: kind, Stage: stage, Message: err.Error(), Cause: err}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return rpcResponse{}, response.Header, &ClientError{Kind: ErrorNetwork, Stage: stage, Message: "read response", Cause: err}
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return rpcResponse{}, response.Header, &ClientError{Kind: ErrorRedirect, Stage: stage, StatusCode: response.StatusCode, Message: "redirects are not allowed"}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return rpcResponse{}, response.Header, &ClientError{Kind: ErrorHTTP, Stage: stage, StatusCode: response.StatusCode, Message: http.StatusText(response.StatusCode)}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		if payload.ID == nil {
			return rpcResponse{}, response.Header, nil
		}
		return rpcResponse{}, response.Header, protocolError(stage, "empty response", nil)
	}
	parsed, err := parseRPCResponse(body, response.Header.Get("Content-Type"))
	if err != nil {
		return rpcResponse{}, response.Header, protocolError(stage, "decode response", err)
	}
	if parsed.Error != nil {
		return rpcResponse{}, response.Header, &ClientError{Kind: ErrorRPC, Stage: stage, Message: parsed.Error.Message}
	}
	if parsed.JSONRPC != "2.0" {
		return rpcResponse{}, response.Header, protocolError(stage, "response jsonrpc must be 2.0", nil)
	}
	if payload.ID != nil && !rpcIDsEqual(payload.ID, parsed.ID) {
		return rpcResponse{}, response.Header, protocolError(stage, "response id does not match request id", nil)
	}
	return parsed, response.Header, nil
}

func (client *Client) nextRequestID() int64 {
	return client.nextID.Add(1) - 1
}

func rejectRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

func protocolError(stage string, message string, cause error) error {
	return &ClientError{Kind: ErrorProtocol, Stage: stage, Message: message, Cause: cause}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func supportedProtocolVersion(value string) bool {
	switch strings.TrimSpace(value) {
	case "2025-06-18", "2025-03-26":
		return true
	default:
		return false
	}
}

func rpcIDsEqual(expected, actual any) bool {
	expectedJSON, expectedErr := json.Marshal(expected)
	actualJSON, actualErr := json.Marshal(actual)
	return expectedErr == nil && actualErr == nil && bytes.Equal(expectedJSON, actualJSON)
}
