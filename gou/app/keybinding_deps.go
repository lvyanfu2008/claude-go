package app

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"goc/gou/app/keybindings"
	state "goc/gou/app/state"
	"goc/tools/toolexecution"
)

// Compile-time check: *model implements keybindings.Deps.
var _ keybindings.Deps = (*model)(nil)

// ── Context queries ─────────────────────────────────────────────────────────

func (m *model) ModalActive() bool {
	return m.Modal.Permission != nil
}

func (m *model) InTranscript() bool {
	return m.Screen.Mode == state.ScreenTranscript
}

func (m *model) TranscriptSearchOpen() bool {
	return m.Screen.SearchOpen
}

func (m *model) TranscriptDumpMode() bool {
	return m.Screen.DumpMode
}

func (m *model) SuggestVisible() bool {
	return m.Input.SuggVisible && len(m.Input.Suggestions) > 0
}

func (m *model) SlashListVisible() bool {
	return m.slashListVisible()
}

func (m *model) SlashPanelActive() bool {
	return m.Input.SlashResultPanel != nil
}

func (m *model) MsgViewportWanted() bool {
	return m.msgViewportWanted()
}

// ── Scroll ───────────────────────────────────────────────────────────────────

func (m *model) scrollWithTranscriptHook() {
	if m.Screen.Mode == state.ScreenTranscript {
		m.transcriptAfterManualScroll()
	}
}

