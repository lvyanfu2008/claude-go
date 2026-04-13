package messagerow

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestSegments_groupedToolUse(t *testing.T) {
	disp := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "d1",
		Content: []byte(`[{"type":"text","text":"Hi"}]`),
	}
	msg := types.Message{
		Type:           types.MessageTypeGroupedToolUse,
		UUID:           "g1",
		ToolName:       "Read",
		Messages:       []types.Message{{Type: types.MessageTypeAssistant, UUID: "m1"}},
		Results:        []types.Message{{Type: types.MessageTypeUser, UUID: "r1"}},
		DisplayMessage: &disp,
	}
	segs := SegmentsFromMessage(msg)
	if len(segs) < 2 || segs[0].Kind != SegGroupedToolUse {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "Read") {
		t.Fatal(segs[0].Text)
	}
}

func TestSegments_collapsedReadSearch(t *testing.T) {
	disp := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "d2",
		Content: []byte(`[{"type":"text","text":"tail"}]`),
	}
	h := "ls -la\nbuild"
	msg := types.Message{
		Type:              types.MessageTypeCollapsedReadSearch,
		UUID:              "c1",
		ReadCount:         3,
		SearchCount:       1,
		ReadFilePaths:     []string{"a.go"},
		LatestDisplayHint: &h,
		DisplayMessage:    &disp,
	}
	segs := SegmentsFromMessage(msg)
	if segs[0].Kind != SegCollapsedReadSearch {
		t.Fatalf("%+v", segs[0])
	}
	// TS uses lowercase continuation verbs after the first clause ("searched for …, read …").
	if !strings.Contains(segs[0].Text, "read 3 files") || !strings.Contains(segs[0].Text, "Searched for 1 pattern") {
		t.Fatalf("want TS-style summary (search then read), got %q", segs[0].Text)
	}
	if !strings.HasPrefix(segs[0].Text, "Searched for 1 pattern") {
		t.Fatalf("TS order: search clause first, got %q", segs[0].Text)
	}
	if strings.Contains(segs[0].Text, "collapsed_read_search") {
		t.Fatalf("should not use debug prefix: %q", segs[0].Text)
	}
	if !strings.Contains(segs[0].Text, CtrlOToExpandHint) {
		t.Fatalf("want ctrl+o hint: %q", segs[0].Text)
	}
	if len(segs) < 3 || segs[1].Kind != SegDisplayHint || !strings.Contains(segs[1].Text, "⎿") {
		t.Fatalf("want hint row after summary, got %+v", segs)
	}
}

func TestSegments_serverAndAdvisor(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{
		{"type": "server_tool_use", "id": "s1", "name": "N", "input": map[string]any{"x": 1}},
		{"type": "advisor_tool_result", "id": "a1", "content": "hint"},
	})
	msg := types.Message{Type: types.MessageTypeAssistant, Content: raw}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 2 || segs[0].Kind != SegServerToolUse || segs[1].Kind != SegAdvisorToolResult {
		t.Fatalf("%+v", segs)
	}
}
