// Package update provides the message dispatch framework for the gou TUI.
// Message handlers are split by message type into separate files within this package.
package update

import (
	tea "charm.land/bubbletea/v2"
	"goc/gou/app/config"
	"goc/gou/app/keybindings"
	"goc/gou/app/state"
	"goc/gou/ccbstream"
	"goc/gou/prompt"
)

// Deps defines the interface the Dispatcher uses to access model state and
// execute complex actions. The app model implements this interface.
//
// Method names use a Get prefix to avoid shadowing embedded state fields
// on the model struct (e.g. model embeds *state.Layout, so GetLayout()
// is used instead of Layout()).
type Deps interface {
	// Model returns the current tea.Model (typically self).
	Model() tea.Model

	// ── State accessors ──────────────────────────────────────────────────
	GetLayout() *state.Layout
	GetScreen() *state.Screen
	GetScroll() *state.Scroll
	GetQuery() *state.Query
	GetModal() *state.Modal
	GetInput() *state.Input
	GetViewport() *state.Viewport
	GetAgent() *state.Agent
	GetMessageTracking() *state.MessageTracking
	GetConversation() *state.Conversation
	GetManualRender() *state.ManualRender

	// ── Modal / intercept ────────────────────────────────────────────────
	ModalQuestionActive() bool
	HandleModalQuestion(msg tea.Msg) (tea.Model, tea.Cmd)

	// ── Manual render ────────────────────────────────────────────────────
	ManualRenderActive() bool
	BufferManualRenderEvent(msg tea.Msg)
	ShouldBufferInManualRender(msg tea.Msg) bool

	// ── Action methods ───────────────────────────────────────────────────
	RebuildHeightCache()
	ClearTranscriptSearchState()
	EndQuerySpinner()
	MaybeRecordTranscript()
	AnyToolSummaryDelayPending() bool
	HandleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)

	// ── App-internal message dispatch ────────────────────────────────────
	TryHandleAppMessage(msg tea.Msg) (handled bool, model tea.Model, cmd tea.Cmd)
}

// Dispatcher owns the message dispatch switch. It delegates to Deps for
// model state access and to keybindings.Dispatcher for keyboard handling.
type Dispatcher struct {
	deps Deps
	kb   *keybindings.Dispatcher
}

// NewDispatcher creates a Dispatcher.
func NewDispatcher(deps Deps, kb *keybindings.Dispatcher) *Dispatcher {
	return &Dispatcher{deps: deps, kb: kb}
}

// Update handles the main message dispatch. It mirrors the structure of the
// original model.Update() but delegates handler bodies to this package.
func (d *Dispatcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// 1. Modal question UI intercept — fully interactive sub-model.
	if d.deps.ModalQuestionActive() {
		return d.deps.HandleModalQuestion(msg)
	}

	// 2. Manual render mode buffering.
	if d.deps.ManualRenderActive() && d.shouldBufferInManualRender(msg) {
		d.deps.BufferManualRenderEvent(msg)
		return d.deps.Model(), nil
	}

	// 3. Main dispatch switch — external / known message types.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return d.handleWindowSize(msg)

	case config.QueryYieldMsg:
		return d.handleQueryYield(msg)

	case config.StreamEventMsg:
		return d.handleStreamEvent(msg)

	case config.StreamingToolUsesMsg:
		return d.handleStreamingToolUses(msg)

	case config.SpinnerTickMsg:
		return d.handleSpinnerTick(msg)

	case config.QueryDoneMsg:
		return d.handleQueryDone(msg)

	case config.MemoryAppendMsg:
		return d.handleMemoryAppend(msg)

	case config.CompactPhaseMsg:
		return d.handleCompactPhase(msg)

	case config.ToolSummaryDelayTickMsg:
		return d.handleToolSummaryDelayTick(msg)

	case ccbstream.Msg:
		return d.handleCCBStream(msg)

	case tea.KeyPressMsg:
		return d.deps.HandleKeyMsg(msg)
	}

	// 4. App-internal messages (types defined in the app package).
	if handled, model, cmd := d.deps.TryHandleAppMessage(msg); handled {
		return model, cmd
	}

	// 5. Fallthrough for unknown messages.
	if syn, ok := prompt.SyntheticTTYKeyFromUnknownMsg(msg); ok {
		return d.deps.HandleKeyMsg(syn)
	}
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetInput().PR.Update(msg)
	}
	return d.deps.Model(), nil
}

// shouldBufferInManualRender returns true when the message should be queued
// during manual render mode instead of processed immediately.
func (d *Dispatcher) shouldBufferInManualRender(msg tea.Msg) bool {
	switch msg.(type) {
	case ccbstream.Msg,
		config.QueryDoneMsg,
		config.QueryYieldMsg,
		config.StreamEventMsg,
		config.SpinnerTickMsg,
		config.StreamingToolUsesMsg,
		config.ToolSummaryDelayTickMsg,
		config.MemoryAppendMsg,
		config.CompactPhaseMsg:
		return true
	}
	return false
}
