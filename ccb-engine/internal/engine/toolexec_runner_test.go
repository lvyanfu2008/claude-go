package engine

import (
	"context"
	"encoding/json"
	"testing"

	"goc/toolexecution"
)

func TestToolexecutionRunner_echoStub(t *testing.T) {
	tools := json.RawMessage(`[{"name":"echo_stub","input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]`)
	reg, err := toolexecution.NewJSONToolRegistry(tools)
	if err != nil {
		t.Fatal(err)
	}
	r := ToolexecutionRunner{Deps: toolexecution.ExecutionDeps{
		RandomUUID: func() string { return "r1" },
		Registry:   reg,
	}}
	out, isErr, err := r.Run(context.Background(), "echo_stub", "tu1", json.RawMessage(`{"message":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if isErr {
		t.Fatalf("isErr=%v out=%q", isErr, out)
	}
	if out == "" {
		t.Fatal("empty out")
	}
}

func TestToolexecutionRunner_queryDeny(t *testing.T) {
	r := ToolexecutionRunner{Deps: toolexecution.ExecutionDeps{
		RandomUUID: func() string { return "r1" },
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
			return toolexecution.DenyDecision("nope"), nil
		},
		InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
			return "should-not-run", false, nil
		},
	}}
	out, _, err := r.Run(context.Background(), "Any", "tu1", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "should-not-run" {
		t.Fatal("invoke ran after deny")
	}
}
