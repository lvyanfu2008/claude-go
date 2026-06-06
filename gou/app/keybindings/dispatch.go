package keybindings

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

// Deps defines the interface the Dispatcher uses to query model state and
// execute complex actions. The app model (or a wrapper) implements this
// interface.
type Deps interface {
	// ── Context queries ──────────────────────────────────────────────────
	ModalActive() bool
	InTranscript() bool
	TranscriptSearchOpen() bool
	TranscriptDumpMode() bool
	SuggestVisible() bool
	SlashListVisible() bool
	SlashPanelActive() bool
	MsgViewportWanted() bool

	// ── Scroll ───────────────────────────────────────────────────────────
	ScrollUp()
	ScrollDown()
	ScrollHalfUp()
	ScrollHalfDown()
	ScrollFullUp()
	ScrollFullDown()
	ScrollLineUp()
	ScrollLineDown()
	ScrollTop()
	ScrollBottom()

	// ── Viewport scroll (bubbles/viewport) ──────────────────────────────
	HandleViewportScrollKey(msg tea.KeyPressMsg) tea.Cmd

	// ── Viewport ─────────────────────────────────────────────────────────
	ToggleFoldAll()

	// ── Cmds ─────────────────────────────────────────────────────────────
	RedrawCmd() tea.Cmd

	// ── Screen transitions ───────────────────────────────────────────────
	HandleToggleTranscript() tea.Cmd
	HandleExitTranscript() tea.Cmd

	// ── Query ────────────────────────────────────────────────────────────
	HandleQuit() tea.Cmd
	HandleInterrupt() tea.Cmd

	// ── Input modes ──────────────────────────────────────────────────────
	HandleToggleSlash()
	HandleEnterManualRender()
	HandleFlushManualRender() tea.Cmd

	// ── Transcript actions ───────────────────────────────────────────────
	HandleOpenEditor() tea.Cmd
	HandleDump() tea.Cmd
	HandleToggleShowAll()
	HandleTranscriptSearchBarKey(msg tea.KeyPressMsg) tea.Cmd
	HandleSearchNext()
	HandleSearchPrev()

	// ── Suggestions ──────────────────────────────────────────────────────
	HandleSuggestAccept()
	HandleSuggestDismiss()
	HandleSuggestPrev()
	HandleSuggestNext()

	// ── Slash list ───────────────────────────────────────────────────────
	HandleSlashSubmit() (tea.Model, tea.Cmd)
	HandleSlashSelectPrev()
	HandleSlashSelectNext()
	HandleSlashSelect()

	// ── Modal ────────────────────────────────────────────────────────────
	HandleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)
}

// Dispatcher resolves the current UI context and matches key presses to
// actions via the binding tables. It owns the sorted binding list and the
// reference to Deps.
type Dispatcher struct {
	deps     Deps
	bindings []KeyBinding
}

// NewDispatcher creates a Dispatcher with the combined binding tables
// sorted by Order (ascending).
func NewDispatcher(deps Deps) *Dispatcher {
	d := &Dispatcher{deps: deps}
	d.bindings = slices.Concat(PromptBindings, TranscriptBindings)
	slices.SortFunc(d.bindings, func(a, b KeyBinding) int { return a.Order - b.Order })
	return d
}

// Resolve returns the most specific active context. Priority (highest first):
//
//	CtxModal → CtxTranscriptDump → CtxTranscriptSearch → CtxTranscript
//	→ CtxSuggestVisible → CtxSlashVisible → CtxPrompt
func (d *Dispatcher) Resolve() KeyContext {
	if d.deps.ModalActive() {
		return CtxModal
	}
	if d.deps.InTranscript() {
		if d.deps.TranscriptDumpMode() {
			return CtxTranscriptDump
		}
		if d.deps.TranscriptSearchOpen() {
			return CtxTranscriptSearch
		}
		return CtxTranscript
	}
	if d.deps.SuggestVisible() {
		return CtxSuggestVisible
	}
	if d.deps.SlashListVisible() || d.deps.SlashPanelActive() {
		return CtxSlashVisible
	}
	return CtxPrompt
}

// Dispatch resolves the context and attempts to match the pressed key.
// Returns (tea.Model, tea.Cmd, handled). When handled is false the caller
// should fall through to the input widget. Model may be nil (no change).
func (d *Dispatcher) Dispatch(msg tea.KeyPressMsg) (_ tea.Model, _ tea.Cmd, handled bool) {
	ctx := d.Resolve()
	key := msg.String()

	// 1. Exact context match (bindings are pre-sorted by Order ascending).
	for _, b := range d.bindings {
		if b.Key == key && b.Context == ctx {
			m2, cmd := d.execute(b.Action, msg)
			return m2, cmd, true
		}
	}

	// 2. Global fallback (ctrl+c, ctrl+l, etc.).
	for _, b := range d.bindings {
		if b.Key == key && b.Context == CtxGlobal {
			m2, cmd := d.execute(b.Action, msg)
			return m2, cmd, true
		}
	}

	// 3. Context-specific swallows — no matching binding but the context
	//    consumes the key anyway.
	switch ctx {
	case CtxTranscriptSearch:
		// Unmatched keys are passed to the search bar for text input.
		cmd := d.deps.HandleTranscriptSearchBarKey(msg)
		return nil, cmd, true
	case CtxTranscriptDump, CtxTranscript:
		// Swallow all keys in transcript / dump mode.
		return nil, nil, true
	case CtxModal:
		m2, cmd := d.deps.HandleModalKey(msg)
		return m2, cmd, true
	case CtxSuggestVisible:
		// Do not swallow — suggest list is an overlay; let unmatched keys
		// fall through to the input widget so the user can type.
		return nil, nil, false
	case CtxSlashVisible:
		// Do not swallow — slash list is an overlay; let unmatched keys
		// fall through to the input widget so the user can type.
		return nil, nil, false
	}

	// 4. Not handled — caller falls through to input widget.
	return nil, nil, false
}
