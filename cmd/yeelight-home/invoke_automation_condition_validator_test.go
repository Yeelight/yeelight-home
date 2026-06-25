package main

import (
	"testing"

	"github.com/yeelight/yeelight-home/internal/api"
)

func TestInvokeAutomationCreateRejectsSourceBackedV2ConditionShape(t *testing.T) {
	tests := []struct {
		name       string
		repeatType int
		params     map[string]any
	}{
		{
			name:       "top level condition group must have a type",
			repeatType: 2,
			params: map[string]any{
				"type":       "and",
				"conditions": []any{map[string]any{"conditions": []any{map[string]any{"type": "fact", "prop": "p", "value": true}}}},
			},
		},
		{
			name:       "top level condition group must not be empty",
			repeatType: 2,
			params: map[string]any{
				"type":       "and",
				"conditions": []any{map[string]any{"type": "or", "conditions": []any{}}},
			},
		},
		{
			name:       "event groups must be unique",
			repeatType: 2,
			params: map[string]any{
				"type": "and",
				"conditions": []any{
					map[string]any{"type": "or", "conditions": []any{map[string]any{"type": "alarm", "clock": "22:00:00"}}},
					map[string]any{"type": "or", "conditions": []any{map[string]any{"type": "event", "pid": 1, "id": 1, "resId": "50018330"}}},
				},
			},
		},
		{
			name:       "fact groups must be unique",
			repeatType: 2,
			params: map[string]any{
				"type": "and",
				"conditions": []any{
					map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "fact", "prop": "p", "value": true}}},
					map[string]any{"type": "or", "conditions": []any{map[string]any{"type": "fact", "prop": "l", "value": 50}}},
				},
			},
		},
		{
			name:       "mixed non event group types are rejected",
			repeatType: 2,
			params: map[string]any{
				"type": "and",
				"conditions": []any{map[string]any{
					"type": "and",
					"conditions": []any{
						map[string]any{"type": "fact", "prop": "p", "value": true},
						map[string]any{"type": "unknown", "value": true},
					},
				}},
			},
		},
		{
			name:       "event conditions cannot repeat once",
			repeatType: 1,
			params: map[string]any{
				"type":       "and",
				"conditions": []any{map[string]any{"type": "or", "conditions": []any{map[string]any{"type": "event", "pid": 1, "id": 1, "resId": "50018330"}}}},
			},
		},
		{
			name:       "fact conditions cannot repeat once",
			repeatType: 1,
			params: map[string]any{
				"type":       "and",
				"conditions": []any{map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "fact", "prop": "p", "value": true}}}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := map[string]any{
				"repeatType": test.repeatType,
				"params":     test.params,
				"actions":    []map[string]any{{"typeId": 2, "resId": "50018330", "params": `{"set":{"p":false}}`}},
			}
			reason := validateAutomationCreatePayload(payload, api.EntityListResult{
				Counts: map[string]int{},
				Entities: []api.EntitySummary{{
					Type: "device",
					ID:   "50018330",
					Name: "主灯",
				}},
			})
			if reason != "invalid_automation_params" {
				t.Fatalf("reason = %q", reason)
			}
		})
	}
}

func TestInvokeAutomationCreateRejectsNestedConditionUnknownReference(t *testing.T) {
	payload := map[string]any{
		"repeatType": 2,
		"params": map[string]any{
			"type":       "and",
			"conditions": []any{map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "fact", "resId": "999999", "typeId": 2, "prop": "p", "value": true}}}},
		},
		"actions": []map[string]any{{"typeId": 2, "resId": "50018330", "params": `{"set":{"p":false}}`}},
	}
	reason := validateAutomationCreatePayload(payload, api.EntityListResult{
		Counts: map[string]int{},
		Entities: []api.EntitySummary{{
			Type: "device",
			ID:   "50018330",
			Name: "主灯",
		}},
	})
	if reason != "invalid_automation_condition_reference" {
		t.Fatalf("reason = %q", reason)
	}
}

func TestInvokeAutomationCreateAllowsSourceBackedV2ConditionShape(t *testing.T) {
	payload := map[string]any{
		"repeatType": 2,
		"params": map[string]any{
			"type": "and",
			"conditions": []any{
				map[string]any{"type": "or", "conditions": []any{map[string]any{"type": "alarm", "clock": "22:00:00"}}},
				map[string]any{"type": "and", "conditions": []any{map[string]any{"type": "fact", "resId": "50018330", "typeId": 2, "prop": "p", "value": true}}},
			},
		},
		"actions": []map[string]any{{"typeId": 2, "resId": "50018330", "params": `{"set":{"p":false}}`}},
	}
	reason := validateAutomationCreatePayload(payload, api.EntityListResult{
		Counts: map[string]int{},
		Entities: []api.EntitySummary{{
			Type: "device",
			ID:   "50018330",
			Name: "主灯",
		}},
	})
	if reason != "" {
		t.Fatalf("reason = %q", reason)
	}
}
