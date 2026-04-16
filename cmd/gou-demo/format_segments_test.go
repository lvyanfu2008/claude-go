package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"goc/gou/conversation"
	"goc/gou/messagerow"
	"goc/types"
)

func countToolLeadGlyphs(s string) int {
	// toolRowLeadPrefix uses ⏺ on darwin, ● elsewhere (before lipgloss; output keeps runes).
	return strings.Count(s, "\u23fa") + strings.Count(s, "\u25cf")
}

func TestFormatMessageSegments_mergedSearchReadSummaryOneCommaLine(t *testing.T) {
	segs := []messagerow.Segment{
		{Kind: messagerow.SegTextMarkdown, Text: "现在让我查看 UseOpenAIChatProvider 函数："},
		{Kind: messagerow.SegToolUseSummaryLine, Text: "Searched for 1 pattern, Read 1 file", ToolUseIDs: []string{"g1", "r1"}},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "", nil, false)
	if strings.Count(out, "Searched for 1 pattern") != 1 || strings.Count(out, "Read 1 file") != 1 {
		t.Fatalf("want single line containing both clauses, got:\n%s", out)
	}
	if !strings.Contains(out, "\n\n") {
		t.Fatalf("want blank line between assistant text and summary, got:\n%s", out)
	}
	if !strings.Contains(out, "  ") || !strings.Contains(out, "Searched for 1 pattern") {
		t.Fatalf("want two-space indent before summary line, got:\n%s", out)
	}
	idx := strings.Index(out, "Searched for 1 pattern")
	if idx < 2 || out[idx-1] != ' ' || out[idx-2] != ' ' {
		t.Fatalf("summary should start after exactly two spaces, got:\n%s", out)
	}
	if !strings.Contains(out, "ctrl+o to expand") {
		t.Fatalf("want ctrl+o when unresolved, got:\n%s", out)
	}
}

func TestFormatMessageSegments_singleLeadGlyphAfterAssistantText(t *testing.T) {
	segs := []messagerow.Segment{
		{Kind: messagerow.SegTextMarkdown, Text: "我来查看一下项目结构和主要业务。"},
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "README.md", ToolHint: "README.md", Text: "Reading README", ToolUseID: "call_x"},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "", nil, false)
	if n := countToolLeadGlyphs(out); n != 1 {
		t.Fatalf("want exactly one ⏺/● lead (on assistant text only), got %d in:\n%s", n, out)
	}
	if !strings.Contains(out, "Read") {
		t.Fatalf("missing tool title: %s", out)
	}
}

func TestFormatMessageSegments_leadGlyphOnToolWhenNoPriorText(t *testing.T) {
	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "README.md", ToolHint: "README.md", Text: "Reading README", ToolUseID: "call_y"},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "", nil, false)
	if n := countToolLeadGlyphs(out); n < 1 {
		t.Fatalf("tool-only message should keep lead on tool row, got %d in:\n%s", n, out)
	}
}

func TestFormatMessageSegments_searchHighlightInsertsANSI(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, Text: "alpha BETA gamma", ToolUseID: "id1"},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "beta", nil, false)
	if !strings.Contains(out, "BETA") && !strings.Contains(out, "beta") {
		t.Fatalf("expected original casing preserved in visible text: %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected lipgloss ANSI from search highlight + styles, got %q", out)
	}
}

func TestFormatMessageSegments_searchHighlightEmptyNeedleNoExtraFromHL(t *testing.T) {
	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, Text: "plain tool title", ToolUseID: "id2"},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "", nil, false)
	outHL := formatMessageSegments(segs, 80, true, nil, true, "   ", nil, false)
	if out != outHL {
		t.Fatalf("whitespace-only searchHL should behave like no highlight: %q vs %q", out, outHL)
	}
}

func TestFormatMessageSegments_resolvedToolOmitsActivityAndHint(t *testing.T) {
	resolved := map[string]struct{}{"call_z": {}}
	segs := []messagerow.Segment{
		{Kind: messagerow.SegTextMarkdown, Text: "hello"},
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "a.md", ToolHint: "a.md", Text: "Reading a", ToolUseID: "call_z"},
	}
	out := formatMessageSegments(segs, 80, true, resolved, true, "", nil, false)
	if strings.Contains(out, "⎿") {
		t.Fatalf("resolved tool should not render ⎿ row, got:\n%s", out)
	}
	if strings.Contains(out, "Reading") {
		t.Fatalf("resolved tool should not render activity line, got:\n%s", out)
	}
}

func TestFormatMessageSegments_transcriptStatsWhenResolvedIDMapEmptyButToolResultPresent(t *testing.T) {
	resultJSON := json.RawMessage(`{"type":"text","file":{"filePath":"/x.go","content":"x","numLines":12,"startLine":1,"totalLines":20}}`)
	byID := map[string]json.RawMessage{"tid": resultJSON}
	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "x.go", ToolHint: "x.go", Text: "Reading x.go", ToolUseID: "tid"},
	}
	out := formatMessageSegments(segs, 80, false, nil, true, "", byID, true)
	if strings.Contains(out, "Reading") {
		t.Fatalf("should not show in-flight activity when tool_result payload is present: %s", out)
	}
	if !strings.Contains(out, "Read 12 lines") {
		t.Fatalf("want Read 12 lines: %s", out)
	}
}