func (m *model) ScrollUp() {
	m.Scroll.Sticky = false
	m.Scroll.Top = max(0, m.Scroll.Top-1)
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollDown() {
	m.Scroll.Sticky = false
	m.Scroll.Top++
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollHalfUp() {
	m.Scroll.Sticky = false
	m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollHalfDown() {
	m.Scroll.Sticky = false
	m.Scroll.Top += listViewportH(m) / 2
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollFullUp() {
	m.Scroll.Sticky = false
	m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m))
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollFullDown() {
	m.Scroll.Sticky = false
	m.Scroll.Top += listViewportH(m)
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollLineUp() {
	m.Scroll.Sticky = false
	m.Scroll.Top = max(0, m.Scroll.Top-1)
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollLineDown() {
	m.Scroll.Sticky = false
	m.Scroll.Top++
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollTop() {
	m.Scroll.Sticky = false
	m.Scroll.Top = 0
	m.scrollWithTranscriptHook()
}

func (m *model) ScrollBottom() {
	m.Scroll.Sticky = true
	m.Scroll.Top = 1 << 30
}

// ── Viewport scroll key ──────────────────────────────────────────────────────

func (m *model) HandleViewportScrollKey(msg tea.KeyPressMsg) tea.Cmd {
	return m.handleMsgViewportScrollKey(msg)
}

// ── Viewport ────────────────────────────────────────────────────────────────

func (m *model) ToggleFoldAll() {
	m.Viewport.FoldAll = !m.Viewport.FoldAll
	m.Viewport.FoldRev++
}

// ── Cmds ────────────────────────────────────────────────────────────────────

func (m *model) RedrawCmd() tea.Cmd {
	return teaGlobalRedrawCmd()
}

// ── Screen transitions ──────────────────────────────────────────────────────

func (m *model) HandleToggleTranscript() tea.Cmd {
	m.slashListUser = false
	return m.enterTranscriptScreen()
}

func (m *model) HandleExitTranscript() tea.Cmd {
	return m.exitTranscriptScreenWithPostCmd()
}

// ── Query ───────────────────────────────────────────────────────────────────

func (m *model) HandleQuit() tea.Cmd {
	if m.Input.SuggestionEngine != nil {
		m.Input.SuggestionEngine.FileIndex().Stop()
	}
	return tea.Quit
}

func (m *model) HandleInterrupt() tea.Cmd {
	if m.Query.Busy && m.Query.Cancel != nil {
		m.Query.Cancel()
		m.Query.Cancel = nil
		return nil
	}
	now := time.Now()
	if now.Sub(m.Query.LastCtrlC) < 800*time.Millisecond && m.Query.CtrlCPending {
		m.Query.CtrlCPending = false
		if m.Input.SuggestionEngine != nil {
			m.Input.SuggestionEngine.FileIndex().Stop()
		}
		return tea.Quit
	}
	m.Query.LastCtrlC = now
	m.Query.CtrlCPending = true
	return nil
}

// ── Input modes ─────────────────────────────────────────────────────────────

func (m *model) HandleToggleSlash() {
	m.toggleSlashListUser()
}

func (m *model) HandleEnterManualRender() {
	gouDemoTracef("f5 pressed: entering manual render mode (buffering events)")
	m.ManualRender.Active = true
}

func (m *model) HandleFlushManualRender() tea.Cmd {
	gouDemoTracef("f6 pressed: flushing %d buffered events", len(m.ManualRender.Events))
	m.ManualRender.Active = false
	var cmds []tea.Cmd
	for _, e := range m.ManualRender.Events {
		_, cmd := m.Update(e)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	m.ManualRender.Events = nil
	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// ── Transcript actions ──────────────────────────────────────────────────────

func (m *model) HandleOpenEditor() tea.Cmd {
	if m.Screen.EditorBusy {
		return nil
	}
	gen := m.Screen.EditorGen
	m.Screen.EditorBusy = true
	m.Screen.EditorStatus = fmt.Sprintf("rendering %d messages…", m.transcriptEffectiveN())
	return m.transcriptEditorPrepCmd(gen)
}

func (m *model) HandleDump() tea.Cmd {
	if m.Screen.DumpMode || m.Screen.SearchOpen {
		return nil
	}
	m.Screen.DumpMode = true
	m.Screen.ShowAll = true
	m.rebuildHeightCache()
	plain := transcriptExportPlain(m, exportTranscriptWidth(m))
	m.Screen.SuspendAltScreenForScrollbackDump = gouDemoAltScreenEnabled()
	return transcriptBracketDumpScrollbackCmd(plain)
}

func (m *model) HandleToggleShowAll() {
	if m.Screen.DumpMode {
		return
	}
	m.Screen.ShowAll = !m.Screen.ShowAll
	m.rebuildHeightCache()
}

func (m *model) HandleTranscriptSearchBarKey(msg tea.KeyPressMsg) tea.Cmd {
	// Call the model's search bar key handler (returns bool, but we just
	// need to forward the keypress; state changes are in-place on the model).
	m.handleTranscriptSearchBarKey(msg)
	return nil
}

func (m *model) HandleSearchNext() {
	m.transcriptSearchStep(1)
}

func (m *model) HandleSearchPrev() {
	m.transcriptSearchStep(-1)
}

// ── Suggestions ─────────────────────────────────────────────────────────────

func (m *model) HandleSuggestAccept() {
	m.applySuggestion(m.Input.Suggestions[m.Input.SelectedSuggIdx])
}

func (m *model) HandleSuggestDismiss() {
	m.Input.SuggVisible = false
	if m.Input.SuggestionEngine != nil {
		m.Input.SuggestionEngine.Dismiss()
	}
}

func (m *model) HandleSuggestPrev() {
	if m.Input.SelectedSuggIdx > 0 {
		m.Input.SelectedSuggIdx--
	} else {
		m.Input.SelectedSuggIdx = len(m.Input.Suggestions) - 1
	}
}

func (m *model) HandleSuggestNext() {
	if m.Input.SelectedSuggIdx+1 < len(m.Input.Suggestions) {
		m.Input.SelectedSuggIdx++
	} else {
		m.Input.SelectedSuggIdx = 0
	}
}

// ── Slash list ──────────────────────────────────────────────────────────────

func (m *model) HandleSlashSubmit() (tea.Model, tea.Cmd) {
	if len(m.visibleSlashList()) > 0 {
		m.applySlashTab()
		fullPrompt := strings.TrimRight(m.Input.PR.Value(), "\r\n")
		m.Input.PR.SetValue("")
		m.slashListUser = false
		m.syncSlashListAfterPrompt()
		line := strings.TrimSpace(fullPrompt)
		if line == "" {
			return m, nil
		}
		return m.gouSubmitFromPromptText(fullPrompt, line)
	}
	return m, nil
}

func (m *model) HandleSlashSelectPrev() {
	if m.slashListSel > 0 {
		m.slashListSel--
	}
}

func (m *model) HandleSlashSelectNext() {
	vis := m.visibleSlashList()
	if m.slashListSel+1 < len(vis) {
		m.slashListSel++
	}
}

func (m *model) HandleSlashSelect() {
	m.applySlashTab()
}

// ── Modal ───────────────────────────────────────────────────────────────────

func (m *model) HandleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		m.finishPermissionAsk(permissionAskReply{
			dec: toolexecution.DenyDecision("interrupted"),
			err: nil,
		})
		if m.Query.Cancel != nil {
			m.Query.Cancel()
			m.Query.Cancel = nil
		}
		return m, nil
	}
	m.handlePermissionKey(msg)
	return m, nil
}

// ── Windows redraw helper ───────────────────────────────────────────────────

func (m *model) windowsRedrawCmd() tea.Cmd {
	if runtime.GOOS == "windows" {
		return teaGlobalRedrawCmd()
	}
	return nil
}
