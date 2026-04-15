// REPL-aligned UI chrome for gou-demo (terminal title OSC, narrow layout, permission mode labels).
// Mirrors Ink REPL / PromptInputFooter conventions without importing TS.
package main

import (
	"os"
	"runtime"
	"strings"

	"goc/types"
)

const (
	esc   = "\x1b"
	bel   = "\a"
	oscST = "\x1b\\" // OSC string terminator (Kitty; TS uses ST instead of BEL)
)

// UserPromptPointerGlyph matches npm figures.pointer in HighlightedThinkingText.tsx (user prompt line).
func UserPromptPointerGlyph() string {
	return "\u276f" // HEAVY RIGHT-POINTING ANGLE BRACKET ORNAMENT
}

// gouDemoTerminalTitleDisabled mirrors CLAUDE_CODE_DISABLE_TERMINAL_TITLE in REPL.tsx.
func gouDemoTerminalTitleDisabled() bool {
	return gouDemoEnvTruthy("CLAUDE_CODE_DISABLE_TERMINAL_TITLE")
}

// gouDemoVirtualScrollDisabled mirrors CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL in REPL.tsx / Messages.tsx.
// When true, gou-demo raises the virtual-list mounted-item cap (see virtualscroll.RangeInput.MaxMountedItemsOverride);
// it does not replicate Ink ScrollBox full non-virtual rendering.
func gouDemoVirtualScrollDisabled() bool {
	return gouDemoEnvTruthy("CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL")
}

// gouDemoMouseCellMotionEnabled mirrors TS isMouseTrackingEnabled (fullscreen.ts): when false, the program
// does not enable SGR mouse tracking (DEC 1006), so terminal-native drag-to-select / copy-on-select keeps working.
// Set CLAUDE_CODE_DISABLE_MOUSE=1 or GOU_DEMO_DISABLE_MOUSE=1 (same semantics as TS).
func gouDemoMouseCellMotionEnabled() bool {
	if gouDemoEnvTruthy("CLAUDE_CODE_DISABLE_MOUSE") || gouDemoEnvTruthy("GOU_DEMO_DISABLE_MOUSE") {
		return false
	}
	return true
}

// gouDemoAltScreenEnabled opts into tea.WithAltScreen: the TUI uses the terminal alternate buffer so the host
// scrollback is not mixed with redraws; the wheel targets the in-app message pane more reliably. On exit the
// previous screen is restored.
func gouDemoAltScreenEnabled() bool {
	return gouDemoEnvTruthy("GOU_DEMO_ALT_SCREEN")
}

// gouDemoMsgHistoryBrowseReleaseEnabled is go-tui/main/test.go style release: when the bubbles message viewport is
// at the top, one wheel-up in the pane runs tea.DisableMouse so the host can scroll terminal history; any key
// restores tea.EnableMouseCellMotion. Opt out with GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE=0|false|off|no.
func gouDemoMsgHistoryBrowseReleaseEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

// gouDemoMessageScrollbarStrip draws a one-column TUI scrollbar beside the message list when content overflows.
// Default off (plain bubbles/viewport like go-tui). Opt in with GOU_DEMO_MESSAGE_SCROLLBAR=1.
// GOU_DEMO_NO_SCROLLBAR=1 still forces the strip off (e.g. legacy scripts).
func gouDemoMessageScrollbarStrip() bool {
	if gouDemoEnvTruthy("GOU_DEMO_NO_SCROLLBAR") {
		return false
	}
	return gouDemoEnvTruthy("GOU_DEMO_MESSAGE_SCROLLBAR")
}

