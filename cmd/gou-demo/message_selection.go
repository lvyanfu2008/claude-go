package main

// isListViewportScrollKey reports keys forwarded to bubbles/viewport for the prompt message pane
// (see gouDemoMsgViewportKeyMap + handleMsgViewportScrollKey). Must run before m.pr.Update so ↑/↓/j/k
// scroll the list instead of being dropped or only affecting the prompt.
// Intentionally omit " " and "b" (half-page in viewport) so Space and b still type in the input.
func isListViewportScrollKey(s string) bool {
	switch s {
	case "up", "down", "k", "j",
		"pgup", "pgdown",
		"home", "end", "g", "G", "shift+g",
		"ctrl+home", "ctrl+end",
		"ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f", "ctrl+n", "ctrl+p":
		return true
	default:
		return false
	}
}
