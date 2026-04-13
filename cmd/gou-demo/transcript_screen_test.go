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
	m := &model{
		store:            st,
		uiScreen:         gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 2, StreamingToolUsesLen: 0},
	}
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
	handled, cmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if !handled || cmd != nil {
		t.Fatalf("expected swallow without cmd: handled=%v cmd=%v", handled, cmd)
	}
}

func TestExitTranscriptScreenWithPostCmd_altScreenAfterDump(t *testing.T) {
	t.Parallel()
	m := &model{
		store:                &conversation.Store{ConversationID: "x"},
		uiScreen:             gouDemoScreenTranscript,
		programUsesAltScreen: true,
		transcriptDumpMode:   true,
	}
	cmd := m.exitTranscriptScreenWithPostCmd()
	if cmd == nil {
		t.Fatal("expected EnterAltScreen cmd when leaving dump with alt-screen program")
	}
	if m.uiScreen != gouDemoScreenPrompt {
		t.Fatalf("uiScreen got %v want prompt", m.uiScreen)
	}
	if m.transcriptDumpMode {
		t.Fatal("dump mode should clear on exit")
	}
	if m.transcriptFrozen != nil {
		t.Fatal("frozen snapshot should clear on exit")
	}
	if m.transcriptShowAll {
		t.Fatal("showAll should reset on exit (TS handleExitTranscript)")
	}
	if cmd() == nil {
		t.Fatal("expected tea.Msg from post cmd")
	}
}

func TestExitTranscriptScreenWithPostCmd_noCmdWithoutAltOrDump(t *testing.T) {
	t.Parallel()
	m := &model{
		store:                &conversation.Store{ConversationID: "x"},
		uiScreen:             gouDemoScreenTranscript,
		programUsesAltScreen: true,
		transcriptDumpMode:   false,
	}
	if m.exitTranscriptScreenWithPostCmd() != nil {
		t.Fatal("expected nil cmd when not leaving dump mode")
	}
}

func TestEnterExitTranscript_frozenSnapshotMatchesTS(t *testing.T) {
	t.Parallel()
	st := &conversation.Store{
		ConversationID: "c",
		Messages:       []types.Message{{UUID: "1"}, {UUID: "2"}},
	}
	m := &model{store: st, uiScreen: gouDemoScreenPrompt, scrollTop: 3, sticky: false}
	m.transcriptShowAll = true
	st.AppendStreamingToolUse(conversation.StreamingToolUse{Index: 0, ToolUseID: "tu-live", Name: "Read"})
	m.enterTranscriptScreen()
	if m.transcriptFrozen == nil || m.transcriptFrozen.MessagesLen != 2 || m.transcriptFrozen.StreamingToolUsesLen != 1 {
		t.Fatalf("frozen %+v", m.transcriptFrozen)
	}
	if m.transcriptShowAll {
		t.Fatal("enter transcript should reset showAll (TS toggle sets setShowAllInTranscript(false))")
	}
	m.exitTranscriptScreen()
	if m.transcriptFrozen != nil {
		t.Fatal("exit should clear frozen state (TS onExitTranscript)")
	}
	if m.transcriptShowAll {
		t.Fatal("exit should reset showAll")
	}
}
