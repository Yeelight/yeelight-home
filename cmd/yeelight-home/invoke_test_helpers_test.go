package main

import (
	"encoding/json"
	"testing"
)

func decodeInvokeResponse(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("invalid invoke response: %v\n%s", err, string(data))
	}
	return response
}
