package messagesapi

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestRepairToolUseToolResultPairing_insertsUserWhenMissing(t *testing.T) {
	assistant := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "asst-1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "tool-abc", "name": "mcp", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "tool-abc", "name": "mcp", "input": map[string]any{}}}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{assistant})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	if out[1].Type != types.MessageTypeUser {
		t.Fatalf("second message type=%q", out[1].Type)
	}
	inner, err := getInner(&out[1])
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks len=%d", len(blocks))
	}
	if blocks[0]["type"] != "tool_result" {
		t.Fatalf("block type=%v", blocks[0]["type"])
	}
	if blocks[0]["tool_use_id"] != "tool-abc" {
		t.Fatalf("tool_use_id=%v", blocks[0]["tool_use_id"])
	}
}

func TestRepairToolUseToolResultPairing_patchesUserWhenPartial(t *testing.T) {
	assistant := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "asst-1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "a", "name": "n", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "a", "name": "n", "input": map[string]any{}}}),
	}
	user := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "u-1",
		Message: mustJSON(t, map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "text", "text": "x"}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "text", "text": "x"}}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{assistant, user})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	inner, err := getInner(&out[1])
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) < 2 {
		t.Fatalf("want >=2 blocks, got %d", len(blocks))
	}
	found := false
	for _, b := range blocks {
		if t, _ := b["type"].(string); t == "tool_result" {
			if b["tool_use_id"] == "a" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("no tool_result for id a: %#v", blocks)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
