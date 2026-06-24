package api

import (
	"context"
	"errors"
	"fmt"
)

type Operation struct {
	SemanticOperation string
	Status            string
	Risk              string
	Method            string
	Path              string
}

type Request struct {
	SemanticOperation string
	Parameters        map[string]any
}

type Response struct {
	Status string
	Data   map[string]any
}

type Registry interface {
	Find(semanticOperation string) (Operation, bool)
}

type Transport interface {
	Call(ctx context.Context, operation Operation, request Request) (Response, error)
}

type Client struct {
	registry  Registry
	transport Transport
}

func NewClient(registry Registry, transport Transport) Client {
	return Client{registry: registry, transport: transport}
}

func (client Client) Call(ctx context.Context, request Request) (Response, error) {
	operation, ok := client.registry.Find(request.SemanticOperation)
	if !ok {
		return Response{}, fmt.Errorf("operation %q is not in registry", request.SemanticOperation)
	}
	if operation.Status != "draft" && operation.Status != "enabled" {
		return Response{}, fmt.Errorf("operation %q is not enabled: %s", request.SemanticOperation, operation.Status)
	}
	return client.transport.Call(ctx, operation, request)
}

type StaticRegistry struct {
	operations map[string]Operation
}

func NewStaticRegistry(operations []Operation) StaticRegistry {
	registry := StaticRegistry{operations: map[string]Operation{}}
	for _, operation := range operations {
		registry.operations[operation.SemanticOperation] = operation
	}
	return registry
}

func (registry StaticRegistry) Find(semanticOperation string) (Operation, bool) {
	operation, ok := registry.operations[semanticOperation]
	return operation, ok
}

type FixtureTransport struct {
	Responses map[string]Response
}

func (transport FixtureTransport) Call(_ context.Context, operation Operation, _ Request) (Response, error) {
	response, ok := transport.Responses[operation.SemanticOperation]
	if !ok {
		return Response{}, errors.New("fixture response is missing")
	}
	return response, nil
}
