package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goc/conversation-runtime/query"
	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/types"
)

func TestCcbStreamEventNeedsFullHeightRebuild(t *testing.T) {
	if !ccbStreamEventNeedsFullHeightRebuild(ccbstream.StreamEvent{Type: "tool_use", ID: "x"}) {
		t.Fatal("tool_use needs rebuild")
	}
	if !ccbStreamEventNeedsFullHeightRebuild(ccbstream.StreamEvent{Type: "turn_complete"}) {
		t.Fatal("turn_complete needs rebuild")
	}
	if ccbStreamEventNeedsFullHeightRebuild(ccbstream.StreamEvent{Type: "assistant_delta", Text: "a"}) {
		t.Fatal("assistant_delta should skip full rebuild")
	}
}

func TestUpdate_ccbstreamAssistantDeltaSkipsHeightRebuild(t *testing.T) {
	m := &model{
		store:    &conversation.Store{ConversationID: "c"},
		uiScreen: gouDemoScreenPrompt,
		width:    80,
		height:   24,
		cols:     76,
		titleH:   1,
		streamH:  4,
	}
	m.store.AppendMessage(types.Message{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`)})
	m.rebuildHeightCache()
	n0 := m.rebuildHeightCacheCalls
	_, _ = m.Update(tea.Msg(ccbstream.Msg{Type: "assistant_delta", Text: "hello"}))
	if m.rebuildHeightCacheCalls != n0 {
		t.Fatalf("assistant_delta should not call rebuildHeightCache: before=%d after=%d", n0, m.rebuildHeightCacheCalls)
	}
	if m.store.StreamingText != "hello" {
		t.Fatalf("StreamingText=%q", m.store.StreamingText)
	}
}

func TestUpdate_ccbstreamToolUseRebuildsHeightCache(t *testing.T) {
	m := &model{
		store:    &conversation.Store{ConversationID: "c"},
		uiScreen: gouDemoScreenPrompt,
		width:    80,
		height:   24,
		cols:     76,
		titleH:   1,
		streamH:  4,
	}
	m.rebuildHeightCache()
	n0 := m.rebuildHeightCacheCalls
	_, _ = m.Update(tea.Msg(ccbstream.Msg{Type: "tool_use", ID: "x1", Name: "Read", Input: map[string]any{"file_path": "/a"}}))
	if m.rebuildHeightCacheCalls != n0+1 {
		t.Fatalf("tool_use should rebuild height cache: before=%d after=%d", n0, m.rebuildHeightCacheCalls)
	}
}

func TestUpdate_streamingToolUsesPromptSkipsHeightRebuild(t *testing.T) {
	m := &model{
		store:    &conversation.Store{ConversationID: "c"},
		uiScreen: gouDemoScreenPrompt,
		width:    80,
		height:   24,
		cols:     76,
		titleH:   1,
		streamH:  4,
	}
	m.rebuildHeightCacheCalls = 0
	_, _ = m.Update(gouStreamingToolUsesMsg{Uses: []query.StreamingToolUseLive{{Index: 0, ToolUseID: "t1", Name: "Bash"}}})
	if m.rebuildHeightCacheCalls != 0 {
		t.Fatalf("prompt: expected no rebuildHeightCache, got %d", m.rebuildHeightCacheCalls)
	}
}

func TestUpdate_streamingToolUsesTranscriptCallsHeightRebuild(t *testing.T) {
	m := &model{
		store: &conversation.Store{
			ConversationID: "c",
			Messages:       []types.Message{{UUID: "m1", Type: types.MessageTypeUser, Content: []byte(`[{"type":"text","text":"x"}]`)}}},
		uiScreen: gouDemoScreenTranscript,
		width:    80,
		height:   24,
		cols:     76,
		titleH:   1,
		streamH:  4,
		transcriptFrozen: &frozenTranscriptSnapshot{
			MessagesLen:          1,
			StreamingToolUsesLen: 5,
		},
	}
	m.rebuildHeightCacheCalls = 0
	_, _ = m.Update(gouStreamingToolUsesMsg{Uses: []query.StreamingToolUseLive{{Index: 0, ToolUseID: "t1", Name: "Bash"}}})
	if m.rebuildHeightCacheCalls != 1 {
		t.Fatalf("transcript: want 1 rebuildHeightCache, got %d", m.rebuildHeightCacheCalls)
	}
}
