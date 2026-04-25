package messagesapi

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestFilterOrphanedThinkingOnlyMessages_PrependsIntoFollowingAssistant(t *testing.T) {
	thinkingOnly := types.Message{
		Type: types.MessageTypeAssistant,
		Message: mustJSONMarshal(t, map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{"type": "thinking", "thinking": "chain step"},
			},
		}),
	}
	toolAssistant := types.Message{
		Type: types.MessageTypeAssistant,
		Message: mustJSONMarshal(t, map[string]any{
			"role": "assistant",
			"id":   "msg_next",
			"content": []map[string]any{
				{
					"type":  "tool_use",
					"id":    "toolu_1",
					"name":  "bash",
					"input": map[string]any{"command": "ls"},
				},
			},
		}),
	}
	out, err := filterOrphanedThinkingOnlyMessages([]types.Message{thinkingOnly, toolAssistant})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 merged assistant, got %d", len(out))
	}
	inner, err := getInner(&out[0])
	if err != nil {
		t.Fatal(err)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(inner.Content, &blocks); err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks: %+v", blocks)
	}
	if typ, _ := blocks[0]["type"].(string); typ != "thinking" {
		t.Fatalf("first block type: %q", typ)
	}
	if typ, _ := blocks[1]["type"].(string); typ != "tool_use" {
		t.Fatalf("second block type: %q", typ)
	}
}

func TestFilterOrphanedThinkingOnlyMessages_KeepsWhenFollowedByUser(t *testing.T) {
	thinkingOnly := types.Message{
		Type: types.MessageTypeAssistant,
		Message: mustJSONMarshal(t, map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{"type": "thinking", "thinking": "prior reasoning"},
			},
		}),
	}
	userAfter := types.Message{
		Type: types.MessageTypeUser,
		Message: mustJSONMarshal(t, map[string]any{
			"role":    "user",
			"content": []map[string]any{{"type": "text", "text": "next turn"}},
		}),
	}
	out, err := filterOrphanedThinkingOnlyMessages([]types.Message{thinkingOnly, userAfter})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 messages (keep thinking assistant + user), got %d", len(out))
	}
	if out[0].Type != types.MessageTypeAssistant {
		t.Fatalf("first: want assistant, got %v", out[0].Type)
	}
	inner, err := getInner(&out[0])
	if err != nil {
		t.Fatal(err)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(inner.Content, &blocks); err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || blocks[0]["type"] != "thinking" {
		t.Fatalf("first msg blocks: %#v", blocks)
	}
}

func TestFilterTrailingThinkingFromLastAssistant_SkipWhenOpenAIOpt(t *testing.T) {
	msg := types.Message{
		Type: types.MessageTypeAssistant,
		Message: mustJSONMarshal(t, map[string]any{
			"role": "assistant",
			"content": []map[string]any{
				{"type": "text", "text": "hi"},
				{"type": "thinking", "thinking": "do not drop"},
			},
		}),
	}
	out, err := filterTrailingThinkingFromLastAssistant([]types.Message{msg}, Options{SkipStripTrailingThinking: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len: %d", len(out))
	}
	inner, err := getInner(&out[0])
	if err != nil {
		t.Fatal(err)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(inner.Content, &blocks); err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("want 2 blocks preserved, got %d: %#v", len(blocks), blocks)
	}
}

func mustJSONMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
