// Package keybindings defines the types and dispatch infrastructure for keyboard-driven
// interactions across the gou/app TUI. Key bindings are registered as data rows
// (KeyBinding structs) grouped by screen context. A Dispatcher resolves the current
// context and matches pressed keys to actions.
package keybindings

// KeyAction identifies a user-visible operation triggered by a key press.
// Names use colon-delimited groups (e.g., "scroll:down", "screen:toggle_transcript").
type KeyAction string

const (
	// Scroll actions
	ActionScrollDown     KeyAction = "scroll:down"
	ActionScrollUp       KeyAction = "scroll:up"
	ActionScrollPageDown KeyAction = "scroll:page_down"
	ActionScrollPageUp   KeyAction = "scroll:page_up"
	ActionScrollBottom   KeyAction = "scroll:bottom"
	ActionScrollTop      KeyAction = "scroll:top"
	ActionScrollHalfDown KeyAction = "scroll:half_down"
	ActionScrollHalfUp   KeyAction = "scroll:half_up"
	ActionScrollFullDown KeyAction = "scroll:full_down"
	ActionScrollFullUp   KeyAction = "scroll:full_up"
	ActionScrollLineDown KeyAction = "scroll:line_down"
	ActionScrollLineUp   KeyAction = "scroll:line_up"

	// Screen / transcript
	ActionToggleTranscript KeyAction = "screen:toggle_transcript"
	ActionExitTranscript   KeyAction = "screen:exit_transcript"
	ActionToggleShowAll    KeyAction = "screen:toggle_show_all"
	ActionToggleDump       KeyAction = "screen:toggle_dump"
	ActionOpenEditor       KeyAction = "screen:open_editor"
	ActionTranscriptSearch KeyAction = "transcript:search"
	ActionSearchNext       KeyAction = "transcript:search_next"
	ActionSearchPrev       KeyAction = "transcript:search_prev"

	// Input / slash / suggestions
	ActionToggleSlash       KeyAction = "input:toggle_slash"
	ActionSlashSelectNext   KeyAction = "slash:select_next"
	ActionSlashSelectPrev   KeyAction = "slash:select_prev"
	ActionSlashSelect       KeyAction = "slash:select" // Tab
	ActionSlashAccept       KeyAction = "slash:accept" // Enter with slash visible
	ActionSuggestNext       KeyAction = "suggest:next"
	ActionSuggestPrev       KeyAction = "suggest:prev"
	ActionSuggestAccept     KeyAction = "suggest:accept"
	ActionSuggestDismiss    KeyAction = "suggest:dismiss"

	// Query / app
	ActionSubmit         KeyAction = "input:submit"
	ActionInterrupt      KeyAction = "query:interrupt"
	ActionQuit           KeyAction = "app:quit"
	ActionForceRedraw    KeyAction = "app:redraw"
	ActionToggleFoldAll  KeyAction = "viewport:toggle_fold"
	ActionEvictAgents    KeyAction = "agent:evict_completed"

	// Manual render (f5/f6)
	ActionEnterManualRender KeyAction = "app:enter_manual_render"
	ActionFlushManualRender KeyAction = "app:flush_manual_render"
)

// KeyContext identifies the active UI region / mode and controls which binding
// rows are considered during dispatch. Only one context is returned by Resolve
// at a time (highest-priority match wins).
type KeyContext int

const (
	CtxModal             KeyContext = iota // Permission modal is showing (swallows most keys)
	CtxTranscript                          // Full transcript view
	CtxTranscriptDump                      // Dump mode inside transcript
	CtxTranscriptSearch                    // Search bar open in transcript
	CtxSuggestVisible                      // @-mention suggestion popup visible
	CtxSlashVisible                        // Slash-command list visible
	CtxPrompt                              // Normal prompt input mode
	CtxGlobal                              // Overrides that apply regardless of context (ctrl+c, ctrl+l)
)

// KeyBinding maps a single key to an action within a context.
// Order controls dispatch priority (lower = checked first) within the
// same context. Global bindings are tried after exact context bindings.
type KeyBinding struct {
	Key     string     // tea.KeyPressMsg.String() representation
	Context KeyContext // Which context this binding is active in
	Action  KeyAction  // The action to perform
	Order   int        // Lower = higher priority within the same context
}
