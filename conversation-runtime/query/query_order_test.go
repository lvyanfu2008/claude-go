package query

import (
	"context"
	"encoding/json"
	"testing"

	"goc/types"
)

// Ensures prependUserContext runs after microcompact (TS query.ts order).
func TestQueryPrependsUserContextAfterMicrocompact(t *testing.T) {
	ctx := context.Background()
	var microLen int
	deps := ProductionDeps()
	deps.Microcompact = func(ctx context.Context, in *MicrocompactInput) (*MicrocompactResult, error) {
		microLen = len(in.Messages)
		return &MicrocompactResult{Messages: in.Messages}, nil
	}
	deps.CallModel = func(ctx context.Context, in *CallModelInput, emit func(QueryYield) bool) error {
		if len(in.Messages) != 2 {
			t.Fatalf("CallModel expected 2 messages (meta + user), got %d", len(in.Messages))
		}
		if in.Messages[0].IsMeta == nil || !*in.Messages[0].IsMeta {
			t.Fatalf("first message should be meta user-context %#v", in.Messages[0])
		}
		return nil
	}
	p := QueryParams{
		Messages: []types.Message{
			{Type: types.MessageTypeUser, UUID: "u1", Message: json.RawMessage(`{"role":"user","content":"hi"}`)},
		},
		SystemPrompt: AsSystemPrompt([]string{"sys"}),
		UserContext:  map[string]string{"cwd": "/proj"},
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{},
		},
		Deps: &deps,
	}
	for range Query(ctx, p) {
	}
	if microLen != 1 {
		t.Fatalf("microcompact saw %d messages, want 1 without prepended context", microLen)
	}
}
