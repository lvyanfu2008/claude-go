package keybindings

import "strings"

// NonRebindableShortcuts defines keyboard shortcuts that cannot be rebound
var NonRebindableShortcuts = []ReservedShortcut{
	{Key: "ctrl+c", Reason: "Cannot be rebound - used for interrupt/exit (hardcoded)"},
	{Key: "ctrl+d", Reason: "Cannot be rebound - used for exit (hardcoded)"},
	{Key: "ctrl+m", Reason: "Cannot be rebound - identical to Enter in terminals (both send CR)"},
}

// TerminalReservedShortcuts defines shortcuts that may conflict with terminal operations
var TerminalReservedShortcuts = []ReservedShortcut{
	{Key: "ctrl+z", Reason: "Unix process suspend (SIGTSTP) (may conflict)"},
	{Key: "ctrl+\\", Reason: "Terminal quit signal (SIGQUIT) (will not work)"},
}

// MacOSReservedShortcuts defines shortcuts reserved by macOS
var MacOSReservedShortcuts = []ReservedShortcut{
	{Key: "cmd+c", Reason: "macOS system copy"},
	{Key: "cmd+v", Reason: "macOS system paste"},
	{Key: "cmd+x", Reason: "macOS system cut"},
	{Key: "cmd+q", Reason: "macOS quit application"},
	{Key: "cmd+w", Reason: "macOS close window/tab"},
	{Key: "cmd+tab", Reason: "macOS app switcher"},
	{Key: "cmd+space", Reason: "macOS Spotlight"},
}

// CommonToolConflicts defines shortcuts that conflict with common terminal tools
var CommonToolConflicts = []ReservedShortcut{
	{Key: "ctrl+b", Reason: "tmux prefix key"},
	{Key: "ctrl+a", Reason: "screen prefix key / beginning of line"},
}

// AllReservedShortcuts returns all reserved shortcuts
func AllReservedShortcuts() []ReservedShortcut {
	var all []ReservedShortcut
	all = append(all, NonRebindableShortcuts...)
	all = append(all, TerminalReservedShortcuts...)
	all = append(all, MacOSReservedShortcuts...)
	all = append(all, CommonToolConflicts...)
	return all
}

// normalizeKeyForComparison normalizes a key string for comparison
func normalizeKeyForComparison(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// IsReservedKey checks if a key is reserved and returns the reason if so
func IsReservedKey(key string) (bool, string) {
	normalizedKey := normalizeKeyForComparison(key)
	
	for _, reserved := range AllReservedShortcuts() {
		if normalizeKeyForComparison(reserved.Key) == normalizedKey {
			return true, reserved.Reason
		}
	}
	return false, ""
}

// IsNonRebindableKey checks if a key is strictly non-rebindable (error level)
func IsNonRebindableKey(key string) (bool, string) {
	normalizedKey := normalizeKeyForComparison(key)
	
	for _, reserved := range NonRebindableShortcuts {
		if normalizeKeyForComparison(reserved.Key) == normalizedKey {
			return true, reserved.Reason
		}
	}
	return false, ""
}