package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/conversation"
	"goc/types"
)

func TestMouseYInMessageListPane_transcriptBand(t *testing.T) {
	m := &model{
		store:    &conversation.Store{ConversationID: "c"},
		height:   40,
		width:    100,
		cols:     96,
		titleH:   1,
		streamH:  4,
		uiScreen: gouDemoScreenTranscript,
	}
	top := m.titleH
	bot := top + listViewportH(m)
	if top != 1 {
		t.Fatalf("top=%d", top)
	}
	if !m.mouseYInMessageListPane(top) {
		t.Fatalf("y=%d should be inside pane", top)
	}
	if m.mouseYInMessageListPane(bot) {
		t.Fatalf("y=%d should be outside pane (exclusive end)", bot)
	}
	if !m.mouseYInMessageListPane(bot - 1) {
		t.Fatalf("y=%d last row inside pane", bot-1)
	}
}

func TestTryHandleMessageListMouse_wheelInPane(t *testing.T) {
	t.Setenv("GOU_DEMO_DISABLE_MOUSE_SCROLL", "")
	m := &model{
		store:     &conversation.Store{ConversationID: "c"},
		height:    40,
		width:     100,
		cols:      96,
		titleH:    1,
		streamH:   4,
		uiScreen:  gouDemoScreenTranscript,
		scrollTop: 100,
	}
	y := m.titleH + 2
	if !m.tryHandleMessageListMouse(tea.MouseMsg{Y: y, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress}) {
		t.Fatal("expected wheel down handled")
	}
	if m.scrollTop <= 100 {
		t.Fatalf("scrollTop should increase, got %d", m.scrollTop)
	}
	if m.sticky {
		t.Fatal("sticky should clear on manual scroll")
	}
}

func TestClampScrollTopForVirtualList_normalizesHugeScrollTop(t *testing.T) {
	m := &model{
		store:     &conversation.Store{ConversationID: "c"},
		height:    40,
		width:     100,
		cols:      96,
		titleH:    1,
		streamH:   4,
		uiScreen:  gouDemoScreenTranscript,
		sticky:    false,
		scrollTop: 1 << 30,
	}
	m.store.Messages = []types.Message{{UUID: "u0"}}
	m.heightCache = map[string]int{conversation.ItemKey(m.store.Messages[0], "c"): 5}
	// scrollItemKeys uses messagesForScroll; single message one key.
	m.clampScrollTopForVirtualList()
	vp := listViewportH(m)
	if m.scrollTop < 0 || m.scrollTop > 1<<20 {
		t.Fatalf("scrollTop should be clamped to a modest range, got %d (vp=%d)", m.scrollTop, vp)
	}
}

func TestTryHandleMessageListMouse_respectsDisableEnv(t *testing.T) {
	t.Setenv("GOU_DEMO_DISABLE_MOUSE_SCROLL", "1")
	m := &model{
		height:    40,
		titleH:    1,
		uiScreen:  gouDemoScreenTranscript,
		scrollTop: 10,
	}
	y := m.titleH + 1
	if m.tryHandleMessageListMouse(tea.MouseMsg{Y: y, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress}) {
		t.Fatal("should not handle when disabled")
	}
	if m.scrollTop != 10 {
		t.Fatalf("scrollTop unchanged, got %d", m.scrollTop)
	}
}
