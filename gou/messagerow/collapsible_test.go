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
		Type:     types.MessageTypeGroupedToolUse,
		UUID:     "g1",
		ToolName: "Agent",
		Messages: []types.Message{
			{
				Type:    types.MessageTypeAssistant,
				UUID:    "m1",
				Content: []byte(`[{"type":"tool_use","id":"123","input":{"name":"worker"}}]`),
			},
		},
		Results:        []types.Message{{Type: types.MessageTypeUser, UUID: "r1"}},
		DisplayMessage: &disp,
	}
	segs := SegmentsFromMessage(msg)
	if len(segs) < 2 || segs[0].Kind != SegGroupedToolUse {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "Running 1 agents…") {
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
	// Past-tense read clause is capitalized after the comma for readability.
	if !strings.Contains(segs[0].Text, "Read 3 files") || !strings.Contains(segs[0].Text, "Searched for 1 pattern") {
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
	// TS: ⎿ latestDisplayHint only when isActiveGroup; default opts => inactive.
	for i := 1; i < len(segs); i++ {
		if segs[i].Kind == SegDisplayHint && strings.Contains(segs[i].Text, "⎿") {
			t.Fatalf("inactive collapsed group should omit ⎿ row, got %+v", segs)
		}
	}
}

func TestSegments_collapsedReadSearch_activeShowsHintRow(t *testing.T) {
	disp := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "d2",
		Content: []byte(`[{"type":"text","text":"tail"}]`),
	}
	h := "pattern-hint"
	msg := types.Message{
		Type:              types.MessageTypeCollapsedReadSearch,
		UUID:              "c1",
		ReadCount:         1,
		SearchCount:       1,
		LatestDisplayHint: &h,
		DisplayMessage:    &disp,
	}
	segs := SegmentsFromMessageOpts(msg, &RenderOpts{CollapsedReadSearchActive: true})
	found := false
	for _, s := range segs {
		if s.Kind == SegDisplayHint && strings.Contains(s.Text, "⎿") && strings.Contains(s.Text, "pattern-hint") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("active group should show ⎿ hint, segs=%+v", segs)
	}
}

func TestSegments_collapsedReadSearch_verboseTranscriptGrepPlusReadNestedRollupOneLine(t *testing.T) {
	grepContent, _ := json.Marshal([]map[string]any{{
		"type": "tool_use", "id": "g1", "name": "Grep",
		"input": map[string]any{"pattern": "foo", "path": "/p"},
	}})
	readContent, _ := json.Marshal([]map[string]any{{
		"type": "tool_use", "id": "r1", "name": "Read",
		"input": map[string]any{"file_path": "/proj/a.go"},
	}})
	msg := types.Message{
		Type:        types.MessageTypeCollapsedReadSearch,
		UUID:        "c2",
		SearchCount: 1,
		ReadCount:   1,
		Messages: []types.Message{
			{Type: types.MessageTypeAssistant, UUID: "a1", Content: grepContent},
			{Type: types.MessageTypeAssistant, UUID: "a2", Content: readContent},
		},
	}
	segs := SegmentsFromMessageOpts(msg, &RenderOpts{
		VerboseCollapsedReadSearch: true,
		TranscriptMode:             true,
	})
	if len(segs) < 1 || segs[0].Kind != SegCollapsedReadSearch {
		t.Fatalf("want rollup first, got %+v", segs)
	}
	want := "Searched for 1 pattern, Read 1 file"
	if strings.TrimSpace(segs[0].Text) != want {
		t.Fatalf("want %q, got %q", want, segs[0].Text)
	}
	for _, s := range segs {
		if s.Kind == SegToolUseSummaryLine {
			t.Fatalf("should not emit nested SegToolUseSummaryLine, got %+v", segs)
		}
	}
}

func TestSegments_collapsedReadSearch_verboseTranscriptRollupOneLineAndSkipsNestedSummaryLines(t *testing.T) {
	raw, _ := json.Marshal([]map[string]any{{
		"type": "tool_use", "id": "tu1", "name": "Read",
		"input": map[string]any{"file_path": "/proj/doc.go"},
	}})
	nested := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "na",
		Content: raw,
	}
	msg := types.Message{
		Type:      types.MessageTypeCollapsedReadSearch,
		UUID:      "c1",
		ReadCount: 1,
		Messages:  []types.Message{nested},
	}
	segs := SegmentsFromMessageOpts(msg, &RenderOpts{
		VerboseCollapsedReadSearch: true,
		TranscriptMode:             true,
	})
	var rollup *Segment
	for i := range segs {
		if segs[i].Kind == SegCollapsedReadSearch {
			rollup = &segs[i]
			break
		}
	}
	if rollup == nil || !strings.Contains(rollup.Text, "Read 1 file") {
		t.Fatalf("want one collapsed_read_search rollup with aggregate summary, got %+v", segs)
	}
	for _, s := range segs {
		if s.Kind == SegToolUseSummaryLine {
			t.Fatalf("nested per-tool summary lines should be merged into rollup, got %+v", segs)
		}
	}
}

func TestSegments_collapsedReadSearch_showAllExpandsPaths(t *testing.T) {
	disp := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "d3",
		Content: []byte(`[{"type":"text","text":"tail"}]`),
	}
	hint := "hint"
	msg := types.Message{
		Type:              types.MessageTypeCollapsedReadSearch,
		UUID:              "c2",
		ReadCount:         2,
		SearchCount:       1,
		ReadFilePaths:     []string{"a.go", "b.go"},
		DisplayMessage:    &disp,
		LatestDisplayHint: &hint,
	}
	segs := SegmentsFromMessageOpts(msg, &RenderOpts{ShowAllInTranscript: true})
	if segs[0].Kind != SegCollapsedReadSearch || !strings.Contains(segs[0].Text, "Files:") {
		t.Fatalf("want Files block in show-all, got %+v", segs)
	}
	if strings.Contains(segs[0].Text, CtrlOToExpandHint) {
		t.Fatalf("show-all first segment should omit ctrl+o hint, got %q", segs[0].Text)
	}
}

func TestSegments_groupedToolUse_showAllInlinesNested(t *testing.T) {
	nested := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "na",
		Content: []byte(`[{"type":"text","text":"inner"}]`),
	}
	res := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "nu",
		Content: []byte(`[{"type":"text","text":"ok"}]`),
	}
	disp := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "d4",
		Content: []byte(`[{"type":"text","text":"Hi"}]`),
	}
	msg := types.Message{
		Type:           types.MessageTypeGroupedToolUse,
		UUID:           "g2",
		ToolName:       "Read",
		Messages:       []types.Message{nested},
		Results:        []types.Message{res},
		DisplayMessage: &disp,
	}
	segs := SegmentsFromMessageOpts(msg, &RenderOpts{ShowAllInTranscript: true})
	if len(segs) < 3 {
		t.Fatalf("want nested segments, got %d: %+v", len(segs), segs)
	}
	found := false
	for _, s := range segs {
		if s.Kind == SegTextMarkdown && strings.Contains(s.Text, "inner") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected nested assistant text, got %+v", segs)
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
