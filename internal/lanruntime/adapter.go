package lanruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yeelight/yeelight-home/internal/lanmcp"
)

type Adapter struct {
	client               *lanmcp.Client
	catalog              catalog
	session              lanmcp.Session
	verificationAttempts int
	verificationInterval time.Duration
}

func (adapter *Adapter) ListTargets(ctx context.Context, houseID string) ([]Target, error) {
	tool, ok := adapter.catalog.tools[roleList]
	if !ok {
		return nil, unsupported("gateway does not expose a schema-compatible node-list tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleList, operationValues{target: Target{HouseID: houseID}})
	if err != nil {
		return nil, err
	}
	result, err := adapter.call(ctx, tool, arguments, false)
	if err != nil {
		return nil, err
	}
	return projectTargets(result.Data, houseID), nil
}

func Connect(ctx context.Context, options Options) (*Adapter, error) {
	if options.Client == nil {
		return nil, &Error{Kind: ErrorPreCall, Stage: "connect", Message: "LAN MCP client is required"}
	}
	tools, err := options.Client.ListAllTools(ctx)
	if err != nil {
		return nil, wrapClientError(ErrorPreCall, "discover", err)
	}
	attempts := options.VerificationAttempts
	if attempts <= 0 {
		attempts = 8
	}
	interval := options.VerificationInterval
	if interval <= 0 {
		interval = 300 * time.Millisecond
	}
	return &Adapter{
		client: options.Client, catalog: discoverCatalog(tools.Tools), session: tools.Session,
		verificationAttempts: attempts, verificationInterval: interval,
	}, nil
}

func (adapter *Adapter) Query(ctx context.Context, request PropertyRequest) (Result, error) {
	target, err := adapter.resolveTarget(ctx, request.Target)
	if err != nil {
		return Result{}, err
	}
	if tool, ok := adapter.catalog.tools[roleState]; ok {
		arguments, err := buildToolArguments(tool.InputSchema, roleState, operationValues{requestID: request.RequestID, target: target, property: request.Property})
		if err != nil {
			return Result{}, err
		}
		called, err := adapter.call(ctx, tool, arguments, false)
		if err != nil {
			return Result{}, err
		}
		if request.Property == "" {
			called.Outcome, called.Target, called.Value = OutcomeReadSuccess, target, called.Data
			return called, nil
		}
		value, found := extractPropertyValue(called.Data, target, request.Property)
		if !found {
			return Result{}, &Error{Kind: ErrorRejected, Stage: "read-back", Message: "gateway response did not contain the requested property"}
		}
		called.Outcome, called.Target, called.Property, called.Value = OutcomeReadSuccess, target, request.Property, value
		return called, nil
	}
	return adapter.queryThroughList(ctx, target, request.Property, request.RequestID)
}

func (adapter *Adapter) Set(ctx context.Context, request PropertyRequest) (Result, error) {
	target, err := adapter.resolveTarget(ctx, request.Target)
	if err != nil {
		return Result{}, err
	}
	tool, ok := adapter.catalog.tools[roleControl]
	if !ok {
		return Result{}, unsupported("gateway does not expose a schema-compatible control tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleControl, operationValues{
		requestID: request.RequestID, target: target, property: request.Property, value: request.Value, valueSet: true,
		properties: map[string]any{request.Property: request.Value}, action: "set",
	})
	if err != nil {
		return Result{}, err
	}
	called, callErr := adapter.call(ctx, tool, arguments, true)
	called.Target, called.Property, called.ExpectedValue = target, request.Property, request.Value
	if callErr != nil {
		if !isUncertain(callErr) {
			return Result{}, callErr
		}
		called.CallError = callErr.Error()
		return adapter.verifyUncertainWrite(ctx, called, request)
	}
	return adapter.verifySuccessfulWrite(ctx, called, request)
}

func (adapter *Adapter) Adjust(ctx context.Context, request AdjustRequest) (Result, error) {
	current, err := adapter.Query(ctx, PropertyRequest{RequestID: request.RequestID, Target: request.Target, Property: request.Property})
	if err != nil {
		return Result{}, err
	}
	value, ok := numberValue(current.Value)
	if !ok {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "adjust", Message: "current property value is not numeric"}
	}
	expected := value + request.Delta
	if expected < request.Min {
		expected = request.Min
	}
	if expected > request.Max {
		expected = request.Max
	}
	return adapter.Set(ctx, PropertyRequest{RequestID: request.RequestID, Target: current.Target, Property: request.Property, Value: expected})
}

func (adapter *Adapter) ExecuteScene(ctx context.Context, request SceneRequest) (Result, error) {
	tool, ok := adapter.catalog.tools[roleScene]
	if !ok {
		return Result{}, unsupported("gateway does not expose a schema-compatible scene tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleScene, operationValues{requestID: request.RequestID, target: request.Target, action: "execute"})
	if err != nil {
		return Result{}, err
	}
	return adapter.executeAcknowledged(ctx, tool, arguments, request.Target)
}

func (adapter *Adapter) resolveTarget(ctx context.Context, target Target) (Target, error) {
	if target.ID != "" {
		return target, nil
	}
	tool, ok := adapter.catalog.tools[roleList]
	if !ok {
		return Target{}, unsupported("gateway cannot resolve a natural target name without a node-list tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleList, operationValues{target: target})
	if err != nil {
		return Target{}, err
	}
	result, err := adapter.call(ctx, tool, arguments, false)
	if err != nil {
		return Target{}, err
	}
	return resolveTargetFromData(result.Data, target)
}

func (adapter *Adapter) queryThroughList(ctx context.Context, target Target, property, requestID string) (Result, error) {
	tool, ok := adapter.catalog.tools[roleList]
	if !ok {
		return Result{}, unsupported("gateway does not expose a state or node-list tool")
	}
	arguments, err := buildToolArguments(tool.InputSchema, roleList, operationValues{requestID: requestID, target: target})
	if err != nil {
		return Result{}, err
	}
	result, err := adapter.call(ctx, tool, arguments, false)
	if err != nil {
		return Result{}, err
	}
	if property == "" {
		result.Outcome, result.Target, result.Value = OutcomeReadSuccess, target, result.Data
		return result, nil
	}
	value, found := extractPropertyValue(result.Data, target, property)
	if !found {
		return Result{}, &Error{Kind: ErrorRejected, Stage: "read-back", Message: "gateway node list did not contain the requested property"}
	}
	result.Outcome, result.Target, result.Property, result.Value = OutcomeReadSuccess, target, property, value
	return result, nil
}

func (adapter *Adapter) call(ctx context.Context, tool lanmcp.Tool, arguments map[string]any, write bool) (Result, error) {
	called, err := adapter.client.CallTool(ctx, adapter.session, tool.Name, arguments)
	if err != nil {
		kind := ErrorPreCall
		if write {
			kind = ErrorUncertain
		}
		return Result{Tool: tool.Name}, wrapClientError(kind, "tools/call", err)
	}
	if called.IsError {
		return Result{Tool: tool.Name, Data: called.Data}, &Error{Kind: ErrorRejected, Stage: "tools/call", Message: "gateway rejected the tool call"}
	}
	return Result{Tool: tool.Name, Data: called.Data}, nil
}

func (adapter *Adapter) verifySuccessfulWrite(ctx context.Context, result Result, request PropertyRequest) (Result, error) {
	verified, readErr := adapter.readUntilExpected(ctx, request)
	if readErr != nil {
		result.Outcome = OutcomeUnverified
		result.Evidence = "gateway_ack"
		result.CallError = readErr.Error()
		return result, nil
	}
	result.Value, result.Verified = verified.Value, valuesMatch(verified.Value, request.Value)
	result.Evidence = "state_readback"
	if result.Verified {
		result.Outcome = OutcomeApplied
	} else {
		result.Outcome = OutcomeNotApplied
	}
	return result, nil
}

func (adapter *Adapter) verifyUncertainWrite(ctx context.Context, result Result, request PropertyRequest) (Result, error) {
	verified, readErr := adapter.readUntilExpected(ctx, request)
	if readErr != nil {
		result.Outcome = OutcomeUncertain
		return result, nil
	}
	result.Value, result.Verified = verified.Value, valuesMatch(verified.Value, request.Value)
	result.Evidence = "state_readback"
	if result.Verified {
		result.Outcome = OutcomeApplied
	} else {
		result.Outcome = OutcomeNotApplied
	}
	return result, nil
}

func (adapter *Adapter) readUntilExpected(ctx context.Context, request PropertyRequest) (Result, error) {
	var last Result
	var lastErr error
	for attempt := 0; attempt < adapter.verificationAttempts; attempt++ {
		last, lastErr = adapter.Query(ctx, request)
		if lastErr == nil && valuesMatch(last.Value, request.Value) {
			return last, nil
		}
		if attempt+1 < adapter.verificationAttempts {
			timer := time.NewTimer(adapter.verificationInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return last, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return last, lastErr
}

func isUncertain(err error) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Kind == ErrorUncertain
}

func wrapClientError(kind ErrorKind, stage string, err error) error {
	return &Error{Kind: kind, Stage: stage, Message: fmt.Sprint(err), Cause: err}
}
