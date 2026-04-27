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

func TestRepairToolUseToolResultPairing_recognizesUserWhenOnlyContentToolResult(t *testing.T) {
	// Omitted: type, message — only `content` array (common bad export from tooling).
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "t_only", "name": "x", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "t_only", "name": "x", "input": map[string]any{}}}),
	}
	u := types.Message{
		UUID: "u1",
		Content: mustJSON(t, []any{
			map[string]any{"type": "tool_result", "tool_use_id": "t_only", "content": "ok"},
		}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{asst, u})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2, types=%v", len(out), messageTypesForTest(out))
	}
}

func TestRepairToolUseToolResultPairing_recognizesUserWhenTopLevelTypeEmpty(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "call_00", "name": "x", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "call_00", "name": "x", "input": map[string]any{}}}),
	}
	// Some JSONL / hydrate paths omit `type` and only set nested message.role.
	inner := map[string]any{
		"role":    "user",
		"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "call_00", "content": "ok"}},
	}
	uWire := mustJSON(t, inner)
	u := types.Message{
		UUID:    "u1",
		Message: uWire,
		Content: mustJSON(t, []any{map[string]any{"type": "tool_result", "tool_use_id": "call_00", "content": "ok"}}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{asst, u})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2 (no synthetic), got types %v", len(out), messageTypesForTest(out))
	}
}

func TestRepairToolUseToolResultPairing_lenientWeirdTypeWithDeepToolResult(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "tid", "name": "n", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "tid", "name": "n", "input": map[string]any{}}}),
	}
	// Unusual top-level "type" that is not in our heuristics; tool_result only discoverable by deep JSON walk.
	weird := types.Message{
		Type:  types.MessageType("custom_row"),
		UUID:  "w1",
		Content: mustJSON(t, []any{
			map[string]any{"type": "tool_result", "tool_use_id": "tid", "content": "x"},
		}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{asst, weird})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d want 2, got %v", len(out), messageTypesForTest(out))
	}
}

func messageTypesForTest(msgs []types.Message) []string {
	s := make([]string, 0, len(msgs))
	for i := range msgs {
		s = append(s, string(msgs[i].Type))
	}
	return s
}

func TestRepairToolUseToolResultPairing_skipsProgressBeforeUser(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role":    "assistant",
			"content": []any{map[string]any{"type": "tool_use", "id": "call_00", "name": "x", "input": map[string]any{}}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_use", "id": "call_00", "name": "x", "input": map[string]any{}}}),
	}
	prog := types.Message{Type: types.MessageTypeProgress, UUID: "p1"}
	u1 := types.Message{
		Type: types.MessageTypeUser,
		UUID: "u1",
		Message: mustJSON(t, map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "call_00", "content": "ok"}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_result", "tool_use_id": "call_00", "content": "ok"}}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{asst, prog, u1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("len=%d want 3 (no synthetic)", len(out))
	}
	if out[1].Type != types.MessageTypeProgress {
		t.Fatalf("order: want progress at 1, got %q", out[1].Type)
	}
	inner, err := getInner(&out[2])
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) != 1 {
		t.Fatalf("user blocks=%v", blocks)
	}
}

func TestRepairToolUseToolResultPairing_noopWhenToolResultsSplitAcrossUsers(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{"type": "tool_use", "id": "call_01", "name": "x", "input": map[string]any{}},
				map[string]any{"type": "tool_use", "id": "call_02", "name": "y", "input": map[string]any{}},
			},
		}),
		Content: mustJSON(t, []any{
			map[string]any{"type": "tool_use", "id": "call_01", "name": "x", "input": map[string]any{}},
			map[string]any{"type": "tool_use", "id": "call_02", "name": "y", "input": map[string]any{}},
		}),
	}
	u1 := types.Message{
		Type: types.MessageTypeUser,
		UUID: "u1",
		Message: mustJSON(t, map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "call_01", "content": "a"}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_result", "tool_use_id": "call_01", "content": "a"}}),
	}
	u2 := types.Message{
		Type: types.MessageTypeUser,
		UUID: "u2",
		Message: mustJSON(t, map[string]any{
			"role":    "user",
			"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "call_02", "content": "b"}},
		}),
		Content: mustJSON(t, []any{map[string]any{"type": "tool_result", "tool_use_id": "call_02", "content": "b"}}),
	}
	out, err := RepairToolUseToolResultPairing([]types.Message{asst, u1, u2})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("len=%d want 3 (no synthetic rows)", len(out))
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
		t.Fatalf("u1 should stay single block, got %d", len(blocks))
	}
}

func TestUserMessagesCoverAssistantToolUses(t *testing.T) {
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "a1",
		Message: mustJSON(t, map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{"type": "tool_use", "id": "t1", "name": "x", "input": map[string]any{}},
				map[string]any{"type": "tool_use", "id": "t2", "name": "y", "input": map[string]any{}},
			},
		}),
		Content: mustJSON(t, []any{
			map[string]any{"type": "tool_use", "id": "t1", "name": "x", "input": map[string]any{}},
			map[string]any{"type": "tool_use", "id": "t2", "name": "y", "input": map[string]any{}},
		}),
	}
	u1 := types.Message{
		Type: types.MessageTypeUser,
		UUID: "u1",
		Message: mustJSON(t, map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "ok"},
			},
		}),
	}
	if UserMessagesCoverAssistantToolUses(asst, []types.Message{u1}) {
		t.Fatal("expected false with one of two ids")
	}
	u2 := types.Message{
		Type: types.MessageTypeUser,
		UUID: "u2",
		Message: mustJSON(t, map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t2", "content": "ok"},
			},
		}),
	}
	if !UserMessagesCoverAssistantToolUses(asst, []types.Message{u1, u2}) {
		t.Fatal("expected true with both user messages")
	}
	// Both in one user message
	combined := types.Message{
		Type: types.MessageTypeUser,
		UUID: "uc",
		Message: mustJSON(t, map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "a"},
				map[string]any{"type": "tool_result", "tool_use_id": "t2", "content": "b"},
			},
		}),
	}
	if !UserMessagesCoverAssistantToolUses(asst, []types.Message{combined}) {
		t.Fatal("expected true with combined user")
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
