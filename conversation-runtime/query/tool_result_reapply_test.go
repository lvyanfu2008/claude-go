package query

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestReapplyToolResultReplacementsFromState(t *testing.T) {
	inner, _ := json.Marshal(map[string]any{
		"role": "user",
		"content": []any{
			map[string]any{"type": "tool_result", "tool_use_id": "tu_1", "content": "HUGE"},
		},
	})
	ms := []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Message: inner},
	}
	state := json.RawMessage(`{"replacements":{"tu_1":"<<preview>>"},"seenIds":["tu_1"]}`)
	got := ReapplyToolResultReplacementsFromState(ms, state)
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
	var env struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(got[0].Message, &env); err != nil {
		t.Fatal(err)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(env.Content, &blocks); err != nil {
		t.Fatal(err)
	}
	if blocks[0]["content"] != "<<preview>>" {
		t.Fatalf("%#v", blocks[0])
	}
}
