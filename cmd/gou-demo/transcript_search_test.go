package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/conversation"
	"goc/types"
)

func TestTranscriptCtrlLGlobalRedraw(t *testing.T) {
	m := &model{
		uiScreen:         gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
	}
	handled, cmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	if !handled || cmd == nil {
		t.Fatalf("handled=%v cmd=%v", handled, cmd)
	}
	got := cmd()
	want := tea.ClearScreen()
	if reflect.TypeOf(got) != reflect.TypeOf(want) {
		t.Fatalf("cmd() type %T want same kind as tea.ClearScreen() %T", got, want)
	}
}

func TestTranscriptSpaceFullPageDown(t *testing.T) {
	st := &conversation.Store{
		ConversationID: "c1",
		Messages:       []types.Message{{UUID: "a"}},
	}
	m := &model{
		store:            st,
		uiScreen:         gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40,
		width:            100,
		cols:             80,
		titleH:           1,
		scrollTop:        0,
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
		store: st, uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40, width: 100, cols: 80, titleH: 1, scrollTop: 5,
	}
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	if !handled {
		t.Fatal("expected ctrl+n handled")
	}
	if m.scrollTop != 6 {
		t.Fatalf("scrollTop got %d want 6", m.scrollTop)
	}
}

func TestTranscriptHomeEndTopBottom(t *testing.T) {
	t.Parallel()
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40, width: 100, cols: 80, titleH: 1, scrollTop: 42,
	}
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyHome})
	if !handled {
		t.Fatal("expected home handled")
	}
	if m.scrollTop != 0 || m.sticky {
		t.Fatalf("home: scrollTop=%d sticky=%v", m.scrollTop, m.sticky)
	}
	m.scrollTop = 0
	handled2, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyEnd})
	if !handled2 {
		t.Fatal("expected end handled")
	}
	if !m.sticky || m.scrollTop != 1<<30 {
		t.Fatalf("end: scrollTop=%d sticky=%v", m.scrollTop, m.sticky)
	}
}

func TestTranscriptCtrlHomeCtrlEndTopBottom(t *testing.T) {
	t.Parallel()
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40, width: 100, cols: 80, titleH: 1, scrollTop: 99,
	}
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlHome})
	if !handled {
		t.Fatal("expected ctrl+home handled")
	}
	if m.scrollTop != 0 || m.sticky {
		t.Fatalf("ctrl+home: scrollTop=%d sticky=%v", m.scrollTop, m.sticky)
	}
	handled2, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlEnd})
	if !handled2 {
		t.Fatal("expected ctrl+end handled")
	}
	if !m.sticky || m.scrollTop != 1<<30 {
		t.Fatalf("ctrl+end: scrollTop=%d sticky=%v", m.scrollTop, m.sticky)
	}
}

