package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"

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

func TestScrollItemKeys_transcript_appendsStreamingToolKeys(t *testing.T) {
	st := &conversation.Store{
		ConversationID: "c1",
		Messages:       []types.Message{{UUID: "m1"}},
		StreamingToolUses: []conversation.StreamingToolUse{
			{Index: 0, ToolUseID: "t1", Name: "Bash", UnparsedInput: `{"x":1}`},
			{Index: 1, ToolUseID: "t2", Name: "Read", UnparsedInput: ""},
		},
	}
	m := &model{
		store:    st,
		uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{
			MessagesLen:          1,
			StreamingToolUsesLen: 2,
		},
	}
	keys := m.scrollItemKeys()
	if len(keys) != 3 {
		t.Fatalf("want 1 msg + 2 stream keys, got %d: %v", len(keys), keys)
	}
	if keys[1] != transcriptStreamToolScrollKey("c1", 0) {
		t.Fatalf("stream key0: %q", keys[1])
	}
	if keys[2] != transcriptStreamToolScrollKey("c1", 1) {
		t.Fatalf("stream key1: %q", keys[2])
	}
}

func TestTranscriptStreamingToolsForView_capsByFrozenLen(t *testing.T) {
	st := &conversation.Store{
		ConversationID: "c1",
		StreamingToolUses: []conversation.StreamingToolUse{
			{ToolUseID: "a"}, {ToolUseID: "b"}, {ToolUseID: "c"},
		},
	}
	m := &model{
		store:    st,
		uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{
			MessagesLen:          0,
			StreamingToolUsesLen: 2,
		},
	}
	v := m.transcriptStreamingToolsForView()
	if len(v) != 2 || v[0].Single.ToolUseID != "a" || v[1].Single.ToolUseID != "b" {
		t.Fatalf("got %+v", v)
	}
}

func TestHandleTranscriptKeySwallowsUnknown(t *testing.T) {
	m := &model{store: &conversation.Store{ConversationID: "x"}, uiScreen: gouDemoScreenTranscript}
	handled, cmd := m.handleTranscriptKey(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if !handled || cmd != nil {
		t.Fatalf("expected swallow without cmd: handled=%v cmd=%v", handled, cmd)
	}
}

func TestExitTranscriptScreenWithPostCmd_afterDump(t *testing.T) {
	t.Parallel()
	m := &model{
		store:              &conversation.Store{ConversationID: "x"},
		uiScreen:           gouDemoScreenTranscript,
		transcriptDumpMode: true,
	}
	cmd := m.exitTranscriptScreenWithPostCmd()
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
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
}

func TestExitTranscriptScreenWithPostCmd_noDumpNilCmd(t *testing.T) {
	t.Parallel()
	m := &model{
		store:              &conversation.Store{ConversationID: "x"},
		uiScreen:           gouDemoScreenTranscript,
		transcriptDumpMode: false,
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
