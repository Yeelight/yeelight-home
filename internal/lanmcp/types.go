package lanmcp

import (
	"encoding/json"
	"fmt"
)

const DefaultProtocolVersion = "2025-06-18"

type ErrorKind string

const (
	ErrorNetwork  ErrorKind = "network"
	ErrorTimeout  ErrorKind = "timeout"
	ErrorHTTP     ErrorKind = "http"
	ErrorRedirect ErrorKind = "redirect"
	ErrorRPC      ErrorKind = "json-rpc"
	ErrorProtocol ErrorKind = "protocol"
)

type ClientError struct {
	Kind       ErrorKind
	Stage      string
	StatusCode int
	Message    string
	Cause      error
}

func (err *ClientError) Error() string {
	if err.StatusCode > 0 {
		return fmt.Sprintf("%s failed: HTTP %d: %s", err.Stage, err.StatusCode, err.Message)
	}
	return fmt.Sprintf("%s failed: %s", err.Stage, err.Message)
}

func (err *ClientError) Unwrap() error {
	return err.Cause
}

type Session struct {
	ProtocolVersion string         `json:"protocolVersion"`
	SessionID       string         `json:"sessionId,omitempty"`
	Stateless       bool           `json:"stateless"`
	ServerInfo      map[string]any `json:"serverInfo,omitempty"`
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

type ListToolsResult struct {
	Session    Session `json:"session"`
	Tools      []Tool  `json:"tools"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

type CallResult struct {
	Session Session         `json:"session"`
	Name    string          `json:"name"`
	IsError bool            `json:"isError"`
	Content json.RawMessage `json:"content,omitempty"`
	Data    any             `json:"data,omitempty"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
