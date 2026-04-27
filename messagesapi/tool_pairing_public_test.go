package messagesapi

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestUserMessagesCoverAssistantToolUses_basic(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		Message: mustJSONTool(t, map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "t1", "name": "n", "input": map[string]any{}}}}),
		Content: mustJSONTool(t, []any{map[string]any{"type": "tool_use", "id": "t1", "name": "n", "input": map[string]any{}}}),
	}
	u := types.Message{
		Type:    types.MessageTypeUser,
		Message: mustJSONTool(t, map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "ok"}}}),
		Content: mustJSONTool(t, []any{map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "ok"}}),
	}
	if !UserMessagesCoverAssistantToolUses(asst, []types.Message{u}) {
		t.Fatal("expected cover")
	}
}

func mustJSONTool(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
