package main

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"goc/gou/conversation"
)

func TestMsgViewportWanted_fallback(t *testing.T) {
	st := &conversation.Store{ConversationID: "c"}
	m := newModel(st, "", "", nil)
	m.useMsgViewport = true
	m.msgViewportFallback = false
	if !m.msgViewportWanted() {
		t.Fatal("viewport should be wanted when fallback is off")
	}
	m.msgViewportFallback = true
	if m.msgViewportWanted() {
		t.Fatal("viewport should not be wanted when fallback is on")
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
