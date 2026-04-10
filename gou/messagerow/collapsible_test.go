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
	msg := types.Message{
		Type:           types.MessageTypeCollapsedReadSearch,
		UUID:           "c1",
		ReadCount:      3,
		SearchCount:    1,
		ReadFilePaths:  []string{"a.go"},
		DisplayMessage: &disp,
	}
	segs := SegmentsFromMessage(msg)
	if segs[0].Kind != SegCollapsedReadSearch {
		t.Fatalf("%+v", segs[0])
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
