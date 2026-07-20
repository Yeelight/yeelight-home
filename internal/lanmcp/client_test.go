package lanmcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestClientSessionPaginationAndToolCall(t *testing.T) {
	var methods []string
	var cursors []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var rpc rpcRequest
		if err := json.NewDecoder(request.Body).Decode(&rpc); err != nil {
			t.Fatalf("Decode request error: %v", err)
		}
		methods = append(methods, rpc.Method)
		writer.Header().Set("Content-Type", "application/json")
		switch rpc.Method {
		case "initialize":
			if request.Header.Get("MCP-Protocol-Version") != DefaultProtocolVersion {
				t.Fatalf("protocol header = %q", request.Header.Get("MCP-Protocol-Version"))
			}
			writer.Header().Set("Mcp-Session-Id", "session-1")
			writeRPCResult(t, writer, rpc.ID, map[string]any{"protocolVersion": DefaultProtocolVersion, "serverInfo": map[string]any{"name": "gateway"}})
		case "notifications/initialized":
			if request.Header.Get("Mcp-Session-Id") != "session-1" {
				t.Fatalf("session header = %q", request.Header.Get("Mcp-Session-Id"))
			}
			writer.WriteHeader(http.StatusAccepted)
		case "tools/list":
			if request.Header.Get("Mcp-Session-Id") != "session-1" {
				t.Fatalf("session header = %q", request.Header.Get("Mcp-Session-Id"))
			}
			params := rpc.Params.(map[string]any)
			cursor, _ := params["cursor"].(string)
			cursors = append(cursors, cursor)
			if cursor == "" {
				writeRPCResult(t, writer, rpc.ID, map[string]any{"tools": []any{map[string]any{"name": "list_nodes", "inputSchema": map[string]any{"type": "object"}}}, "nextCursor": "page-2"})
			} else {
				writeRPCResult(t, writer, rpc.ID, map[string]any{"tools": []any{map[string]any{"name": "control_node", "description": "Control a node", "inputSchema": map[string]any{"type": "object"}}}})
			}
		case "tools/call":
			params := rpc.Params.(map[string]any)
			if params["name"] != "control_node" {
				t.Fatalf("params = %#v", params)
			}
			writeRPCResult(t, writer, rpc.ID, map[string]any{
				"isError": false,
				"content": []any{map[string]any{"type": "text", "text": `{"ok":true,"power":true}`}},
			})
		default:
			http.Error(writer, "unexpected method", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/mcp", Options{HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	tools, err := client.ListAllTools(context.Background())
	if err != nil {
		t.Fatalf("ListAllTools error: %v", err)
	}
	if len(tools.Tools) != 2 || tools.Tools[1].Name != "control_node" || !reflect.DeepEqual(cursors, []string{"", "page-2"}) {
		t.Fatalf("tools = %#v, cursors = %#v", tools.Tools, cursors)
	}
	result, err := client.CallTool(context.Background(), tools.Session, "control_node", map[string]any{"nodeId": "1"})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	data := result.Data.(map[string]any)
	if data["ok"] != true || data["power"] != true {
		t.Fatalf("data = %#v", result.Data)
	}
	if !reflect.DeepEqual(methods, []string{"initialize", "notifications/initialized", "tools/list", "tools/list", "tools/call"}) {
		t.Fatalf("methods = %#v", methods)
	}
}

func TestClientParsesSSEResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = writer.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"tools\":[{\"name\":\"one\"}]}}\n\n"))
	}))
	defer server.Close()
	client, err := NewClient(server.URL+"/mcp", Options{HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	result, err := client.ListTools(context.Background(), Session{ProtocolVersion: DefaultProtocolVersion, Stateless: true}, "")
	if err != nil || len(result.Tools) != 1 || result.Tools[0].Name != "one" {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
}

func TestClientSupportsStatelessServer(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var rpc rpcRequest
		_ = json.NewDecoder(request.Body).Decode(&rpc)
		methods = append(methods, rpc.Method)
		writer.Header().Set("Content-Type", "application/json")
		writeRPCResult(t, writer, rpc.ID, map[string]any{"protocolVersion": DefaultProtocolVersion})
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/mcp", Options{HTTPClient: server.Client()})
	session, err := client.Initialize(context.Background())
	if err != nil || !session.Stateless || len(methods) != 1 {
		t.Fatalf("session = %#v, methods = %#v, err = %v", session, methods, err)
	}
}

func TestClientClassifiesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(100 * time.Millisecond)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/mcp", Options{Timeout: 10 * time.Millisecond})
	_, err := client.Initialize(context.Background())
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Kind != ErrorTimeout {
		t.Fatalf("error = %#v", err)
	}
}

func TestClientRejectsRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL+"/mcp", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/mcp", Options{})
	_, err := client.Initialize(context.Background())
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Kind != ErrorRedirect {
		t.Fatalf("error = %#v", err)
	}
}

func TestClientReturnsJSONRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"unsupported"}}`))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL+"/mcp", Options{HTTPClient: server.Client()})
	_, err := client.Initialize(context.Background())
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Kind != ErrorRPC || clientErr.Message != "unsupported" {
		t.Fatalf("error = %#v", err)
	}
}

func TestClientRejectsInvalidJSONRPCEnvelopeAndRequestID(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing jsonrpc", body: `{"id":1,"result":{"protocolVersion":"2025-06-18"}}`},
		{name: "mismatched id", body: `{"jsonrpc":"2.0","id":2,"result":{"protocolVersion":"2025-06-18"}}`},
		{name: "empty response", body: ``},
		{name: "unsupported protocol", body: `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2099-01-01"}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := NewClient(server.URL+"/mcp", Options{HTTPClient: server.Client()})
			if err != nil {
				t.Fatalf("NewClient error: %v", err)
			}
			_, err = client.Initialize(context.Background())
			var clientErr *ClientError
			if !errors.As(err, &clientErr) || clientErr.Kind != ErrorProtocol {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestDefaultLANClientBypassesEnvironmentProxy(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:18080/mcp", Options{})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatalf("transport = %#v", client.httpClient.Transport)
	}
}

func writeRPCResult(t *testing.T, writer http.ResponseWriter, id any, result any) {
	t.Helper()
	if err := json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result}); err != nil {
		t.Fatalf("Encode response error: %v", err)
	}
}