func TestFormatMessageSegments_transcriptResolvedReadShowsResultSummary(t *testing.T) {
	resultJSON := json.RawMessage(`{"type":"text","file":{"filePath":"/x.go","content":"x","numLines":30,"startLine":1,"totalLines":100}}`)
	byID := map[string]json.RawMessage{"t1": resultJSON}
	resolved := map[string]struct{}{"t1": {}}
	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "x.go · lines 1-30", ToolHint: "x.go", Text: "Reading x.go", ToolUseID: "t1"},
	}
	out := formatMessageSegments(segs, 80, false, resolved, true, "", byID, true)
	if !strings.Contains(out, "⎿") || !strings.Contains(out, "Read 30 lines") {
		t.Fatalf("want ⎿ Read 30 lines, got:\n%s", out)
	}
}

func TestFormatMessageSegments_transcriptResolvedGrepShowsMatchLine(t *testing.T) {
	resultJSON := json.RawMessage(`{"mode":"content","numFiles":0,"filenames":[],"content":"conversation-runtime/query/openai_provider.go:14:func UseOpenAIChatProvider() bool {","numLines":1}`)
	byID := map[string]json.RawMessage{"g1": resultJSON}
	resolved := map[string]struct{}{"g1": {}}
	segs := []messagerow.Segment{
		{Kind: messagerow.SegToolUse, ToolFacing: "Search", ToolParen: `pattern: "func UseOpenAIChatProvider"`, ToolHint: `"func`, ToolUseID: "g1", Text: "Searching"},
	}
	out := formatMessageSegments(segs, 80, false, resolved, true, "", byID, true)
	if !strings.Contains(out, "Found 1 line") {
		t.Fatalf("want Found 1 line, got:\n%s", out)
	}
	if !strings.Contains(out, "openai_provider.go:14") {
		t.Fatalf("want match preview line, got:\n%s", out)
	}
}

func TestMeasureMessageRows_skipsToolResultOnlyUserOnPrompt(t *testing.T) {
	m := &model{
		uiScreen: gouDemoScreenPrompt,
		store:    &conversation.Store{ConversationID: "c"},
	}
	raw := `[{"type":"tool_result","tool_use_id":"call_x","content":"ok"}]`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(raw)}
	if got := m.measureMessageRows(msg, 80, ""); got != 0 {
		t.Fatalf("measureMessageRows = %d, want 0", got)
	}
	if s := m.renderMessageRow(msg, 80, 99, ""); strings.TrimSpace(s) != "" {
		t.Fatalf("renderMessageRow want empty, got %q", s)
	}
}

func TestMeasureMessageRows_skipsToolResultOnlyUserInTranscriptCompact(t *testing.T) {
	m := &model{
		uiScreen: gouDemoScreenTranscript,
		store:    &conversation.Store{ConversationID: "c"},
	}
	raw := `[{"type":"tool_result","tool_use_id":"call_x","content":"ok"}]`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(raw)}
	if got := m.measureMessageRows(msg, 80, ""); got != 0 {
		t.Fatalf("measureMessageRows = %d, want 0 (compact transcript omits tool-only user rows)", got)
	}
}

func TestMeasureMessageRows_keepsToolResultOnlyUserInTranscriptShowAll(t *testing.T) {
	m := &model{
		uiScreen:          gouDemoScreenTranscript,
		transcriptShowAll: true,
		store:             &conversation.Store{ConversationID: "c"},
	}
	raw := `[{"type":"tool_result","tool_use_id":"call_x","content":"ok"}]`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(raw)}
	if got := m.measureMessageRows(msg, 80, ""); got < 1 {
		t.Fatalf("measureMessageRows = %d, want >=1 when show-all", got)
	}
}

func TestMeasureMessageRows_keepsUserToolResultWithText(t *testing.T) {
	m := &model{uiScreen: gouDemoScreenPrompt, store: &conversation.Store{ConversationID: "c"}}
	raw := `[{"type":"text","text":"hi"},{"type":"tool_result","tool_use_id":"x","content":"ok"}]`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u2", Content: []byte(raw)}
	if got := m.measureMessageRows(msg, 80, ""); got < 1 {
		t.Fatalf("measureMessageRows = %d, want >=1", got)
	}
}

func TestMeasureMessageRows_skipsUserWithEmptyTextPlusToolResult(t *testing.T) {
	m := &model{uiScreen: gouDemoScreenPrompt, store: &conversation.Store{ConversationID: "c"}}
	raw := `[{"type":"text","text":""},{"type":"tool_result","tool_use_id":"call_x","content":"ok"}]`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u3", Content: []byte(raw)}
	if got := m.measureMessageRows(msg, 80, ""); got != 0 {
		t.Fatalf("measureMessageRows = %d, want 0", got)
	}
}

func TestMeasureMessageRows_skipsUserToolResultFromMessageField(t *testing.T) {
	m := &model{uiScreen: gouDemoScreenPrompt, store: &conversation.Store{ConversationID: "c"}}
	inner := `{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_x","content":"ok"}]}`
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u4", Message: []byte(inner)}
	if got := m.measureMessageRows(msg, 80, ""); got != 0 {
		t.Fatalf("measureMessageRows = %d, want 0 (normalize message→content)", got)
	}
}
