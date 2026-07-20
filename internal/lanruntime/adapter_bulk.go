package lanruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

func (adapter *Adapter) Toggle(ctx context.Context, request PropertyRequest) (Result, error) {
	current, err := adapter.Query(ctx, PropertyRequest{RequestID: request.RequestID, Target: request.Target, Property: request.Property})
	if err != nil {
		return Result{}, err
	}
	value, ok := booleanValue(current.Value)
	if !ok {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "toggle", Message: "current property value is not boolean"}
	}
	return adapter.Set(ctx, PropertyRequest{RequestID: request.RequestID, Target: current.Target, Property: request.Property, Value: !value})
}

func (adapter *Adapter) SetProperties(ctx context.Context, request PropertiesRequest) (Result, error) {
	if len(request.Properties) == 0 {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "properties", Message: "at least one property is required"}
	}
	keys := make([]string, 0, len(request.Properties))
	for key := range request.Properties {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	results := make([]any, 0, len(keys))
	verified := map[string]any{}
	target := request.Target
	for index, property := range keys {
		result, err := adapter.Set(ctx, PropertyRequest{RequestID: childRequestID(request.RequestID, index), Target: target, Property: property, Value: request.Properties[property]})
		if err != nil {
			return Result{}, err
		}
		results = append(results, result)
		target = result.Target
		verified[property] = result.Value
		if result.Outcome != OutcomeApplied {
			result.Data = results
			return result, nil
		}
	}
	return Result{Outcome: OutcomeApplied, Tool: adapter.catalog.tools[roleControl].Name, Target: target, ExpectedValue: request.Properties, Value: verified, Verified: true, Data: results}, nil
}

func (adapter *Adapter) BatchSet(ctx context.Context, request BatchPropertyRequest) (Result, error) {
	if len(request.Targets) == 0 || strings.TrimSpace(request.Property) == "" {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "batch", Message: "targets and property are required"}
	}
	results := make([]any, 0, len(request.Targets))
	for index, target := range request.Targets {
		result, err := adapter.Set(ctx, PropertyRequest{RequestID: childRequestID(request.RequestID, index), Target: target, Property: request.Property, Value: request.Value})
		if err != nil {
			return Result{}, err
		}
		results = append(results, result)
		if result.Outcome != OutcomeApplied {
			result.Data = results
			return result, nil
		}
	}
	return Result{
		Outcome: OutcomeApplied, Tool: adapter.catalog.tools[roleControl].Name,
		Target:   Target{Type: request.Targets[0].Type, ID: joinedTargetIDs(request.Targets)},
		Property: request.Property, ExpectedValue: request.Value, Value: request.Value,
		Verified: true, Data: results,
	}, nil
}

func childRequestID(requestID string, index int) string {
	if strings.TrimSpace(requestID) == "" {
		return ""
	}
	return fmt.Sprintf("%s-%d", requestID, index+1)
}

func (adapter *Adapter) BatchQuery(ctx context.Context, requests []PropertyRequest) (Result, error) {
	if len(requests) == 0 {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "batch-query", Message: "at least one query is required"}
	}
	results := make([]any, 0, len(requests))
	for _, request := range requests {
		result, err := adapter.Query(ctx, request)
		if err != nil {
			return Result{}, err
		}
		results = append(results, result)
	}
	return Result{Outcome: OutcomeReadSuccess, Tool: "batch", Value: results, Data: results}, nil
}

func booleanValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "on", "1":
			return true, true
		case "false", "off", "0":
			return false, true
		}
	case float64:
		if typed == 0 || typed == 1 {
			return typed == 1, true
		}
	}
	return false, false
}

func joinedTargetIDs(targets []Target) string {
	ids := make([]string, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	return strings.Join(ids, ",")
}