func sanitizeWindowTitle(s string) string {
	s = strings.ReplaceAll(s, "\x1b", "")
	s = strings.ReplaceAll(s, bel, "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return strings.TrimSpace(s)
}

// oscSetWindowTitle returns OSC 0 (set icon + window title) or empty when disabled / Windows.
func oscSetWindowTitle(plain string) string {
	if runtime.GOOS == "windows" || gouDemoTerminalTitleDisabled() {
		return ""
	}
	plain = sanitizeWindowTitle(plain)
	if plain == "" {
		plain = "gou-demo"
	}
	term := bel
	if strings.TrimSpace(os.Getenv("KITTY_WINDOW_ID")) != "" {
		term = oscST
	}
	return esc + "]0;" + plain + term
}

// gouDemoPermissionModeFromEnv parses CLAUDE_CODE_PERMISSION_MODE (TS toolPermissionContext.mode).
func gouDemoPermissionModeFromEnv() types.PermissionMode {
	s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_PERMISSION_MODE"))
	if s == "" {
		return types.PermissionDefault
	}
	switch types.PermissionMode(s) {
	case types.PermissionAcceptEdits,
		types.PermissionBypassPermissions,
		types.PermissionDefault,
		types.PermissionDontAsk,
		types.PermissionPlan,
		types.PermissionAuto,
		types.PermissionBubble:
		return types.PermissionMode(s)
	default:
		return types.PermissionDefault
	}
}

// permissionModeShortTitle mirrors permissionModeShortTitle in TS PermissionMode.ts.
func permissionModeShortTitle(mode types.PermissionMode) string {
	switch mode {
	case "", types.PermissionDefault:
		return "Default"
	case types.PermissionPlan:
		return "Plan"
	case types.PermissionAcceptEdits:
		return "Accept"
	case types.PermissionBypassPermissions:
		return "Bypass"
	case types.PermissionDontAsk:
		return "DontAsk"
	case types.PermissionAuto:
		return "Auto"
	case types.PermissionBubble:
		return "Bubble"
	default:
		return "Default"
	}
}

// permissionModeSymbol mirrors permissionModeSymbol in TS PermissionMode.ts (subset).
func permissionModeSymbol(mode types.PermissionMode) string {
	switch mode {
	case types.PermissionPlan:
		return "\u23f8" // ⏸ PAUSE_ICON
	case types.PermissionAcceptEdits, types.PermissionBypassPermissions, types.PermissionDontAsk, types.PermissionAuto:
		return "\u23f5\u23f5" // ⏵⏵
	default:
		return ""
	}
}

// replChromeTopBar returns the single-line header (TS isNarrow uses columns < 80).
func replChromeTopBar(narrow bool) string {
	if narrow {
		return "gou-demo  ↑↓ Pg F2 Ctrl+l Enter Shift+Enter q"
	}
	return "gou-demo — ↑↓ scroll  PgUp/PgDn  End bottom  F2 slash  Ctrl+l redraw  Enter send  Shift+Enter/Ctrl+J/Alt+Enter newline  q or Esc quit"
}

// replChromeTranscriptTopBar is the header line while TS-style transcript mode is active.
func replChromeTranscriptTopBar(narrow bool) string {
	if narrow {
		return "TRANSCRIPT  jk Pg / gG ctrl+udbf ctrl+l ctrl+o ctrl+e Esc"
	}
	return "TRANSCRIPT — j/k line · g top · G End bottom · ctrl+u/d half-page · ctrl+b/f page · b page up · PgUpDn · / search · ctrl+l redraw · ctrl+o · ctrl+e · Esc"
}

// replChromeFooterHint is reserved for a faint line under the prompt; shortcuts are omitted by design.
func replChromeFooterHint(narrow bool) string {
	_ = narrow
	return ""
}

// replChromePermissionFragment returns a short permission pill (empty when mode is default).
func replChromePermissionFragment(mode types.PermissionMode, narrow bool) string {
	_ = narrow
	if mode == "" {
		mode = types.PermissionDefault
	}
	if mode == types.PermissionDefault {
		return ""
	}
	sym := permissionModeSymbol(mode)
	short := permissionModeShortTitle(mode)
	if sym != "" {
		return sym + " " + short
	}
	return short
}

// replChromeComposeTerminalTitle builds the plain tab title (before OSC wrapping).
func replChromeComposeTerminalTitle(sessionID string, queryBusy, streamBufNonEmpty bool) string {
	base := "gou-demo"
	sid := strings.TrimSpace(sessionID)
	if sid != "" && sid != "demo" {
		if len(sid) > 16 {
			sid = sid[:14] + "…"
		}
		base = base + " · " + sid
	}
	if queryBusy || streamBufNonEmpty {
		return "… " + base
	}
	return base
}
