package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const cloudMCPMaxMessageBytes = 8 << 20

type cloudMCPProxy struct {
	endpoint        string
	headers         http.Header
	httpClient      *http.Client
	sessionID       string
	protocolVersion string
	contextProvider func() (runtimeContext, error)
	connectionKey   string
}

type cloudMCPRequest struct {
	id    json.RawMessage
	hasID bool
}

func (app *app) runCloudMCPProxy(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil || !mcpProxyFlagsAllowed(flags) || !flags.bool("stdio") {
		printMCPUsage(stderr)
		return exitInvalidInput
	}
	target := strings.ToLower(strings.TrimSpace(flags.string("target", "")))
	if target != "metadata" && target != "iot" {
		_, _ = fmt.Fprintln(stderr, "mcp proxy: target must be metadata or iot")
		return exitInvalidInput
	}
	contextInfo, err := app.resolveRuntimeContext(flags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp proxy: %v\n", err)
		return exitInvalidInput
	}
	if strings.TrimSpace(contextInfo.AccessToken) == "" {
		_, _ = fmt.Fprintln(stderr, "mcp proxy: Yeelight sign-in is required")
		return exitInvalidInput
	}
	proxy := newCloudMCPProxy(cloudMCPURL(contextInfo, target), contextInfo)
	proxy.contextProvider = func() (runtimeContext, error) {
		return app.resolveRuntimeContext(flags)
	}
	if err := proxy.serveStdio(context.Background(), stdin, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp proxy: %v\n", err)
		return exitInternalError
	}
	return exitOK
}

func mcpProxyFlagsAllowed(flags cliFlags) bool {
	for name := range flags.values {
		switch name {
		case "stdio", "target", "profile", "region", "house-id":
		default:
			return false
		}
	}
	return true
}

func cloudMCPURL(contextInfo runtimeContext, target string) string {
	baseURL := strings.TrimSuffix(contextInfo.Endpoint.BaseURL, "/apis/iot")
	if target == "metadata" {
		return baseURL + "/apis/metadata_mcp_server/v1/mcp"
	}
	return baseURL + "/apis/mcp_server/v1/mcp"
}

func newCloudMCPProxy(endpoint string, contextInfo runtimeContext) *cloudMCPProxy {
	headers := make(http.Header)
	headers.Set("Authorization", normalizeMCPAuthorization(contextInfo.AccessToken))
	headers.Set("Yeelight-Region", contextInfo.Region)
	if contextInfo.HouseID != "" {
		headers.Set("House-Id", contextInfo.HouseID)
	}
	proxy := &cloudMCPProxy{
		endpoint: endpoint,
		headers:  headers,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	proxy.connectionKey = cloudMCPConnectionKey(endpoint, headers)
	return proxy
}

func normalizeMCPAuthorization(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return value
	}
	return "Bearer " + value
}

func (proxy *cloudMCPProxy) serveStdio(ctx context.Context, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), cloudMCPMaxMessageBytes)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		request, err := parseCloudMCPRequest(line)
		if err != nil {
			if writeErr := writeProxyError(output, nil, err); writeErr != nil {
				return writeErr
			}
			continue
		}
		responses, err := proxy.forward(ctx, line)
		if err != nil {
			if !request.hasID {
				continue
			}
			if writeErr := writeProxyError(output, request.id, err); writeErr != nil {
				return writeErr
			}
			continue
		}
		if !request.hasID {
			continue
		}
		for _, response := range responses {
			if _, err := output.Write(append(response, '\n')); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func (proxy *cloudMCPProxy) forward(ctx context.Context, payload []byte) ([][]byte, error) {
	if _, err := parseCloudMCPRequest(payload); err != nil {
		return nil, err
	}
	if err := proxy.refreshRuntimeContext(); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, proxy.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request")
	}
	request.Header = proxy.headers.Clone()
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	if proxy.protocolVersion != "" {
		request.Header.Set("MCP-Protocol-Version", proxy.protocolVersion)
	}
	if proxy.sessionID != "" {
		request.Header.Set("Mcp-Session-Id", proxy.sessionID)
	}
	response, err := proxy.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("cloud MCP request failed")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, cloudMCPMaxMessageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read cloud MCP response")
	}
	if len(body) > cloudMCPMaxMessageBytes {
		return nil, fmt.Errorf("cloud MCP response is too large")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("cloud MCP returned HTTP %d", response.StatusCode)
	}
	if sessionID := strings.TrimSpace(response.Header.Get("Mcp-Session-Id")); sessionID != "" {
		proxy.sessionID = sessionID
	}
	messages, err := parseCloudMCPMessages(body, response.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	for _, message := range messages {
		proxy.captureProtocolVersion(message)
	}
	return messages, nil
}

