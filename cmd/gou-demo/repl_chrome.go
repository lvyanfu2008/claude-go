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

// gouDemoTerminalTitleDisabled mirrors CLAUDE_CODE_DISABLE_TERMINAL_TITLE in REPL.tsx.
func gouDemoTerminalTitleDisabled() bool {
	return gouDemoEnvTruthy("CLAUDE_CODE_DISABLE_TERMINAL_TITLE")
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
		return "gou-demo  ↑↓ Pg F2 Enter Ctrl+J q"
	}
	return "gou-demo — ↑↓ scroll  PgUp/PgDn  End bottom  F2 slash  Enter send  Ctrl+J/Alt+Enter newline  q or Esc quit"
}

// replChromeTranscriptTopBar is the header line while TS-style transcript mode is active.
func replChromeTranscriptTopBar(narrow bool) string {
	if narrow {
		return "TRANSCRIPT  ↑↓ Pg / ctrl+o ctrl+e Esc"
	}
	return "TRANSCRIPT — frozen history · ↑↓ PgUp/PgDn End · ctrl+o close · ctrl+e expand · / search · Esc/q/ctrl+c close"
}

// replChromeFooterHint is the faint line under the prompt.
func replChromeFooterHint(narrow bool) string {
	if narrow {
		return "Ctrl+J newline · F2 · q"
	}
	return "Ctrl+J / Alt+Enter newline · Shift+↑↓ line · F2 commands · q or Esc quit"
}

// replChromePermissionFragment returns a short permission pill (empty when default in narrow).
func replChromePermissionFragment(mode types.PermissionMode, narrow bool) string {
	if mode == "" {
		mode = types.PermissionDefault
	}
	if narrow && mode == types.PermissionDefault {
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
