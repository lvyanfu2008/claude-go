package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/conversation"
	"goc/types"
)

func TestTranscriptSpaceFullPageDown(t *testing.T) {
	st := &conversation.Store{
		ConversationID: "c1",
		Messages:       []types.Message{{UUID: "a"}},
	}
	m := &model{
		store:               st,
		uiScreen:            gouDemoScreenTranscript,
		transcriptFreezeN: 1,
		height:              40,
		width:               100,
		cols:                80,
		titleH:              1,
		scrollTop:           0,
	}
	before := m.scrollTop
	vp := listViewportH(m)
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeySpace})
	if !handled {
		t.Fatal("expected space handled")
	}
	if m.scrollTop != before+vp {
		t.Fatalf("full page down: scrollTop got %d want %d", m.scrollTop, before+vp)
	}
}

func TestTranscriptCtrlNLineDown(t *testing.T) {
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript, transcriptFreezeN: 1,
		height: 40, width: 100, cols: 80, titleH: 1, scrollTop: 5,
	}
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	if !handled {
		t.Fatal("expected ctrl+n handled")
	}
	if m.scrollTop != 6 {
		t.Fatalf("scrollTop got %d want 6", m.scrollTop)
	}
}

func TestTranscriptSearchOpenDisablesPagerArrows(t *testing.T) {
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript, transcriptFreezeN: 1,
		height: 40, width: 100, cols: 80, titleH: 1, scrollTop: 100,
		transcriptSearchOpen: true,
	}
	before := m.scrollTop
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyUp})
	if !handled {
		t.Fatal("expected key swallowed in transcript")
	}
	if m.scrollTop != before {
		t.Fatalf("up must not scroll when search bar open: got %d want %d", m.scrollTop, before)
	}
}

func TestPlainMessageSearchText_collapsedPaths(t *testing.T) {
	msg := types.Message{
		Type:          types.MessageTypeCollapsedReadSearch,
		ReadFilePaths: []string{"src/foo.go"},
		SearchArgs:    []string{"TODO"},
	}
	s := plainMessageSearchText(msg)
	if !strings.Contains(s, "src/foo.go") || !strings.Contains(s, "todo") {
		t.Fatalf("got %q", s)
	}
}
