package messagerow

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestSegments_toolUseAndText(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{
		{"type": "text", "text": "Hello"},
		{"type": "tool_use", "id": "x1", "name": "Bash", "input": map[string]any{"command": "ls"}},
	})
	msg := types.Message{Type: types.MessageTypeAssistant, Content: raw}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 2 {
		t.Fatalf("got %d segments", len(segs))
	}
	if segs[0].Kind != SegTextMarkdown || segs[1].Kind != SegToolUse {
		t.Fatalf("%+v", segs)
	}
	// TS chrome: activity in Text; Bash facing name + paren summary.
	if segs[1].ToolFacing != "Bash" || !strings.Contains(segs[1].Text, "Running") || !strings.Contains(segs[1].Text, "ls") {
		t.Fatalf("%+v text=%q facing=%q paren=%q", segs[1], segs[1].Text, segs[1].ToolFacing, segs[1].ToolParen)
	}
}

func TestSegments_toolResult(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "x1", "content": "ok\noutput"},
	})
	msg := types.Message{Type: types.MessageTypeUser, Content: raw}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 1 || segs[0].Kind != SegToolResult {
		t.Fatalf("%+v", segs)
	}
}

func TestSegments_assistantEmptyTextBlocksShowsPlaceholder(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{
		{"type": "text", "text": ""},
	})
	msg := types.Message{Type: types.MessageTypeAssistant, Content: raw}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 1 || segs[0].Kind != SegTextMarkdown {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "no visible text") {
		t.Fatal(segs[0].Text)
	}
}
