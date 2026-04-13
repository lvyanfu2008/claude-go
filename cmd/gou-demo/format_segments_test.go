package main

import (
	"strings"
	"testing"

	"goc/gou/messagerow"
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
	out := formatMessageSegments(segs, 80, true, nil, true)
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
	out := formatMessageSegments(segs, 80, true, nil, true)
	if n := countToolLeadGlyphs(out); n < 1 {
		t.Fatalf("tool-only message should keep lead on tool row, got %d in:\n%s", n, out)
	}
}

func TestFormatMessageSegments_resolvedToolOmitsActivityAndHint(t *testing.T) {
	resolved := map[string]struct{}{"call_z": {}}
	segs := []messagerow.Segment{
		{Kind: messagerow.SegTextMarkdown, Text: "hello"},
		{Kind: messagerow.SegToolUse, ToolFacing: "Read", ToolParen: "a.md", ToolHint: "a.md", Text: "Reading a", ToolUseID: "call_z"},
	}
	out := formatMessageSegments(segs, 80, true, resolved, true)
	if strings.Contains(out, "⎿") {
		t.Fatalf("resolved tool should not render ⎿ row, got:\n%s", out)
	}
	if strings.Contains(out, "Reading") {
		t.Fatalf("resolved tool should not render activity line, got:\n%s", out)
	}
}
