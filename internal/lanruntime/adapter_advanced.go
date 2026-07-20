package lanruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

func (adapter *Adapter) ExecuteAction(ctx context.Context, request ActionRequest) (Result, error) {
	actionName := strings.TrimSpace(request.ActionName)
	if actionName == "" {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "action", Message: "action name is required"}
	}
	target, capabilities, err := adapter.capabilitiesForTarget(ctx, request.Target)
	if err != nil {
		return Result{}, err
	}
	if !capabilities.supportsAction(actionName) {
		return Result{}, unsupported(fmt.Sprintf("target does not advertise action %q", actionName))
	}
	tool, ok := adapter.catalog.tools[roleAction]
	if !ok {
		return Result{}, unsupported("gateway does not expose a schema-compatible action tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleAction, operationValues{
		requestID: request.RequestID, target: target, action: "execute", actionName: actionName,
		payload: request.Payload, duration: request.Duration, delay: request.Delay,
	})
	if err != nil {
		return Result{}, err
	}
	if !schemaHasAnyArgument(tool.InputSchema, "action", "actionname") && (!mappedArgumentEquals(arguments, "capability", actionName) || !mappedArgumentExists(arguments, "value")) {
		return Result{}, unsupported("action tool cannot map the advertised action capability and payload")
	}
	return adapter.executeAcknowledged(ctx, tool, arguments, target)
}

func (adapter *Adapter) ExecuteFlow(ctx context.Context, request FlowRequest) (Result, error) {
	flowName := flowCapabilityName(request.Flow)
	if flowName == "" {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "flow", Message: "flow name or mode is required"}
	}
	target, capabilities, err := adapter.capabilitiesForTarget(ctx, request.Target)
	if err != nil {
		return Result{}, err
	}
	if !capabilities.supportsFlow(flowName) {
		return Result{}, unsupported(fmt.Sprintf("target does not advertise flow %q", flowName))
	}
	tool, ok := adapter.catalog.tools[roleFlow]
	if !ok {
		return Result{}, unsupported("gateway does not expose a schema-compatible flow tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleFlow, operationValues{
		requestID: request.RequestID, target: target, action: "execute", flow: request.Flow,
		payload: request.Payload, duration: request.Duration, delay: request.Delay,
	})
	if err != nil {
		return Result{}, err
	}
	if !schemaHasAnyArgument(tool.InputSchema, "flow") && (!mappedArgumentEquals(arguments, "capability", flowName) || !mappedArgumentExists(arguments, "value")) {
		return Result{}, unsupported("flow tool cannot map the advertised flow capability and payload")
	}
	return adapter.executeAcknowledged(ctx, tool, arguments, target)
}

func (adapter *Adapter) executeAcknowledged(ctx context.Context, tool lanmcp.Tool, arguments map[string]any, target Target) (Result, error) {
	called, err := adapter.call(ctx, tool, arguments, true)
	called.Target = target
	if err != nil {
		if isUncertain(err) {
			called.Outcome, called.CallError = OutcomeUncertain, err.Error()
			return called, nil
		}
		return Result{}, err
	}
	if gatewayAcknowledged(called.Data) {
		called.Outcome, called.Verified, called.Evidence = OutcomeApplied, true, "gateway_ack"
		return called, nil
	}
	called.Outcome = OutcomeUnverified
	return called, nil
}

func mappedArgumentExists(value any, name string) bool {
	wanted := normalizedArgumentName(name)
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if normalizedArgumentName(key) == wanted || mappedArgumentExists(child, name) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if mappedArgumentExists(child, name) {
				return true
			}
		}
	}
	return false
}

func mappedArgumentEquals(value any, name, expected string) bool {
	wanted := normalizedArgumentName(name)
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if normalizedArgumentName(key) == wanted && strings.EqualFold(strings.TrimSpace(fmt.Sprint(child)), strings.TrimSpace(expected)) {
				return true
			}
			if mappedArgumentEquals(child, name, expected) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if mappedArgumentEquals(child, name, expected) {
				return true
			}
		}
	}
	return false
}

func gatewayAcknowledged(data any) bool {
	switch typed := data.(type) {
	case map[string]any:
		for _, key := range []string{"accepted", "acknowledged", "applied", "executed", "ok", "success"} {
			if value, exists := typed[key]; exists {
				if accepted, ok := value.(bool); ok {
					return accepted
				}
			}
		}
		for _, key := range []string{"status", "result"} {
			if status, ok := typed[key].(string); ok {
				switch strings.ToLower(strings.TrimSpace(status)) {
				case "accepted", "applied", "executed", "ok", "success":
					return true
				}
			}
			if gatewayAcknowledged(typed[key]) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if gatewayAcknowledged(item) {
				return true
			}
		}
	}
	return false
}
