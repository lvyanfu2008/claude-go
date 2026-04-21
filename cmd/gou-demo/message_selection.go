package main

// isListViewportScrollKey reports keys forwarded to bubbles/viewport for the prompt message pane
// (see gouDemoMsgViewportKeyMap + handleMsgViewportScrollKey). Must run before m.pr.Update so ↑/↓
// scroll the list instead of being dropped or only affecting the prompt.
// Intentionally omit "j", "k", " ", and "b" so those keys type in the prompt (transcript mode still binds j/k via handleTranscriptKey).
func isListViewportScrollKey(s string) bool {
	switch s {
	case "up", "down",
		"pgup", "pgdown",
		"home", "end", "G", "shift+g",
		"ctrl+home", "ctrl+end",
		"ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f", "ctrl+n", "ctrl+p":
		return true
	default:
		return false
	}
}
