package main

import (
	"strings"
	"testing"

	"goc/gou/conversation"
	"goc/types"
)

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
	if !strings.Contains(s, "[folded]") || !strings.Contains(s, string(types.MessageTypeUser)) {
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
