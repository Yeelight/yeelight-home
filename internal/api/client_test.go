package api

import (
	"context"
	"testing"
)

func TestClientRejectsOperationNotInRegistry(t *testing.T) {
	client := NewClient(NewStaticRegistry([]Operation{
		{SemanticOperation: "home.summary", Status: "draft", Risk: "R0"},
	}), FixtureTransport{})

	_, err := client.Call(context.Background(), Request{SemanticOperation: "raw.api.call"})
	if err == nil {
		t.Fatal("expected unknown operation to be rejected")
	}
}

func TestClientRejectsDisabledOperation(t *testing.T) {
	client := NewClient(NewStaticRegistry([]Operation{
		{SemanticOperation: "room.create", Status: "disabled_until_phase3", Risk: "R2"},
	}), FixtureTransport{})

	_, err := client.Call(context.Background(), Request{SemanticOperation: "room.create"})
	if err == nil {
		t.Fatal("expected disabled operation to be rejected")
	}
}

func TestClientCallsDraftReadOnlyOperationThroughTransport(t *testing.T) {
	transport := FixtureTransport{
		Responses: map[string]Response{
			"home.summary": {Status: "success", Data: map[string]any{"homes": 1}},
		},
	}
	client := NewClient(NewStaticRegistry([]Operation{
		{SemanticOperation: "home.summary", Status: "draft", Risk: "R0"},
	}), transport)

	response, err := client.Call(context.Background(), Request{SemanticOperation: "home.summary"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if response.Status != "success" {
		t.Fatalf("status = %s", response.Status)
	}
	if response.Data["homes"] != 1 {
		t.Fatalf("data = %#v", response.Data)
	}
}
