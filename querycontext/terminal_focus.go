package querycontext

import "sync"

// terminalFocusState mirrors TS TerminalFocusState in packages/@ant/ink/src/core/terminal-focus-state.ts.
// 'unknown' is the default for terminals that don't support focus reporting;
// consumers treat 'unknown' identically to 'focused'.
type terminalFocusState string

const (
	terminalFocused  terminalFocusState = "focused"
	terminalBlurred  terminalFocusState = "blurred"
	terminalUnknown  terminalFocusState = "unknown"
)

var (
	currentTerminalFocusState terminalFocusState = terminalUnknown
	terminalFocusMu           sync.RWMutex
)

// SetTerminalFocused updates the terminal focus state. Call this from the TUI
// when DECSET 1004 focus events are received (CSI I = focused, CSI O = blurred).
func SetTerminalFocused(focused bool) {
	terminalFocusMu.Lock()
	defer terminalFocusMu.Unlock()
	if focused {
		currentTerminalFocusState = terminalFocused
	} else {
		currentTerminalFocusState = terminalBlurred
	}
}

// GetTerminalFocused returns true unless the terminal is known to be blurred.
// 'unknown' state is treated as focused (optimistic).
func GetTerminalFocused() bool {
	terminalFocusMu.RLock()
	defer terminalFocusMu.RUnlock()
	return currentTerminalFocusState != terminalBlurred
}

// GetTerminalFocusState returns the current terminal focus state.
func GetTerminalFocusState() string {
	terminalFocusMu.RLock()
	defer terminalFocusMu.RUnlock()
	return string(currentTerminalFocusState)
}

// ResetTerminalFocusState resets the focus state to 'unknown'.
func ResetTerminalFocusState() {
	terminalFocusMu.Lock()
	defer terminalFocusMu.Unlock()
	currentTerminalFocusState = terminalUnknown
}

// TerminalFocusContextValue returns the value to inject into user context when
// terminal is unfocused and proactive mode is active. Returns empty string when
// the terminalFocus field should not be injected.
// Mirrors TS REPL.tsx lines 3316-3326:
//
//	terminalFocus: 'The terminal is unfocused — the user is not actively watching.'
func TerminalFocusContextValue() string {
	if GetTerminalFocused() {
		return ""
	}
	return "The terminal is unfocused — the user is not actively watching."
}
