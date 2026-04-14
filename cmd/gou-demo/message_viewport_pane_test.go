package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

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

func TestHandleMsgViewportScrollKey_spaceHalfPageNotFullPage(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	m := newModel(st, "", "", nil)
	m.msgViewport = viewport.New(40, 10)
	m.msgViewport.KeyMap = gouDemoMsgViewportKeyMap()
	m.msgViewport.MouseWheelEnabled = false
	m.msgViewport.SetContent(strings.Repeat("line\n", 120))
	m.msgViewport.GotoTop()
	yo := m.msgViewport.YOffset
	m.handleMsgViewportScrollKey(tea.KeyMsg{Type: tea.KeySpace})
	delta := m.msgViewport.YOffset - yo
	if delta < 3 || delta > 7 {
		t.Fatalf("space should ~½ page (height 10 → ~5 lines), delta=%d y %d→%d", delta, yo, m.msgViewport.YOffset)
	}
}