func TestTranscriptSearchOpenDisablesPagerArrows(t *testing.T) {
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40, width: 100, cols: 80, titleH: 1, scrollTop: 100,
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

func TestTranscriptSearchOpenDisablesCtrlHome(t *testing.T) {
	t.Parallel()
	st := &conversation.Store{ConversationID: "c1", Messages: []types.Message{{UUID: "a"}}}
	m := &model{
		store: st, uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{MessagesLen: 1, StreamingToolUsesLen: 0},
		height:           40, width: 100, cols: 80, titleH: 1, scrollTop: 50,
		transcriptSearchOpen: true,
	}
	before := m.scrollTop
	handled, _ := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyCtrlHome})
	if !handled {
		t.Fatal("expected ctrl+home swallowed")
	}
	if m.scrollTop != before {
		t.Fatalf("ctrl+home must not jump to top when search bar open: got %d want %d", m.scrollTop, before)
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

func testTranscriptModelWithMessages(t *testing.T, msgs []types.Message) *model {
	t.Helper()
	st := &conversation.Store{ConversationID: "c1", Messages: msgs}
	return &model{
		store:    st,
		uiScreen: gouDemoScreenTranscript,
		transcriptFrozen: &frozenTranscriptSnapshot{
			MessagesLen:          len(msgs),
			StreamingToolUsesLen: 0,
		},
		height:               40,
		width:                120,
		cols:                 100,
		titleH:               1,
		heightCache:          make(map[string]int),
		programUsesAltScreen: false,
	}
}

func TestHandleTranscriptKey_bracketEntersDumpModeAndShowAll(t *testing.T) {
	t.Parallel()
	userJSON := `[{"type":"text","text":"hello"}]`
	m := testTranscriptModelWithMessages(t, []types.Message{
		{UUID: "u1", Type: types.MessageTypeUser, Content: []byte(userJSON)},
	})
	handled, cmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if !handled {
		t.Fatal("expected [ handled")
	}
	if cmd == nil {
		t.Fatal("expected dump cmd")
	}
	if !m.transcriptDumpMode || !m.transcriptShowAll {
		t.Fatalf("dumpMode=%v showAll=%v", m.transcriptDumpMode, m.transcriptShowAll)
	}
	if cmd() == nil {
		t.Fatal("expected printable message from dump cmd")
	}
}

func TestHandleTranscriptKey_bracketNoopWhenSearchOpen(t *testing.T) {
	t.Parallel()
	m := testTranscriptModelWithMessages(t, []types.Message{{UUID: "a", Type: types.MessageTypeUser}})
	m.transcriptSearchOpen = true
	handled, cmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if !handled || cmd != nil {
		t.Fatalf("expected swallow without cmd: handled=%v cmd=%v", handled, cmd)
	}
	if m.transcriptDumpMode {
		t.Fatal("must not enter dump mode while search bar is open")
	}
}

func TestHandleTranscriptKey_vEditorPrepWritesFileWithoutEditor(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	userJSON := `[{"type":"text","text":"line"}]`
	m := testTranscriptModelWithMessages(t, []types.Message{
		{UUID: "u1", Type: types.MessageTypeUser, Content: []byte(userJSON)},
	})
	handled, cmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !handled || cmd == nil {
		t.Fatalf("expected v handled with cmd: handled=%v cmd=%v", handled, cmd)
	}
	if !m.transcriptEditorBusy {
		t.Fatal("expected editor busy during prep")
	}
	prep, ok := cmd().(gouTranscriptEditorPrepMsg)
	if !ok {
		t.Fatalf("expected gouTranscriptEditorPrepMsg, got %T", cmd())
	}
	if prep.Err != nil {
		t.Fatal(prep.Err)
	}
	if prep.Path == "" {
		t.Fatal("expected temp path")
	}
	if _, err := os.Stat(prep.Path); err != nil {
		t.Fatalf("transcript file: %v", err)
	}
	body, err := os.ReadFile(prep.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "user") || !strings.Contains(string(body), "line") {
		t.Fatalf("unexpected export body: %q", string(body))
	}
	t.Cleanup(func() { _ = os.Remove(prep.Path) })

	_ = m.handleTranscriptEditorChainMsg(prep)
	if m.transcriptEditorBusy {
		t.Fatal("expected busy cleared after prep when no editor")
	}
	if !strings.Contains(m.transcriptEditorStatus, "wrote") || !strings.Contains(m.transcriptEditorStatus, "no $VISUAL") {
		t.Fatalf("status: %q", m.transcriptEditorStatus)
	}
}

func TestHandleTranscriptKey_vIgnoredWhenEditorBusy(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	m := testTranscriptModelWithMessages(t, []types.Message{{UUID: "a", Type: types.MessageTypeUser, Content: []byte(`[{"type":"text","text":"x"}]`)}})
	handled, prepCmd := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !handled || prepCmd == nil {
		t.Fatal("first v should start prep")
	}
	if !m.transcriptEditorBusy {
		t.Fatal("expected busy before Update handles gouTranscriptEditorPrepMsg")
	}
	_ = prepCmd() // synchronous prep; busy clears only in handleTranscriptEditorChainMsg
	if !m.transcriptEditorBusy {
		t.Fatal("expected still busy until chain runs")
	}
	handled2, cmd2 := m.handleTranscriptKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !handled2 || cmd2 != nil {
		t.Fatalf("second v while busy: handled=%v cmd=%v", handled2, cmd2)
	}
}
