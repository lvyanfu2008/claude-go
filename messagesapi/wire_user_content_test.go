package messagesapi

import (
	"encoding/json"
	"testing"
)

func TestUserAllTextContentAsJSONString_joinsTextBlocks(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal([]map[string]any{
		{"type": "text", "text": "a"},
		{"type": "text", "text": "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, ok := UserAllTextContentAsJSONString(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	var s string
	if err := json.Unmarshal(out, &s); err != nil {
		t.Fatal(err)
	}
	if s != "ab" {
		t.Fatalf("got %q", s)
	}
}

func TestUserAllTextContentAsJSONString_rejectsNonText(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal([]map[string]any{
		{"type": "text", "text": "a"},
		{"type": "tool_result", "tool_use_id": "x", "content": "y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := UserAllTextContentAsJSONString(raw); ok {
		t.Fatal("expected not ok")
	}
}

func TestUserAllTextContentAsJSONString_alreadyString(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal("hello")
	if _, ok := UserAllTextContentAsJSONString(json.RawMessage(raw)); ok {
		t.Fatal("expected not ok")
	}
}
