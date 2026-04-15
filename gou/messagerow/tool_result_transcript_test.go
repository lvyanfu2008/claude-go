package messagerow

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestTranscriptResolvedHintExtra_readText(t *testing.T) {
	raw := json.RawMessage(`{"type":"text","file":{"filePath":"/x.go","content":"a\nb","numLines":30,"startLine":1,"totalLines":100}}`)
	h, x := TranscriptResolvedHintExtra("Read", raw)
	if h != "Read 30 lines" || x != "" {
		t.Fatalf("h=%q x=%q", h, x)
	}
}

func TestTranscriptResolvedHintExtra_readTextDoubleEncodedJSONString(t *testing.T) {
	inner := `{"type":"text","file":{"filePath":"/x.go","content":"a","numLines":5,"startLine":1,"totalLines":10}}`
	wrapped, err := json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	h, x := TranscriptResolvedHintExtra("Read", json.RawMessage(wrapped))
	if h != "Read 5 lines" || x != "" {
		t.Fatalf("h=%q x=%q", h, x)
	}
}

func TestTranscriptResolvedHintExtra_readTextInferLineCount(t *testing.T) {
	raw := json.RawMessage(`{"type":"text","file":{"filePath":"/x.go","content":"a\nb\nc","numLines":0,"startLine":1,"totalLines":10}}`)
	h, x := TranscriptResolvedHintExtra("Read", raw)
	if h != "Read 3 lines" || x != "" {
		t.Fatalf("h=%q x=%q", h, x)
	}
}

func TestTranscriptResolvedHintExtra_grepContent(t *testing.T) {
	raw := json.RawMessage(`{"mode":"content","numFiles":0,"filenames":[],"content":"p.go:1:foo","numLines":1}`)
	h, x := TranscriptResolvedHintExtra("Search", raw)
	if h != "Found 1 line" || !strings.Contains(x, "p.go:1") {
		t.Fatalf("h=%q x=%q", h, x)
	}
}

func TestCollectToolResultContentByToolUseID(t *testing.T) {
	u, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "z1", "content": `{"mode":"content","numLines":1,"content":"x"}`},
	})
	msgs := []types.Message{{Type: types.MessageTypeUser, Content: u}}
	got := CollectToolResultContentByToolUseID(msgs)
	if len(got["z1"]) == 0 {
		t.Fatal(got)
	}
}

func TestCollectToolResultContentByToolUseID_userMessageFieldOnly(t *testing.T) {
	// toolexecution.CreateUserMessage sets Message.{role,content}, leaves Content empty until NormalizeMessageJSON.
	inner, err := json.Marshal(map[string]any{
		"role": "user",
		"content": []map[string]any{{
			"type": "tool_result", "tool_use_id": "call_1",
			"content":        `{"type":"text","file":{"filePath":"/a.go","content":"x","numLines":42,"startLine":1,"totalLines":99}}`,
			"is_error": false,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs := []types.Message{{Type: types.MessageTypeUser, Message: inner}}
	got := CollectToolResultContentByToolUseID(msgs)
	raw, ok := got["call_1"]
	if !ok || len(raw) == 0 {
		t.Fatalf("expected tool result for call_1, got %v", got)
	}
	h, _ := TranscriptResolvedHintExtra("Read", raw)
	if h != "Read 42 lines" {
		t.Fatalf("hint=%q", h)
	}
}
