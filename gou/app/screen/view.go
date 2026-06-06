// Package screen assembles the Bubble Tea View for gou-demo.
// It extracts the view assembly logic from main.go, keeping each section
// (chrome, prompt, transcript, input_area, viewport) in its own file.
package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/app/state"
	"goc/types"
)

// Deps defines all model state and methods needed for view assembly.
type Deps interface {
	// --- Modal ---
	ModalQuestionViewContent() string // empty string if no question
	ModalPermission() interface{}

	// --- Layout ---
	Width() int
	Cols() int
	Height() int
	MsgBodyCols() int
	MsgScrollbarW() int

	// --- Screen ---
	ScreenMode() state.ScreenMode
	ShowAll() bool
	DumpMode() bool
	TranscriptSearchOpen() bool

	// --- Chrome ---
	LastEmittedTitlePlain() string
	SetLastEmittedTitlePlain(v string)
	ConversationID() string
	IsBusy() bool
	HasStreaming() bool
	PermissionMode() types.PermissionMode

	// --- Message Pane (viewport) ---
	ListViewportH() int
	ViewportWanted() bool
	SyncViewportGeometry()
	ApplyViewportContent()
	ViewportFallback() bool
	ViewportBlock(vpH, bodyCols int) string

	// --- Message Pane (non-viewport) ---
	RenderMessagePane() string
	IntegrateRenderer()
	MessagePtrCount() int
	MessagePtrSlice() []*types.Message
	ComputeVisibleRange(msgs []*types.Message, scrollTop, vpHeight int, isTranscript, verbose bool, width int) (int, int, int)

	// --- Scroll ---
	ScrollTop() int

	// --- Bottom Area (prompt) ---
	PromptStreamRows() []string
	TaskListVisible() bool
	TaskListView(maxDisplay, cols int) string
	TaskListViewMaxDisplay() int
	TaskListViewReservedRows() int
	AgentCoordinatorView() string

	// --- Status Line ---
	StatusLine() string

	// --- Transcript Footer ---
	TranscriptFootLines(narrow bool) string
	TranscriptEditorStatus() string

	// --- Input Area ---
	AtSuggestionView() string
	BuiltinStatusView() string
	UserInputView() string
	SlashPanelBlock() string
	SlashListVisible() bool
	SlashPicker(width, height int) string

	// --- Permission Modal ---
	PermissionModalView(width int) string

	// --- Root View ---
	SuspendAltScreen() bool
	HistoryBrowseMouseOff() bool
}

// ViewAssembler orchestrates view assembly from model state via Deps.
type ViewAssembler struct {
	deps Deps
}

// New creates a new ViewAssembler.
func New(deps Deps) *ViewAssembler {
	return &ViewAssembler{deps: deps}
}

// Assemble builds the complete Bubble Tea View from model state.
func (va *ViewAssembler) Assemble() tea.View {
	d := va.deps

	// Modal intercept: interactive question UI takes over the full view.
	if qv := d.ModalQuestionViewContent(); qv != "" {
		return wrapRootView(tea.NewView(qv), d)
	}

	// Width==0 loading state (before first WindowSizeMsg).
	if d.Width() == 0 {
		return wrapRootView(tea.NewView("Loading…"), d)
	}

	vpH := d.ListViewportH()
	bodyCols := d.MsgBodyCols()
	useVp := d.ViewportWanted()
	if useVp {
		d.SyncViewportGeometry()
		d.ApplyViewportContent()
		if d.ViewportFallback() {
			useVp = false
		}
	}

	var b strings.Builder
	narrow := d.Cols() > 0 && d.Cols() < 80

	// Terminal title (OSC 0 write once on change).
	plainTitle := ComposeTerminalTitle(d.ConversationID(), d.IsBusy(), d.HasStreaming())
	if !TerminalTitleDisabled() && plainTitle != d.LastEmittedTitlePlain() {
		d.SetLastEmittedTitlePlain(plainTitle)
		if osc := SetWindowTitle(plainTitle); osc != "" {
			b.WriteString(osc)
		}
	}

	// Top bar.
	topBar := TopBar(narrow)
	if d.ScreenMode() == state.ScreenTranscript {
		topBar = TranscriptTopBar(narrow)
	}
	title := lipgloss.NewStyle().Bold(true).Render(topBar)
	b.WriteString(title)
	b.WriteByte('\n')

	// Message pane.
	if useVp {
		b.WriteString(d.ViewportBlock(vpH, bodyCols))
		b.WriteByte('\n')
	} else {
		msgPaneContent := d.RenderMessagePane()
		lines := strings.Split(msgPaneContent, "\n")
		if len(lines) > vpH {
			lines = lines[:vpH]
		}
		for len(lines) < vpH {
			lines = append(lines, "")
		}
		d.IntegrateRenderer()
		msgsPtr := d.MessagePtrSlice()
		isTranscript := d.ScreenMode() == state.ScreenTranscript
		verbose := d.ShowAll() || (d.ScreenMode() == state.ScreenTranscript && d.TranscriptSearchOpen())
		_, _, totalHeight := d.ComputeVisibleRange(msgsPtr, 0, 1, isTranscript, verbose, bodyCols)
		b.WriteString(JoinMessagePaneLinesWithScrollbar(lines, bodyCols, vpH, totalHeight, d.ScrollTop(), d.MsgScrollbarW()))
		b.WriteByte('\n')
	}

	// Bottom area (prompt only: stream rows, task list, agent panel).
	if d.ScreenMode() != state.ScreenTranscript {
		streamRows := d.PromptStreamRows()
		if len(streamRows) > 0 {
			b.WriteString(strings.Join(streamRows, "\n"))
			b.WriteByte('\n')
		}
		if d.TaskListVisible() {
			maxDisplay := d.TaskListViewMaxDisplay()
			if tl := d.TaskListView(maxDisplay, d.Cols()); tl != "" {
				b.WriteString(ApplyMessagePaneGutter(tl, d.Width()))
				b.WriteByte('\n')
			}
		}
		if cv := d.AgentCoordinatorView(); cv != "" {
			b.WriteString(ApplyMessagePaneGutter(cv, d.Width()))
			b.WriteByte('\n')
		}
	}

	// Status line.
	if s := d.StatusLine(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}

	// Transcript footer or input area.
	if d.ScreenMode() == state.ScreenTranscript {
		foot := d.TranscriptFootLines(narrow)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(d.Cols()).Render(foot))
	} else {
		if s := d.AtSuggestionView(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		if s := d.BuiltinStatusView(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		b.WriteString(AboveInputRuleLine(d.Cols()))
		b.WriteByte('\n')
		b.WriteString(d.UserInputView())
		if blk := d.SlashPanelBlock(); blk != "" {
			b.WriteByte('\n')
			b.WriteString(blk)
		}
		if d.SlashListVisible() {
			if sp := d.SlashPicker(d.Cols(), d.Height()); sp != "" {
				b.WriteByte('\n')
				b.WriteString(sp)
			}
		}
	}

	out := lipgloss.NewStyle().MaxWidth(d.Width()).Render(b.String())
	if d.ModalPermission() != nil {
		mod := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(d.PermissionModalView(d.Width()))
		out = lipgloss.JoinVertical(lipgloss.Left, out, mod)
	}
	return wrapRootView(tea.NewView(out), d)
}
