package lanmcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func parseRPCResponse(body []byte, contentType string) (rpcResponse, error) {
	if strings.Contains(strings.ToLower(contentType), "text/event-stream") {
		body = lastSSEData(body)
	}
	var response rpcResponse
	if len(bytes.TrimSpace(body)) == 0 {
		return response, fmt.Errorf("empty response")
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return response, err
	}
	return response, nil
}

func lastSSEData(body []byte) []byte {
	events := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n\n")
	var last string
	for _, event := range events {
		parts := make([]string, 0)
		for _, line := range strings.Split(event, "\n") {
			if strings.HasPrefix(line, "data:") {
				value := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if value != "" && value != "[DONE]" {
					parts = append(parts, value)
				}
			}
		}
		if len(parts) > 0 {
			last = strings.Join(parts, "\n")
		}
	}
	return []byte(last)
}

func parseToolContent(content json.RawMessage) any {
	if len(content) == 0 {
		return nil
	}
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil || len(blocks) == 0 {
		return nil
	}
	first := blocks[0]
	text, textOK := first["text"].(string)
	if first["type"] != "text" || !textOK {
		return first
	}
	var value any
	if json.Unmarshal([]byte(text), &value) == nil {
		return value
	}
	return text
}
