package screen

import (
	"os"
	"runtime"
	"strings"
)

const (
	esc   = "\x1b"
	bel   = "\a"
	oscST = "\x1b\\" // OSC string terminator
)

// TerminalTitleDisabled checks whether terminal title updates are disabled.
func TerminalTitleDisabled() bool {
	return envTruthy("CLAUDE_CODE_DISABLE_TERMINAL_TITLE")
}

// AltScreenEnabled checks whether alternate screen is enabled.
func AltScreenEnabled() bool {
	return envTruthy("GOU_DEMO_ALT_SCREEN")
}

// MouseCellMotionEnabled checks whether SGR mouse tracking is enabled.
func MouseCellMotionEnabled() bool {
	if DisallowDisableMouse() {
		return true
	}
	if envTruthy("CLAUDE_CODE_DISABLE_MOUSE") || envTruthy("GOU_DEMO_DISABLE_MOUSE") {
		return false
	}
	return true
}

// DisallowDisableMouse checks whether mouse disable is forced off.
func DisallowDisableMouse() bool {
	return envTruthy("GOU_DEMO_DISALLOW_DISABLE_MOUSE")
}

// PromptEnterSubmits checks if Enter sends (REPL style) or adds a newline.
func PromptEnterSubmits() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_REPL_ENTER_SUBMITS")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

// MessageScrollbarStrip checks whether the one-column TUI scrollbar should be drawn.
func MessageScrollbarStrip() bool {
	if envTruthy("GOU_DEMO_NO_SCROLLBAR") {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_MESSAGE_SCROLLBAR")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" || v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
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

// SetWindowTitle returns an OSC 0 escape sequence to set the terminal title.
// Returns empty when disabled or on Windows.
func SetWindowTitle(plain string) string {
	if runtime.GOOS == "windows" || TerminalTitleDisabled() {
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

// ComposeTerminalTitle builds the plain tab title (before OSC wrapping).
func ComposeTerminalTitle(sessionID string, queryBusy, streamBufNonEmpty bool) string {
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

// TopBar returns the single-line header for prompt mode.
func TopBar(narrow bool) string {
	_ = narrow
	if PromptEnterSubmits() {
		return "gou-demo — ↑↓ scroll   Enter send · Alt/Option+Enter newline"
	}
	return "gou-demo — ↑↓ scroll   Enter newline · Alt+Enter send"
}

// TranscriptTopBar returns the header line for transcript mode.
func TranscriptTopBar(narrow bool) string {
	if narrow {
		return "TRANSCRIPT  jk Pg / gG ctrl+udbf ctrl+l ctrl+o ctrl+e Esc"
	}
	return "TRANSCRIPT — j/k line · g top · G End bottom · ctrl+u/d half-page · ctrl+b/f page · b page up · PgUpDn · / search · ctrl+l redraw · ctrl+o · ctrl+e · Esc"
}

// PermissionModeShortTitle mirrors permissionModeShortTitle in TS PermissionMode.ts.
func PermissionModeShortTitle(mode string) string {
	switch mode {
	case "", "default":
		return "Default"
	case "plan":
		return "Plan"
	case "accept_edits":
		return "Accept"
	case "bypass_permissions":
		return "Bypass"
	case "dont_ask":
		return "DontAsk"
	case "auto":
		return "Auto"
	case "bubble":
		return "Bubble"
	default:
		return "Default"
	}
}

// PermissionModeSymbol mirrors permissionModeSymbol in TS PermissionMode.ts.
func PermissionModeSymbol(mode string) string {
	switch mode {
	case "plan":
		return "⏸" // ⏸
	case "accept_edits", "bypass_permissions", "dont_ask", "auto":
		return "⏵⏵" // ⏵⏵
	default:
		return ""
	}
}

// PermissionFragment returns a short permission pill (empty when mode is default).
func PermissionFragment(mode string, narrow bool) string {
	_ = narrow
	if mode == "" {
		mode = "default"
	}
	if mode == "default" {
		return ""
	}
	sym := PermissionModeSymbol(mode)
	short := PermissionModeShortTitle(mode)
	if sym != "" {
		return sym + " " + short
	}
	return short
}
