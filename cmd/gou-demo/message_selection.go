package main

// isListViewportScrollKey reports keys that scroll the virtual message list (prompt or transcript pager).
func isListViewportScrollKey(s string) bool {
	switch s {
	case "up", "down", "pgup", "pgdown", "home", "ctrl+home", "end", "ctrl+end",
		"j", "k", "g", "G", "shift+g", "ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f", "b", " ", "ctrl+n", "ctrl+p":
		return true
	default:
		return false
	}
}
