package main

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"goc/gou/conversation"
	"goc/types"
)

func TestTryBuildFullMessagePaneContent_userThenStreamingToolBlankLine(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	// API-shaped Message so [IsStreamToolUsing] matches StreamingToolUses (append tool chrome block in tryBuildFullMessagePaneContent).
	userMsg := `{"role":"user","content":[{"type":"text","text":"hi"},{"type":"tool_use","id":"t1","name":"Grep","input":{}}]}`
	st.Messages = []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`), Message: []byte(userMsg)},
	}
	st.StreamingToolUses = []conversation.StreamingToolUse{{ToolUseID: "t1", Name: "Grep", UnparsedInput: "{}"}}
	m := newModel(st, "", "", nil)
	m.width = 80
	m.height = 40
	m.cols = 76
	m.titleH = 1
	m.streamH = 4
	m.uiScreen = gouDemoScreenPrompt
	m.useMsgViewport = true
	m.rebuildHeightCache()
	s, ok := m.tryBuildFullMessagePaneContent()
	if !ok {
		t.Fatal("tryBuildFullMessagePaneContent failed")
	}
	if !strings.Contains(s, "\n\n") {
		t.Fatalf("expected blank line between last user message and first streaming tool row, got %q", s)
	}
}

func TestTryBuildFullMessagePaneContent_userThenStreamingTextBlankLine(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	st.Messages = []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`)},
	}
	st.StreamingText = "streaming reply"
	m := newModel(st, "", "", nil)
	m.width = 80
	m.height = 40
	m.cols = 76
	m.titleH = 1
	m.streamH = 4
	m.uiScreen = gouDemoScreenPrompt
	m.useMsgViewport = true
	m.rebuildHeightCache()
	s, ok := m.tryBuildFullMessagePaneContent()
	if !ok {
		t.Fatal("tryBuildFullMessagePaneContent failed")
	}
	if !strings.Contains(s, "\n\n") {
		t.Fatalf("expected blank line between last user message and StreamingText tail, got %q", s)
	}
}

func TestTryBuildFullMessagePaneContent_userAssistantBlankLine(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	st.Messages = []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`)},
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: []byte(`[{"type":"text","text":"yo"}]`)},
	}
	m := newModel(st, "", "", nil)
	m.width = 80
	m.height = 40
	m.cols = 76
	m.titleH = 1
	m.streamH = 4
	m.uiScreen = gouDemoScreenPrompt
	m.useMsgViewport = true
	m.rebuildHeightCache()
	s, ok := m.tryBuildFullMessagePaneContent()
	if !ok {
		t.Fatal("tryBuildFullMessagePaneContent failed")
	}
	// One blank line between user block and assistant block (two consecutive newlines in raw string).
	if !strings.Contains(s, "\n\n") {
		t.Fatalf("expected blank line between user and assistant in pane content, got %q", s)
	}
}

func TestTryBuildFullMessagePaneContent_fold(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	st.Messages = []types.Message{{
		Type:    types.MessageTypeUser,
		UUID:    "u1",
		Content: []byte(`[{"type":"text","text":"hello"}]`),
	}}
	m := newModel(st, "", "", nil)
	m.width = 80
	m.height = 40
	m.cols = 76
	m.titleH = 1
	m.streamH = 4
	m.uiScreen = gouDemoScreenPrompt
	m.useMsgViewport = true
	m.msgFoldAll = true
	m.rebuildHeightCache()
	s, ok := m.tryBuildFullMessagePaneContent()
	if !ok {
		t.Fatal("tryBuildFullMessagePaneContent failed")
	}
	if !strings.Contains(s, "[folded]") || !strings.Contains(s, ">") || !strings.Contains(s, "u1") {
		t.Fatalf("unexpected folded output: %q", s)
	}
}

func TestMsgViewportWanted_transcriptOff(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	m := newModel(st, "", "", nil)
	m.useMsgViewport = true
	m.uiScreen = gouDemoScreenTranscript
	if m.msgViewportWanted() {
		t.Fatal("viewport pane is prompt-only")
	}
}

func TestHandleMsgViewportScrollKey_spaceHalfPageNotFullPage(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	m := newModel(st, "", "", nil)
	m.msgViewport = viewport.New(viewport.WithWidth(40), viewport.WithHeight(10))
	m.msgViewport.KeyMap = gouDemoMsgViewportKeyMap()
	m.msgViewport.MouseWheelEnabled = false
	m.msgViewport.SetContent(strings.Repeat("line\n", 120))
	m.msgViewport.GotoTop()
	yo := m.msgViewport.YOffset()
	m.handleMsgViewportScrollKey(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace}))
	delta := m.msgViewport.YOffset() - yo
	if delta < 3 || delta > 7 {
		t.Fatalf("space should ~½ page (height 10 → ~5 lines), delta=%d y %d→%d", delta, yo, m.msgViewport.YOffset())
	}
}
