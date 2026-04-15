package messagerow

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestToolChromeParts_grep(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"pattern": "src/tools/*", "path": "/tmp"})
	f, p, h := ToolChromeParts("Grep", raw)
	if f != "Search" || !strings.Contains(p, "pattern:") || h != `"src/tools/*"` {
		t.Fatalf("f=%q p=%q h=%q", f, p, h)
	}
}

func TestToolChromeParts_readLinesRange(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"file_path": "/Users/lvyanfu/Work/claude/claude-go/conversation-runtime/query/streaming_parity_test.go",
		"offset":    1,
		"limit":     30,
	})
	f, p, h := ToolChromeParts("Read", raw)
	if f != "Read" {
		t.Fatalf("f=%q", f)
	}
	if !strings.Contains(p, "· lines 1-30") {
		t.Fatalf("paren=%q want · lines 1-30", p)
	}
	if h == "" {
		t.Fatal("want hint path")
	}
}

func TestCollectResolvedToolUseIDs(t *testing.T) {
	u, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "a1", "content": "ok"},
	})
	msgs := []types.Message{{Type: types.MessageTypeUser, Content: u}}
	got := CollectResolvedToolUseIDs(msgs)
	if _, ok := got["a1"]; !ok {
		t.Fatal(got)
	}
}
