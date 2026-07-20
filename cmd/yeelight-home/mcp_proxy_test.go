package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yeelight/yeelight-home/internal/api"
)

func TestCloudMCPProxyInjectsCredentialsAndPreservesSession(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer secret-token" || request.Header.Get("Yeelight-Region") != "cn" || request.Header.Get("House-Id") != "house-1" {
			t.Errorf("headers = %#v", request.Header)
		}
		body, _ := io.ReadAll(request.Body)
		if requests == 1 {
			if !bytes.Contains(body, []byte(`"method":"initialize"`)) {
				t.Errorf("initialize body = %s", body)
			}
			writer.Header().Set("Mcp-Session-Id", "session-1")
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"cloud"}}}`)
			return
		}
		if request.Header.Get("Mcp-Session-Id") != "session-1" || request.Header.Get("MCP-Protocol-Version") != "2025-06-18" {
			t.Errorf("session headers = %#v", request.Header)
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[]}}\n\n")
	}))
	defer server.Close()

	proxy := newCloudMCPProxy(server.URL, runtimeContext{AccessToken: "secret-token", Region: "cn", HouseID: "house-1"})
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	var output bytes.Buffer
	if err := proxy.serveStdio(context.Background(), input, &output); err != nil {
		t.Fatalf("serveStdio error: %v", err)
	}
	if requests != 2 || !strings.Contains(output.String(), `"id":1`) || !strings.Contains(output.String(), `"id":2`) {
		t.Fatalf("requests = %d, output = %s", requests, output.String())
	}
}

func TestCloudMCPProxyParsesMultilineSSEEvents(t *testing.T) {
	body := []byte("event: message\r\ndata: {\"jsonrpc\":\"2.0\",\r\ndata: \"id\":2,\"result\":{\"tools\":[]}}\r\n\r\ndata: [DONE]\r\n\r\n")
	messages, err := parseCloudMCPMessages(body, "text/event-stream; charset=utf-8")
	if err != nil {
		t.Fatalf("parseCloudMCPMessages error: %v", err)
	}
	if len(messages) != 1 || !strings.Contains(string(messages[0]), `"id":2`) {
		t.Fatalf("messages = %q", messages)
	}
}

func TestCloudMCPProxyNotificationNeverWritesStdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	proxy := newCloudMCPProxy(server.URL, runtimeContext{AccessToken: "secret", Region: "cn"})
	var output bytes.Buffer
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n")
	if err := proxy.serveStdio(context.Background(), input, &output); err != nil {
		t.Fatalf("serveStdio error: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("notification output = %q", output.String())
	}
}

func TestCloudMCPProxyRejectsInvalidEnvelopeAndBatch(t *testing.T) {
	proxy := newCloudMCPProxy("http://127.0.0.1", runtimeContext{AccessToken: "secret", Region: "cn"})
	var output bytes.Buffer
	input := strings.NewReader("[]\n{\"jsonrpc\":\"1.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	if err := proxy.serveStdio(context.Background(), input, &output); err != nil {
		t.Fatalf("serveStdio error: %v", err)
	}
	if strings.Count(strings.TrimSpace(output.String()), "\n") != 1 || strings.Count(output.String(), `"id":null`) != 2 {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCloudMCPProxyRefreshesLocalCredentialsAndResetsSession(t *testing.T) {
	var authorizations []string
	var sessions []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorizations = append(authorizations, request.Header.Get("Authorization"))
		sessions = append(sessions, request.Header.Get("Mcp-Session-Id"))
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Mcp-Session-Id", "session")
		_, _ = fmt.Fprintf(writer, `{"jsonrpc":"2.0","id":%d,"result":{"tools":[]}}`, len(authorizations))
	}))
	defer server.Close()

	contextInfo := runtimeContext{AccessToken: "first", Region: "cn", Endpoint: runtimeContextEndpoint(server.URL)}
	proxy := newCloudMCPProxy(server.URL+"/apis/metadata_mcp_server/v1/mcp", contextInfo)
	proxy.contextProvider = func() (runtimeContext, error) { return contextInfo, nil }
	if _, err := proxy.forward(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)); err != nil {
		t.Fatalf("first forward error: %v", err)
	}
	contextInfo.AccessToken = "second"
	if _, err := proxy.forward(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)); err != nil {
		t.Fatalf("second forward error: %v", err)
	}
	if fmt.Sprint(authorizations) != "[Bearer first Bearer second]" {
		t.Fatalf("authorizations = %v", authorizations)
	}
	if fmt.Sprint(sessions) != "[ ]" {
		t.Fatalf("sessions = %v", sessions)
	}
	if proxy.sessionID != "session" {
		t.Fatalf("sessionID = %q", proxy.sessionID)
	}
}

func TestCloudMCPProxyTimeoutAndOversizedResponseAreRedacted(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		timeout time.Duration
	}{
		{name: "timeout", timeout: 10 * time.Millisecond, handler: func(writer http.ResponseWriter, _ *http.Request) {
			time.Sleep(50 * time.Millisecond)
			_, _ = fmt.Fprint(writer, `{"jsonrpc":"2.0","id":1,"result":{}}`)
		}},
		{name: "oversized", timeout: time.Second, handler: func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write(bytes.Repeat([]byte("x"), cloudMCPMaxMessageBytes+1))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(test.handler)
			defer server.Close()
			proxy := newCloudMCPProxy(server.URL, runtimeContext{AccessToken: "client-secret", Region: "cn"})
			proxy.httpClient.Timeout = test.timeout
			var output bytes.Buffer
			input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/list\"}\n")
			if err := proxy.serveStdio(context.Background(), input, &output); err != nil {
				t.Fatalf("serveStdio error: %v", err)
			}
			if !strings.Contains(output.String(), `"id":1`) || strings.Contains(output.String(), "client-secret") {
				t.Fatalf("output = %q", output.String())
			}
		})
	}
}

func runtimeContextEndpoint(baseURL string) api.Endpoint {
	return api.Endpoint{Region: "cn", BaseURL: strings.TrimSuffix(baseURL, "/") + "/apis/iot"}
}

func TestCloudMCPProxyReturnsRedactedJSONRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(writer, "Bearer server-secret")
	}))
	defer server.Close()

	proxy := newCloudMCPProxy(server.URL, runtimeContext{AccessToken: "client-secret", Region: "cn"})
	var output bytes.Buffer
	if err := proxy.serveStdio(context.Background(), strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"tools/list\"}\n"), &output); err != nil {
		t.Fatalf("serveStdio error: %v", err)
	}
	if !strings.Contains(output.String(), `"id":9`) || strings.Contains(output.String(), "server-secret") || strings.Contains(output.String(), "client-secret") {
		t.Fatalf("output = %s", output.String())
	}
}