func (proxy *cloudMCPProxy) refreshRuntimeContext() error {
	if proxy.contextProvider == nil {
		return nil
	}
	contextInfo, err := proxy.contextProvider()
	if err != nil {
		return fmt.Errorf("load local Yeelight credentials")
	}
	if strings.TrimSpace(contextInfo.AccessToken) == "" {
		return fmt.Errorf("Yeelight sign-in is required")
	}
	headers := make(http.Header)
	headers.Set("Authorization", normalizeMCPAuthorization(contextInfo.AccessToken))
	headers.Set("Yeelight-Region", contextInfo.Region)
	if contextInfo.HouseID != "" {
		headers.Set("House-Id", contextInfo.HouseID)
	}
	endpoint := cloudMCPURL(contextInfo, cloudMCPTarget(proxy.endpoint))
	key := cloudMCPConnectionKey(endpoint, headers)
	if key != proxy.connectionKey {
		proxy.sessionID = ""
		proxy.protocolVersion = ""
	}
	proxy.endpoint = endpoint
	proxy.headers = headers
	proxy.connectionKey = key
	return nil
}

func cloudMCPTarget(endpoint string) string {
	if strings.Contains(endpoint, "/metadata_mcp_server/") {
		return "metadata"
	}
	return "iot"
}

func cloudMCPConnectionKey(endpoint string, headers http.Header) string {
	return strings.Join([]string{
		endpoint,
		headers.Get("Authorization"),
		headers.Get("Yeelight-Region"),
		headers.Get("House-Id"),
	}, "\x00")
}

func parseCloudMCPMessages(body []byte, contentType string) ([][]byte, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil
	}
	if !strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		if err := validateCloudMCPMessage(body); err != nil {
			return nil, fmt.Errorf("cloud MCP returned invalid JSON")
		}
		return [][]byte{append([]byte(nil), body...)}, nil
	}
	var messages [][]byte
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), cloudMCPMaxMessageBytes)
	var eventData []string
	flush := func() error {
		if len(eventData) == 0 {
			return nil
		}
		data := bytes.TrimSpace([]byte(strings.Join(eventData, "\n")))
		eventData = eventData[:0]
		if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
			return nil
		}
		if err := validateCloudMCPMessage(data); err != nil {
			return fmt.Errorf("cloud MCP returned invalid SSE data")
		}
		messages = append(messages, append([]byte(nil), data...))
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		if strings.HasPrefix(data, " ") {
			data = data[1:]
		}
		eventData = append(eventData, data)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return messages, nil
}

func validateCloudMCPMessage(payload []byte) error {
	var message map[string]json.RawMessage
	if err := json.Unmarshal(payload, &message); err != nil || message == nil {
		return fmt.Errorf("invalid JSON-RPC message")
	}
	var version string
	if err := json.Unmarshal(message["jsonrpc"], &version); err != nil || version != "2.0" {
		return fmt.Errorf("invalid JSON-RPC version")
	}
	if _, hasMethod := message["method"]; hasMethod {
		return nil
	}
	_, hasResult := message["result"]
	_, hasError := message["error"]
	if hasResult == hasError {
		return fmt.Errorf("invalid JSON-RPC response")
	}
	return nil
}

func (proxy *cloudMCPProxy) captureProtocolVersion(message []byte) {
	var response struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if json.Unmarshal(message, &response) == nil && response.Result.ProtocolVersion != "" {
		proxy.protocolVersion = response.Result.ProtocolVersion
	}
}

func parseCloudMCPRequest(payload []byte) (cloudMCPRequest, error) {
	var message map[string]json.RawMessage
	if err := json.Unmarshal(payload, &message); err != nil || message == nil {
		return cloudMCPRequest{}, fmt.Errorf("invalid JSON-RPC message")
	}
	var version string
	if err := json.Unmarshal(message["jsonrpc"], &version); err != nil || version != "2.0" {
		return cloudMCPRequest{}, fmt.Errorf("invalid JSON-RPC version")
	}
	var method string
	if err := json.Unmarshal(message["method"], &method); err != nil || strings.TrimSpace(method) == "" {
		return cloudMCPRequest{}, fmt.Errorf("invalid JSON-RPC method")
	}
	id, hasID := message["id"]
	return cloudMCPRequest{id: id, hasID: hasID}, nil
}

func writeProxyError(output io.Writer, id json.RawMessage, err error) error {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error": map[string]any{
			"code":    -32000,
			"message": err.Error(),
		},
	}
	if len(id) > 0 {
		response["id"] = id
	}
	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return marshalErr
	}
	_, writeErr := output.Write(append(data, '\n'))
	return writeErr
}
