package lanmcp

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const DefaultPort = "18080"

func EndpointForGateway(gatewayIP string) (string, error) {
	host, err := normalizeLocalHost(gatewayIP)
	if err != nil {
		return "", err
	}
	return (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, DefaultPort),
		Path:   "/mcp",
	}).String(), nil
}

func NormalizeEndpoint(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid LAN MCP endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("LAN MCP endpoint must use http or https")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("LAN MCP endpoint must not contain credentials")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("LAN MCP endpoint must not contain a query or fragment")
	}
	host, err := normalizeLocalHost(parsed.Hostname())
	if err != nil {
		return "", err
	}
	port := parsed.Port()
	if port == "" {
		port = DefaultPort
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	if path == "" {
		path = "/mcp"
	}
	if path != "/mcp" {
		return "", fmt.Errorf("LAN MCP endpoint path must be /mcp")
	}
	parsed.Host = net.JoinHostPort(host, port)
	parsed.Path = "/mcp"
	parsed.RawPath = ""
	return parsed.String(), nil
}

func normalizeLocalHost(raw string) (string, error) {
	host := strings.TrimSpace(strings.Trim(raw, "[]"))
	if strings.EqualFold(host, "localhost") {
		return "localhost", nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("gateway host must be a local IP address")
	}
	if !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
		return "", fmt.Errorf("gateway host must be a private, loopback, or link-local IP address")
	}
	return ip.String(), nil
}
