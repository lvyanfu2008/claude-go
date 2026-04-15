package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestContentIsEmptyForMerge(t *testing.T) {
	if !contentIsEmptyForMerge(nil) {
		t.Fatal("nil should be empty")
	}
	if !contentIsEmptyForMerge(json.RawMessage(`[]`)) {
		t.Fatal("[] should be empty")
	}
	if !contentIsEmptyForMerge(json.RawMessage(`  []  `)) {
		t.Fatal("whitespace around [] should be empty")
	}
	if !contentIsEmptyForMerge(json.RawMessage(`null`)) {
		t.Fatal("null unmarshals to empty slice for merge")
	}
	if contentIsEmptyForMerge(json.RawMessage(`[{"type":"text","text":"x"}]`)) {
		t.Fatal("non-empty array should not be empty")
	}
}

func TestNormalizeMessageJSON_topLevelContentEmptyArrayMergesMessage(t *testing.T) {
	inner := map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "tool_result", "tool_use_id": "tid1", "content": "ok"},
		},
	}
	msgB, _ := json.Marshal(inner)
	msg := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "u1",
		Content: json.RawMessage(`[]`),
		Message: json.RawMessage(msgB),
	}
	out := NormalizeMessageJSON(msg)
	// Merged content should be the inner array (not []).
	var blocks []map[string]any
	if err := json.Unmarshal(out.Content, &blocks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(blocks) != 1 || blocks[0]["type"] != "tool_result" {
		t.Fatalf("want one tool_result block, got %+v", blocks)
	}
}

func TestNormalizeMessageJSON_preservesContentWhenNonEmpty(t *testing.T) {
	inner := map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "tool_result", "tool_use_id": "other", "content": "x"},
		},
	}
	msgB, _ := json.Marshal(inner)
	orig := `[{"type":"text","text":"keep"}]`
	msg := types.Message{
		Type:    types.MessageTypeUser,
		Content: json.RawMessage(orig),
		Message: json.RawMessage(msgB),
	}
	out := NormalizeMessageJSON(msg)
	if string(out.Content) != orig {
		t.Fatalf("content should stay unchanged, got %q", out.Content)
	}
}
