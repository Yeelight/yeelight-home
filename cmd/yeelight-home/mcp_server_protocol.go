package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

const (
	mcpLatestProtocolVersion = "2025-11-25"
	mcpMaxMessageBytes       = 4 << 20
)

var mcpSupportedProtocolVersions = map[string]bool{
	"2025-11-25": true,
	"2025-06-18": true,
	"2025-03-26": true,
}

type mcpRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpRPCError    `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type mcpInitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      map[string]any `json:"clientInfo"`
	Meta            map[string]any `json:"_meta,omitempty"`
}

type mcpListToolsParams struct {
	Cursor string         `json:"cursor,omitempty"`
	Meta   map[string]any `json:"_meta,omitempty"`
}

type mcpCallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

type localMCPServer struct {
	app                *app
	flags              cliFlags
	locale             string
	initializeReceived bool
	initialized        bool
	requestSeq         uint64
	stderr             io.Writer
}

func (server *localMCPServer) serveStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 64*1024), mcpMaxMessageBytes)
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		response, writeResponse := server.handleMessage(ctx, line)
		if !writeResponse {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write MCP response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP request: %w", err)
	}
	return nil
}

func (server *localMCPServer) handleMessage(ctx context.Context, data []byte) (mcpRPCResponse, bool) {
	var request mcpRPCRequest
	if err := json.Unmarshal(data, &request); err != nil {
		if json.Valid(data) {
			return mcpErrorResponse(nil, -32600, "Invalid Request", nil), true
		}
		return mcpErrorResponse(nil, -32700, "Parse error", nil), true
	}
	hasID := len(request.ID) > 0 && string(request.ID) != "null"
	if request.JSONRPC != "2.0" || request.Method == "" || (hasID && !validMCPID(request.ID)) {
		if !hasID {
			return mcpRPCResponse{}, false
		}
		return mcpErrorResponse(request.ID, -32600, "Invalid Request", nil), true
	}
	if !hasID {
		server.handleNotification(request)
		return mcpRPCResponse{}, false
	}

	switch request.Method {
	case "initialize":
		if server.initializeReceived {
			return mcpErrorResponse(request.ID, -32600, "Server already initialized", nil), true
		}
		return server.initialize(request), true
	case "ping":
		return mcpResultResponse(request.ID, map[string]any{}), true
	}
	if !server.initialized {
		return mcpErrorResponse(request.ID, -32002, "Server not initialized", nil), true
	}
	switch request.Method {
	case "tools/list":
		var params mcpListToolsParams
		if err := decodeMCPParams(request.Params, &params); err != nil {
			return mcpErrorResponse(request.ID, -32602, "Invalid tools/list params", err.Error()), true
		}
		return mcpResultResponse(request.ID, map[string]any{"tools": localMCPToolDefinitions()}), true
	case "tools/call":
		return server.callTool(ctx, request), true
	default:
		return mcpErrorResponse(request.ID, -32601, "Method not found", map[string]any{"method": request.Method}), true
	}
}

func (server *localMCPServer) initialize(request mcpRPCRequest) mcpRPCResponse {
	var params mcpInitializeParams
	if err := decodeMCPParams(request.Params, &params); err != nil {
		return mcpErrorResponse(request.ID, -32602, "Invalid initialize params", err.Error())
	}
	protocolVersion := params.ProtocolVersion
	if !mcpSupportedProtocolVersions[protocolVersion] {
		protocolVersion = mcpLatestProtocolVersion
	}
	server.initializeReceived = true
	return mcpResultResponse(request.ID, map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"serverInfo":   map[string]any{"name": "yeelight-home", "version": version},
		"instructions": "Use the Yeelight Home tools for home discovery, state queries, lighting control, and scenes. Ask the user before high-impact changes.",
	})
}

func (server *localMCPServer) handleNotification(request mcpRPCRequest) {
	if request.Method == "notifications/initialized" && server.initializeReceived {
		server.initialized = true
	}
}

func (server *localMCPServer) callTool(ctx context.Context, request mcpRPCRequest) mcpRPCResponse {
	var params mcpCallToolParams
	if err := decodeMCPParams(request.Params, &params); err != nil || params.Name == "" {
		message := "tool name is required"
		if err != nil {
			message = err.Error()
		}
		return mcpErrorResponse(request.ID, -32602, "Invalid tools/call params", message)
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}
	result, err := server.invokeTool(ctx, params.Name, params.Arguments)
	if err != nil {
		return mcpResultResponse(request.ID, mcpToolErrorResult(err))
	}
	return mcpResultResponse(request.ID, mcpToolResponseResult(result))
}

func decodeMCPParams(data json.RawMessage, target any) error {
	if len(data) == 0 || string(data) == "null" {
		data = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func mcpResultResponse(id json.RawMessage, result any) mcpRPCResponse {
	return mcpRPCResponse{JSONRPC: "2.0", ID: normalizedMCPID(id), Result: result}
}

func mcpErrorResponse(id json.RawMessage, code int, message string, data any) mcpRPCResponse {
	return mcpRPCResponse{JSONRPC: "2.0", ID: normalizedMCPID(id), Error: &mcpRPCError{Code: code, Message: message, Data: data}}
}

func normalizedMCPID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

func validMCPID(id json.RawMessage) bool {
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	switch value.(type) {
	case string, json.Number:
		return true
	default:
		return false
	}
}
