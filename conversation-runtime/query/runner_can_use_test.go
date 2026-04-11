package query

import (
	"context"
	"encoding/json"
	"testing"

	"goc/conversation-runtime/streamingtool"
	"goc/toolexecution"
	"goc/types"
)

type fakeToolCtxPort struct {
	q *streamingtool.AbortController
}

func (f *fakeToolCtxPort) QueryAbort() *streamingtool.AbortController { return f.q }

func (f *fakeToolCtxPort) SetInProgressToolUseIDs(updater func(map[string]struct{}) map[string]struct{}) {}

func (f *fakeToolCtxPort) SetHasInterruptibleToolInProgress(bool) {}

func TestRunToolUseToolRunner_executorCanUseDeniesBeforeInvoke(t *testing.T) {
	q := streamingtool.NewAbortController()
	port := &fakeToolCtxPort{q: q}
	deny := toolexecution.QueryCanUseToolFn(func(ctx context.Context, n, id string, in json.RawMessage) (toolexecution.PermissionDecision, error) {
		return toolexecution.DenyDecision("from-exec"), nil
	})
	r := RunToolUseToolRunner{
		ParentCtx: context.Background(),
		Deps: toolexecution.ExecutionDeps{
			RandomUUID: func() string { return "r" },
			InvokeTool: func(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
				t.Fatal("invoke should not run after deny")
				return "", false, nil
			},
		},
	}
	tab := streamingtool.NewAbortController()
	ch := r.RunToolUpdates(
		streamingtool.ToolUseBlock{ID: "x", Name: "Any", Input: json.RawMessage(`{}`)},
		types.Message{Type: types.MessageTypeAssistant, UUID: "a"},
		deny,
		port,
		tab,
	)
	var body string
	for u := range ch {
		if u.Message != nil {
			body = string(u.Message.Message)
		}
	}
	if body == "" {
		t.Fatal("expected tool_result message")
	}
}
