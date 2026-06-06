package keybindings

import tea "charm.land/bubbletea/v2"

// execute performs the action, delegating to Deps as needed.
// Returns (tea.Model, tea.Cmd). A nil Model means "no model change" (the
// caller should return its own Model reference).
func (d *Dispatcher) execute(action KeyAction, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch action {

	// ── Scroll ─────────────────────────────────────────────────────────────
	// When the bubbles/viewport message pane is active, forward scroll keys to
	// the viewport model (which handles its own scrolling). Otherwise use the
	// traditional scroll-manager methods.
	case ActionScrollDown:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollDown()
		return nil, nil
	case ActionScrollUp:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollUp()
		return nil, nil
	case ActionScrollHalfDown:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollHalfDown()
		return nil, nil
	case ActionScrollHalfUp:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollHalfUp()
		return nil, nil
	case ActionScrollFullDown:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollFullDown()
		return nil, nil
	case ActionScrollFullUp:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollFullUp()
		return nil, nil
	case ActionScrollLineDown:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollLineDown()
		return nil, nil
	case ActionScrollLineUp:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollLineUp()
		return nil, nil
	case ActionScrollBottom:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollBottom()
		return nil, nil
	case ActionScrollTop:
		if d.deps.MsgViewportWanted() {
			return nil, d.deps.HandleViewportScrollKey(msg)
		}
		d.deps.ScrollTop()
		return nil, nil

	// ── Viewport ───────────────────────────────────────────────────────────
	case ActionToggleFoldAll:
		d.deps.ToggleFoldAll()
		return nil, nil

	// ── Redraw / quit ──────────────────────────────────────────────────────
	case ActionForceRedraw:
		return nil, d.deps.RedrawCmd()
	case ActionQuit:
		return nil, d.deps.HandleQuit()
	case ActionInterrupt:
		return nil, d.deps.HandleInterrupt()

	// ── Screen ─────────────────────────────────────────────────────────────
	case ActionToggleTranscript:
		return nil, d.deps.HandleToggleTranscript()
	case ActionExitTranscript:
		return nil, d.deps.HandleExitTranscript()
	case ActionToggleShowAll:
		d.deps.HandleToggleShowAll()
		return nil, nil
	case ActionToggleDump:
		return nil, d.deps.HandleDump()
	case ActionOpenEditor:
		return nil, d.deps.HandleOpenEditor()

	// ── Transcript search ──────────────────────────────────────────────────
	case ActionTranscriptSearch:
		// In transcript search context, pass the key to the search bar.
		// In normal transcript context, open search (only '/') — handled by
		// HandleTranscriptSearchBarKey which also processes text chars.
		return nil, d.deps.HandleTranscriptSearchBarKey(msg)
	case ActionSearchNext:
		d.deps.HandleSearchNext()
		return nil, nil
	case ActionSearchPrev:
		d.deps.HandleSearchPrev()
		return nil, nil

	// ── Input / slash ──────────────────────────────────────────────────────
	case ActionToggleSlash:
		d.deps.HandleToggleSlash()
		return nil, nil
	case ActionSlashAccept:
		return d.deps.HandleSlashSubmit()

	// ── Slash list navigation ──────────────────────────────────────────────
	case ActionSlashSelectPrev:
		d.deps.HandleSlashSelectPrev()
		return nil, nil
	case ActionSlashSelectNext:
		d.deps.HandleSlashSelectNext()
		return nil, nil
	case ActionSlashSelect:
		d.deps.HandleSlashSelect()
		return nil, nil

	// ── Suggestions ────────────────────────────────────────────────────────
	case ActionSuggestAccept:
		d.deps.HandleSuggestAccept()
		return nil, nil
	case ActionSuggestDismiss:
		d.deps.HandleSuggestDismiss()
		return nil, nil
	case ActionSuggestPrev:
		d.deps.HandleSuggestPrev()
		return nil, nil
	case ActionSuggestNext:
		d.deps.HandleSuggestNext()
		return nil, nil

	// ── Manual render ──────────────────────────────────────────────────────
	case ActionEnterManualRender:
		d.deps.HandleEnterManualRender()
		return nil, nil
	case ActionFlushManualRender:
		return nil, d.deps.HandleFlushManualRender()

	// ── Unreachable (should be caught by Dispatch fallback) ────────────────
	default:
		return nil, nil
	}
}
