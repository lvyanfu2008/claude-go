package main

// isListViewportScrollKey reports keys that scroll the virtual message list (prompt or transcript pager).
func isListViewportScrollKey(s string) bool {
	switch s {
	case "pgup", "pgdown", "ctrl+home", "ctrl+end",
		"ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f":
		return true
	default:
		return false
	}
}
