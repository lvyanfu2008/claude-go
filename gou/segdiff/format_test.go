package segdiff

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"goc/gou/messagerow"
)

func TestFormatToolResultSegmentForTranscript_unifiedBodyUsesANSI(t *testing.T) {
	seg := messagerow.Segment{
		Kind: messagerow.SegToolResult,
		Text: "tool_result tool_use_id=x\n--- a.go\n+++ a.go\n@@ -1,1 +1,1 @@\n-old\n+new",
	}
	withHL := func(s string) string { return s }
	base := func(userRow bool) lipgloss.Style { return lipgloss.NewStyle() }
	out := FormatToolResultSegmentForTranscript(seg, false, false, withHL, base)
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected styled output, got %q", out)
	}
	if !strings.Contains(out, "tool_result") {
		t.Fatalf("missing header: %q", out)
	}
}
