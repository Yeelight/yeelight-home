package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

type StaticTokenSource string

func (source StaticTokenSource) Token(context.Context) (string, error) {
	return string(source), nil
}

type HTTPTransport struct {
	endpoint    Endpoint
	tokenSource TokenSource
	client      *http.Client
	ClientID    string
}

func NewHTTPTransport(endpoint Endpoint, tokenSource TokenSource, client *http.Client) HTTPTransport {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return HTTPTransport{
		endpoint:    endpoint,
		tokenSource: tokenSource,
		client:      client,
	}
}

func (transport HTTPTransport) Call(ctx context.Context, operation Operation, request Request) (Response, error) {
	method := strings.ToUpper(strings.TrimSpace(operation.Method))
	if method == "" {
		method = http.MethodPost
	}
	url := strings.TrimRight(transport.endpoint.BaseURL, "/") + "/" + strings.TrimLeft(operation.Path, "/")

	var body io.Reader
	if method != http.MethodGet {
		payload, err := json.Marshal(request.Parameters)
		if err != nil {
			return Response{}, fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	if method != http.MethodGet {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	token, err := transport.tokenSource.Token(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("load credential: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		httpRequest.Header.Set("Authorization", normalizeAuthorization(token))
	}
	if strings.TrimSpace(transport.ClientID) != "" {
		httpRequest.Header.Set("Client-Id", strings.TrimSpace(transport.ClientID))
	}

	httpResponse, err := transport.client.Do(httpRequest)
	if err != nil {
		return Response{}, fmt.Errorf("call %s: %w", operation.SemanticOperation, err)
	}
	defer httpResponse.Body.Close()

	data, err := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return Response{}, fmt.Errorf("call %s failed with status %d", operation.SemanticOperation, httpResponse.StatusCode)
	}

	parsed := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return Response{}, fmt.Errorf("decode response: %w", err)
		}
	}
	return Response{Status: "success", Data: parsed}, nil
}

func normalizeAuthorization(value string) string {
	token := strings.TrimSpace(value)
	for strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		return ""
	}
	return "Bearer " + token
}
