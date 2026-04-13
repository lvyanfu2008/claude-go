package main

import (
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

func TestFormatMessageSegments_singleLeadGlyphAfterAssistantText(t *testing.T) {
	segs := []messagerow.Segment{
		{Kind: messagerow.SegTextMarkdown, Text: "我来查看一下项目结构和主要业务。"},
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "README.md", ToolHint: "README.md", Text: "Reading README", ToolUseID: "call_x"},
	}
	out := formatMessageSegments(segs, 80, true, nil, true, "")
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
	out := formatMessageSegments(segs, 80, true, nil, true, "")
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
	out := formatMessageSegments(segs, 80, true, nil, true, "beta")
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
	out := formatMessageSegments(segs, 80, true, nil, true, "")
	outHL := formatMessageSegments(segs, 80, true, nil, true, "   ")
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
	out := formatMessageSegments(segs, 80, true, resolved, true, "")
	if strings.Contains(out, "⎿") {
		t.Fatalf("resolved tool should not render ⎿ row, got:\n%s", out)
	}
	if strings.Contains(out, "Reading") {
		t.Fatalf("resolved tool should not render activity line, got:\n%s", out)
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
