package messagerow

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestActivityLineForToolUse_Read(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"file_path": "/tmp/foo.go"})
	got := ActivityLineForToolUse("Read", raw)
	if !strings.HasPrefix(got, "Reading ") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "foo.go") {
		t.Fatalf("got %q", got)
	}
}

func TestActivityLineForToolUse_unknown(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"x": 1})
	if got := ActivityLineForToolUse("CustomTool", raw); got != "" {
		t.Fatalf("expected empty fallback to caller, got %q", got)
	}
}

func TestActivitySegmentForToolBlock_verboseJSON(t *testing.T) {
	t.Setenv("GOU_DEMO_VERBOSE_TOOL_OUTPUT", "1")
	raw, _ := json.Marshal(map[string]any{"command": "ls"})
	b := types.MessageContentBlock{Type: "tool_use", Name: "Bash", ID: "id1", Input: raw}
	segs := ActivitySegmentForToolBlock(b, SegToolUse, nil)
	if len(segs) != 1 || segs[0].Kind != SegToolUse {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "tool_use") || !strings.Contains(segs[0].Text, "Bash") {
		t.Fatal(segs[0].Text)
	}
}

func TestActivitySegmentForToolBlock_grepSummaryLine(t *testing.T) {
	t.Setenv("GOU_DEMO_VERBOSE_TOOL_OUTPUT", "")
	t.Setenv("GOU_DEMO_TOOL_USE_SUMMARY_LINE", "1")
	raw, _ := json.Marshal(map[string]any{"pattern": "x", "path": "/p"})
	b := types.MessageContentBlock{Type: "tool_use", Name: "Grep", ID: "tu1", Input: raw}
	segs := ActivitySegmentForToolBlock(b, SegToolUse, nil)
	if len(segs) != 1 || segs[0].Kind != SegToolUseSummaryLine {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "Searching for") || !strings.Contains(segs[0].Text, "pattern") {
		t.Fatalf("got %q", segs[0].Text)
	}
}

func TestActivitySegmentForToolBlock_grepSummaryLineResolved(t *testing.T) {
	t.Setenv("GOU_DEMO_VERBOSE_TOOL_OUTPUT", "")
	t.Setenv("GOU_DEMO_TOOL_USE_SUMMARY_LINE", "1")
	raw, _ := json.Marshal(map[string]any{"pattern": "x"})
	b := types.MessageContentBlock{Type: "tool_use", Name: "Grep", ID: "tu1", Input: raw}
	segs := ActivitySegmentForToolBlock(b, SegToolUse, &RenderOpts{
		ResolvedToolUseIDs: map[string]struct{}{"tu1": {}},
	})
	if len(segs) != 1 || segs[0].Kind != SegToolUseSummaryLine {
		t.Fatalf("%+v", segs)
	}
	if !strings.Contains(segs[0].Text, "Searched for") {
		t.Fatalf("want past tense, got %q", segs[0].Text)
	}
}
