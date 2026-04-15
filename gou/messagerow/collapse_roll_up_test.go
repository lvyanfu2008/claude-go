package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func toolUseContent(id, name string, input map[string]any) json.RawMessage {
	raw, _ := json.Marshal([]map[string]any{{
		"type":  "tool_use",
		"id":    id,
		"name":  name,
		"input": input,
	}})
	return raw
}

func toolResultContent(toolUseID string, content string) json.RawMessage {
	raw, _ := json.Marshal([]map[string]any{{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
		"content":     content,
	}})
	return raw
}

func TestCollapseReadSearchTail_oneRead(t *testing.T) {
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/proj/README.md"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "ok")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[0].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("type=%s", msgs[0].Type)
	}
	if msgs[0].ReadCount != 1 || msgs[0].SearchCount != 0 {
		t.Fatalf("counts read=%d search=%d", msgs[0].ReadCount, msgs[0].SearchCount)
	}
	if len(msgs[0].Messages) != 2 {
		t.Fatalf("nested len=%d", len(msgs[0].Messages))
	}
	if msgs[0].DisplayMessage == nil || msgs[0].DisplayMessage.UUID != "a1" {
		t.Fatalf("displayMessage=%v", msgs[0].DisplayMessage)
	}
	if msgs[0].UUID != "collapsed-a1" {
		t.Fatalf("uuid=%q", msgs[0].UUID)
	}
}

func TestCollapseReadSearchTail_readThenGrep(t *testing.T) {
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/x/a.go"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "a")},
		{Type: types.MessageTypeAssistant, UUID: "a2", Content: toolUseContent("t2", "Grep", map[string]any{"pattern": "foo", "path": "/x"})},
		{Type: types.MessageTypeUser, UUID: "u2", Content: toolResultContent("t2", "hits")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 1 {
		t.Fatalf("len=%d", len(msgs))
	}
	m := msgs[0]
	if m.SearchCount != 1 || m.ReadCount != 1 {
		t.Fatalf("search=%d read=%d", m.SearchCount, m.ReadCount)
	}
	if len(m.SearchArgs) != 1 || m.SearchArgs[0] != "foo" {
		t.Fatalf("searchArgs=%v", m.SearchArgs)
	}
	if m.LatestDisplayHint == nil || *m.LatestDisplayHint != `"foo"` {
		t.Fatalf("hint=%v", m.LatestDisplayHint)
	}
}

func TestCollapseReadSearchTail_prefixUnchanged(t *testing.T) {
	textRaw, _ := json.Marshal([]map[string]string{{"type": "text", "text": "hi"}})
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "t0", Content: textRaw},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Glob", map[string]any{"pattern": "*.go"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "{}")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[0].UUID != "t0" || msgs[1].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("got %+v / %+v", msgs[0], msgs[1])
	}
}

func TestCollapseReadSearchTail_bashBreaksSuffix(t *testing.T) {
	// Non-collapsible Bash (git is not in BASH_* sets) breaks the rollup; only Read tail collapses.
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "b1", Content: toolUseContent("tb", "Bash", map[string]any{"command": "git status"})},
		{Type: types.MessageTypeUser, UUID: "ub", Content: toolResultContent("tb", "out")},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/f"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "x")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 3 {
		t.Fatalf("len=%d want 3 (bash pair + collapsed)", len(msgs))
	}
	if msgs[2].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("last type=%s", msgs[2].Type)
	}
}

func TestCollapseReadSearchTail_bashListCollapsesWithRead(t *testing.T) {
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "b1", Content: toolUseContent("tb", "Bash", map[string]any{"command": "ls"})},
		{Type: types.MessageTypeUser, UUID: "ub", Content: toolResultContent("tb", "out")},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/f"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "x")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 1 {
		t.Fatalf("len=%d want 1", len(msgs))
	}
	m := msgs[0]
	if m.Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("type=%s", m.Type)
	}
	if m.ListCount != 1 || m.ReadCount != 1 || m.SearchCount != 0 {
		t.Fatalf("list=%d read=%d search=%d", m.ListCount, m.ReadCount, m.SearchCount)
	}
}

func TestCollapseReadSearchTail_collapseAllBash_gitAndRead(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_ALL_BASH", "1")
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "b1", Content: toolUseContent("tb", "Bash", map[string]any{"command": "git status"})},
		{Type: types.MessageTypeUser, UUID: "ub", Content: toolResultContent("tb", "out")},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/f"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "x")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 1 {
		t.Fatalf("len=%d want 1", len(msgs))
	}
	m := msgs[0]
	if m.BashCount == nil || *m.BashCount != 1 {
		t.Fatalf("bashCount=%v", m.BashCount)
	}
	if m.ReadCount != 1 {
		t.Fatalf("read=%d", m.ReadCount)
	}
	summary := SearchReadSummaryTextFromMessage(false, m)
	if summary != "Read 1 file, ran 1 bash command" {
		t.Fatalf("summary=%q", summary)
	}
}

func TestCollapseReadSearchTail_mismatchedIDNoCollapse(t *testing.T) {
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/f"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("wrong", "x")},
	}
	CollapseReadSearchTail(&msgs)
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
}
