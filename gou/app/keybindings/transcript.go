package keybindings

// TranscriptBindings registers all key bindings for the transcript screen and its
// sub-contexts (search bar open, dump mode). Order reflects the original priority
// from handleTranscriptKey and the transcript switch in handleKeyMsgPreserving.
var TranscriptBindings = []KeyBinding{
	// ── Transcript: search bar open (CtxTranscriptSearch) ───────────────────
	{Key: "esc", Context: CtxTranscriptSearch, Action: ActionTranscriptSearch, Order: 10},
	{Key: "enter", Context: CtxTranscriptSearch, Action: ActionTranscriptSearch, Order: 10},
	{Key: "backspace", Context: CtxTranscriptSearch, Action: ActionTranscriptSearch, Order: 10},
	{Key: "ctrl+h", Context: CtxTranscriptSearch, Action: ActionTranscriptSearch, Order: 10},

	// ── Transcript: dump mode (CtxTranscriptDump) ───────────────────────────
	// In dump mode most keys are swallowed; only these exit.
	{Key: "esc", Context: CtxTranscriptDump, Action: ActionExitTranscript, Order: 10},
	{Key: "q", Context: CtxTranscriptDump, Action: ActionExitTranscript, Order: 10},
	{Key: "ctrl+c", Context: CtxTranscriptDump, Action: ActionExitTranscript, Order: 10},
	{Key: "ctrl+o", Context: CtxTranscriptDump, Action: ActionExitTranscript, Order: 10},

	// ── Transcript: normal mode (CtxTranscript) ─────────────────────────────
	// Search / navigation
	{Key: "/", Context: CtxTranscript, Action: ActionTranscriptSearch, Order: 20},
	{Key: "n", Context: CtxTranscript, Action: ActionSearchNext, Order: 30},
	{Key: "N", Context: CtxTranscript, Action: ActionSearchPrev, Order: 30},

	// Screen toggles
	{Key: "ctrl+e", Context: CtxTranscript, Action: ActionToggleShowAll, Order: 40},
	{Key: "ctrl+o", Context: CtxTranscript, Action: ActionExitTranscript, Order: 50},
	{Key: "[", Context: CtxTranscript, Action: ActionToggleDump, Order: 50},
	{Key: "v", Context: CtxTranscript, Action: ActionOpenEditor, Order: 50},

	// Force redraw
	{Key: "ctrl+l", Context: CtxTranscript, Action: ActionForceRedraw, Order: 60},

	// Exit keys
	{Key: "esc", Context: CtxTranscript, Action: ActionExitTranscript, Order: 70},
	{Key: "q", Context: CtxTranscript, Action: ActionExitTranscript, Order: 70},
	{Key: "ctrl+c", Context: CtxTranscript, Action: ActionExitTranscript, Order: 70},

	// Scroll keys (transcript uses per-line scroll + transcriptAfterManualScroll)
	{Key: "up", Context: CtxTranscript, Action: ActionScrollUp, Order: 80},
	{Key: "down", Context: CtxTranscript, Action: ActionScrollDown, Order: 80},
	{Key: "pgup", Context: CtxTranscript, Action: ActionScrollHalfUp, Order: 80},
	{Key: "pgdown", Context: CtxTranscript, Action: ActionScrollHalfDown, Order: 80},
	{Key: "home", Context: CtxTranscript, Action: ActionScrollTop, Order: 80},
	{Key: "end", Context: CtxTranscript, Action: ActionScrollBottom, Order: 80},

	// Global overrides (also apply in transcript)
	{Key: "ctrl+c", Context: CtxGlobal, Action: ActionInterrupt, Order: 5},
	{Key: "ctrl+l", Context: CtxGlobal, Action: ActionForceRedraw, Order: 5},
}
