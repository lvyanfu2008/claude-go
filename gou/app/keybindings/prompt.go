package keybindings

// PromptBindings registers all key bindings for the prompt input screen and its
// sub-contexts (modal, @-suggest, slash list). Order mirrors the original
// handleKeyMsgPreserving priority: permission modal → global overrides →
// transcript toggle → manual render → @-suggest → slash nav → scroll → escape/quit.
var PromptBindings = []KeyBinding{
	// ── Global overrides ─────────────────────────────────────────────────────
	// CtxModal keys are handled by the Dispatch fallback (HandleModalKey).
	{Key: "ctrl+c", Context: CtxGlobal, Action: ActionInterrupt, Order: 5},
	{Key: "ctrl+l", Context: CtxGlobal, Action: ActionForceRedraw, Order: 5},

	// ── Prompt mode: fixed-function keys ─────────────────────────────────────
	{Key: "ctrl+o", Context: CtxPrompt, Action: ActionToggleTranscript, Order: 10},
	{Key: "ctrl+y", Context: CtxPrompt, Action: ActionToggleFoldAll, Order: 10},
	{Key: "f2", Context: CtxPrompt, Action: ActionToggleSlash, Order: 20},
	{Key: "f5", Context: CtxPrompt, Action: ActionEnterManualRender, Order: 20},
	{Key: "f6", Context: CtxPrompt, Action: ActionFlushManualRender, Order: 20},

	// ── @-suggestion popup (CtxSuggestVisible) ──────────────────────────────
	{Key: "tab", Context: CtxSuggestVisible, Action: ActionSuggestAccept, Order: 30},
	{Key: "enter", Context: CtxSuggestVisible, Action: ActionSuggestAccept, Order: 30},
	{Key: "up", Context: CtxSuggestVisible, Action: ActionSuggestPrev, Order: 30},
	{Key: "down", Context: CtxSuggestVisible, Action: ActionSuggestNext, Order: 30},
	{Key: "esc", Context: CtxSuggestVisible, Action: ActionSuggestDismiss, Order: 30},

	// ── Slash-command list (CtxSlashVisible) ────────────────────────────────
	{Key: "enter", Context: CtxSlashVisible, Action: ActionSlashAccept, Order: 40},
	{Key: "tab", Context: CtxSlashVisible, Action: ActionSlashSelect, Order: 40},
	{Key: "up", Context: CtxSlashVisible, Action: ActionSlashSelectPrev, Order: 40},
	{Key: "down", Context: CtxSlashVisible, Action: ActionSlashSelectNext, Order: 40},

	// ── Prompt mode: scroll keys (must not conflict with slash/suggest) ─────
	{Key: "up", Context: CtxPrompt, Action: ActionScrollUp, Order: 60},
	{Key: "down", Context: CtxPrompt, Action: ActionScrollDown, Order: 60},
	{Key: "pgup", Context: CtxPrompt, Action: ActionScrollHalfUp, Order: 60},
	{Key: "pgdown", Context: CtxPrompt, Action: ActionScrollHalfDown, Order: 60},
	{Key: "space", Context: CtxPrompt, Action: ActionScrollHalfDown, Order: 60},
	{Key: "b", Context: CtxPrompt, Action: ActionScrollHalfUp, Order: 60},
	{Key: "home", Context: CtxPrompt, Action: ActionScrollTop, Order: 60},
	{Key: "ctrl+home", Context: CtxPrompt, Action: ActionScrollTop, Order: 60},
	{Key: "end", Context: CtxPrompt, Action: ActionScrollBottom, Order: 60},
	{Key: "G", Context: CtxPrompt, Action: ActionScrollBottom, Order: 60},
	{Key: "shift+g", Context: CtxPrompt, Action: ActionScrollBottom, Order: 60},
	{Key: "ctrl+end", Context: CtxPrompt, Action: ActionScrollBottom, Order: 60},
	{Key: "ctrl+u", Context: CtxPrompt, Action: ActionScrollHalfUp, Order: 60},
	{Key: "ctrl+d", Context: CtxPrompt, Action: ActionScrollHalfDown, Order: 60},
	{Key: "ctrl+b", Context: CtxPrompt, Action: ActionScrollFullUp, Order: 60},
	{Key: "ctrl+f", Context: CtxPrompt, Action: ActionScrollFullDown, Order: 60},
	{Key: "ctrl+n", Context: CtxPrompt, Action: ActionScrollLineDown, Order: 60},
	{Key: "ctrl+p", Context: CtxPrompt, Action: ActionScrollLineUp, Order: 60},

	// ── Quit / escape keys (lowest priority) ────────────────────────────────
	{Key: "esc", Context: CtxPrompt, Action: ActionQuit, Order: 200},
}
