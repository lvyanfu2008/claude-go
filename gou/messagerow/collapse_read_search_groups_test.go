package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestCollapseReadSearchGroupsInList_envOffNoop(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL", "")
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/a"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "ok")},
	}
	got := CollapseReadSearchGroupsInList(msgs)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
}

func TestCollapseReadSearchGroupsInList_twoPairs(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL", "1")
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/a"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "ok")},
		{Type: types.MessageTypeAssistant, UUID: "a2", Content: toolUseContent("t2", "Read", map[string]any{"file_path": "/b"})},
		{Type: types.MessageTypeUser, UUID: "u2", Content: toolResultContent("t2", "ok")},
	}
	got := CollapseReadSearchGroupsInList(msgs)
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if got[0].Type != types.MessageTypeCollapsedReadSearch || got[0].ReadCount != 2 {
		t.Fatalf("got type=%s read=%d", got[0].Type, got[0].ReadCount)
	}
}

func TestCollapseReadSearchGroupsInList_thinkingAndToolUseSameAssistantMessage(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL", "1")
	raw, _ := json.Marshal([]map[string]any{
		{"type": "thinking", "thinking": "..."},
		{"type": "tool_use", "id": "t1", "name": "Read", "input": map[string]any{"file_path": "/a"}},
	})
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: raw},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "ok")},
	}
	got := CollapseReadSearchGroupsInList(msgs)
	if len(got) != 1 || got[0].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("got len=%d first=%v", len(got), got)
	}
	if got[0].ReadCount != 1 {
		t.Fatalf("readCount=%d", got[0].ReadCount)
	}
}

func TestCollapseReadSearchGroupsInList_thinkingDeferredAfterCollapsed(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL", "1")
	thinkingRaw, _ := json.Marshal([]map[string]any{{"type": "thinking", "thinking": "..."}})
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Read", map[string]any{"file_path": "/a"})},
		{Type: types.MessageTypeAssistant, UUID: "think", Content: thinkingRaw},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "ok")},
	}
	got := CollapseReadSearchGroupsInList(msgs)
	if len(got) != 2 {
		t.Fatalf("len=%d want collapsed + deferred thinking", len(got))
	}
	if got[0].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("first type=%s", got[0].Type)
	}
	if got[1].UUID != "think" {
		t.Fatalf("second should be deferred thinking, got %+v", got[1])
	}
}

func TestCollapseReadSearchGroupsInList_prefixUnchanged(t *testing.T) {
	t.Setenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL", "1")
	textRaw, _ := json.Marshal([]map[string]string{{"type": "text", "text": "hi"}})
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "t0", Content: textRaw},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: toolUseContent("t1", "Grep", map[string]any{"pattern": "x"})},
		{Type: types.MessageTypeUser, UUID: "u1", Content: toolResultContent("t1", "{}")},
	}
	got := CollapseReadSearchGroupsInList(msgs)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].UUID != "t0" || got[1].Type != types.MessageTypeCollapsedReadSearch {
		t.Fatalf("got %+v", got)
	}
}
