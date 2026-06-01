package message

import (
	"testing"

	"goc/types"
)

func TestBuildMessageRowContexts_Empty(t *testing.T) {
	ctxs := BuildMessageRowContexts(nil, MessageRowBuildOpts{})
	if len(ctxs) != 0 {
		t.Fatalf("expected 0 contexts, got %d", len(ctxs))
	}
}

func TestBuildMessageRowContexts_UserContinuation(t *testing.T) {
	msgs := []*types.Message{
		{Type: types.MessageTypeUser, UUID: "1"},
		{Type: types.MessageTypeUser, UUID: "2"},
		{Type: types.MessageTypeAssistant, UUID: "3"},
	}
	ctxs := BuildMessageRowContexts(msgs, MessageRowBuildOpts{})
	if len(ctxs) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(ctxs))
	}
	if !ctxs[1].IsUserContinuation {
		t.Error("second user message should be continuation")
	}
	if ctxs[2].IsUserContinuation {
		t.Error("assistant after user should not be continuation")
	}
}

func TestBuildMessageRowContexts_IsInProgress(t *testing.T) {
	msgs := []*types.Message{
		{Type: types.MessageTypeAssistant, UUID: "1", Content: []byte(
			`[{"type":"tool_use","id":"tu_1","name":"Read","input":{}}]`,
		)},
	}
	// With resolved
	resolved := map[string]struct{}{"tu_1": {}}
	ctxs := BuildMessageRowContexts(msgs, MessageRowBuildOpts{
		ResolvedToolUseIDs: resolved,
	})
	if len(ctxs) != 1 {
		t.Fatalf("expected 1 context, got %d", len(ctxs))
	}
	if ctxs[0].IsInProgress {
		t.Error("tool_use with resolved result should not be in progress")
	}

	// Without resolved
	ctxs2 := BuildMessageRowContexts(msgs, MessageRowBuildOpts{
		ResolvedToolUseIDs: map[string]struct{}{},
	})
	if !ctxs2[0].IsInProgress {
		t.Error("tool_use without resolved result should be in progress")
	}
}
