package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/conversation"
	"goc/types"
)

func TestClampTranscriptFreeze(t *testing.T) {
	if g := clampTranscriptFreeze(5, 3); g != 3 {
		t.Fatalf("cap to nMsgs: got %d want 3", g)
	}
	if g := clampTranscriptFreeze(-1, 10); g != 0 {
		t.Fatalf("negative freeze: got %d want 0", g)
	}
	if g := clampTranscriptFreeze(4, 4); g != 4 {
		t.Fatalf("equal: got %d want 4", g)
	}
}

func TestScrollItemKeysTranscriptSubset(t *testing.T) {
	st := &conversation.Store{
		ConversationID: "c1",
		Messages: []types.Message{
			{UUID: "a"}, {UUID: "b"}, {UUID: "c"},
		},
	}
	m := &model{store: st, uiScreen: gouDemoScreenTranscript, transcriptFreezeN: 2}
	keys := m.scrollItemKeys()
	if len(keys) != 2 {
		t.Fatalf("len keys: got %d want 2", len(keys))
	}
	want0 := conversation.ItemKey(st.Messages[0], "c1")
	if keys[0] != want0 {
		t.Fatalf("key0: got %q want %q", keys[0], want0)
	}
}

func TestHandleTranscriptKeySwallowsUnknown(t *testing.T) {
	m := &model{store: &conversation.Store{ConversationID: "x"}, uiScreen: gouDemoScreenTranscript}
	if !m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}) {
		t.Fatal("expected swallow")
	}
}
