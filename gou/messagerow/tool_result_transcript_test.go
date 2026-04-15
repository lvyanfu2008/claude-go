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
