package main

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"goc/gou/prompt"
	"goc/types"
)

func TestOscSetWindowTitle_nonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("OSC title skipped on windows")
	}
	t.Setenv("CLAUDE_CODE_DISABLE_TERMINAL_TITLE", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	s := oscSetWindowTitle("hello")
	if s == "" {
		t.Fatal("expected OSC on non-windows test env")
	}
	if !strings.HasPrefix(s, "\x1b]0;") {
		t.Fatalf("prefix: %q", s)
	}
	if !strings.HasSuffix(s, "\a") {
		t.Fatalf("want BEL suffix got %q", s)
	}
}

func TestOscSetWindowTitle_disabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_TERMINAL_TITLE", "1")
	if oscSetWindowTitle("x") != "" {
		t.Fatal("want empty when disabled")
	}
}

func TestGouDemoVirtualScrollDisabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL", "")
	if gouDemoVirtualScrollDisabled() {
		t.Fatal("empty env should be false")
	}
	t.Setenv("CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL", "1")
	if !gouDemoVirtualScrollDisabled() {
		t.Fatal("truthy env should enable disable-virtual-scroll mode")
	}
}

func TestGouDemoMsgHistoryBrowseReleaseEnabled(t *testing.T) {
	t.Setenv("GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE", "")
	t.Setenv("GOU_DEMO_DISALLOW_DISABLE_MOUSE", "")
	if !gouDemoMsgHistoryBrowseReleaseEnabled() {
		t.Fatal("default should enable history mouse release (go-tui test.go parity)")
	}
	t.Setenv("GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE", "0")
	if gouDemoMsgHistoryBrowseReleaseEnabled() {
		t.Fatal("0 should disable")
	}
}

func TestGouDemoDisallowDisableMouse_overridesDisableEnvs(t *testing.T) {
	t.Setenv("GOU_DEMO_DISALLOW_DISABLE_MOUSE", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_MOUSE", "1")
	t.Setenv("GOU_DEMO_DISABLE_MOUSE", "1")
	if !gouDemoMouseCellMotionEnabled() {
		t.Fatal("DISALLOW should force mouse cell motion on")
	}
	if gouDemoMsgHistoryBrowseReleaseEnabled() {
		t.Fatal("DISALLOW should disable history-browse mouse release (no tea.DisableMouse path)")
	}
}

func TestGouDemoAltScreenEnabled(t *testing.T) {
	t.Setenv("GOU_DEMO_ALT_SCREEN", "")
	if gouDemoAltScreenEnabled() {
		t.Fatal("empty env should be false")
	}
	t.Setenv("GOU_DEMO_ALT_SCREEN", "1")
	if !gouDemoAltScreenEnabled() {
		t.Fatal("truthy GOU_DEMO_ALT_SCREEN should enable alt screen")
	}
}

func TestReplChromeFooterHint_empty(t *testing.T) {
	t.Setenv("GOU_DEMO_DISABLE_MOUSE", "")
	t.Setenv("CLAUDE_CODE_DISABLE_MOUSE", "")
	for _, narrow := range []bool{false, true} {
		if s := replChromeFooterHint(narrow); s != "" {
			t.Fatalf("narrow=%v: want empty got %q", narrow, s)
		}
	}
}

func TestOscSetWindowTitle_kittyST(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("OSC title skipped on windows")
	}
	t.Setenv("CLAUDE_CODE_DISABLE_TERMINAL_TITLE", "")
	t.Setenv("KITTY_WINDOW_ID", "1")
	s := oscSetWindowTitle("k")
	if !strings.HasSuffix(s, "\x1b\\") {
		t.Fatalf("want ST suffix got %q", s)
	}
}

func TestGouDemoPermissionModeFromEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_PERMISSION_MODE", "plan")
	if got := gouDemoPermissionModeFromEnv(); got != types.PermissionPlan {
		t.Fatalf("got %q", got)
	}
	t.Setenv("CLAUDE_CODE_PERMISSION_MODE", "nope")
	if got := gouDemoPermissionModeFromEnv(); got != types.PermissionDefault {
		t.Fatalf("invalid: got %q", got)
	}
}

func TestReplChromeComposeTerminalTitle(t *testing.T) {
	if got := replChromeComposeTerminalTitle("demo", false, false); got != "gou-demo" {
		t.Fatalf("%q", got)
	}
	if got := replChromeComposeTerminalTitle("abc-uuid-here", true, false); !strings.HasPrefix(got, "… ") {
		t.Fatalf("busy prefix: %q", got)
	}
}

func TestPermissionModeSymbolPlan(t *testing.T) {
	if permissionModeSymbol(types.PermissionPlan) != "\u23f8" {
		t.Fatalf("plan symbol")
	}
}

func TestUserPromptPointerGlyph(t *testing.T) {
	if UserPromptPointerGlyph() != ">" {
		t.Fatalf("pointer glyph")
	}
}

func TestUserMessageHasPromptText(t *testing.T) {
	raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": "hi"}})
	m := types.Message{Type: types.MessageTypeUser, Content: raw}
	if !userMessageHasPromptText(m) {
		t.Fatal("expected true for text user message")
	}
	raw2, _ := json.Marshal([]map[string]string{{"type": "tool_result", "tool_use_id": "x", "content": "ok"}})
	m2 := types.Message{Type: types.MessageTypeUser, Content: raw2}
	if userMessageHasPromptText(m2) {
		t.Fatal("tool_result-only user message should not get pointer")
	}
}

func TestUserMessageHasPromptText_messageEnvelopeOnly(t *testing.T) {
	// API-shaped row: Content empty, body in Message.{role,content} string (same as messagerow.NormalizeMessageJSON).
	inner := `{"role":"user","content":"你好"}`
	m := types.Message{Type: types.MessageTypeUser, Message: []byte(inner)}
	if !userMessageHasPromptText(m) {
		t.Fatal("expected true after normalizing Message envelope into Content")
	}
	out := withUserPromptPointerIfNeeded(m, "你好")
	if !strings.Contains(out, UserPromptPointerGlyph()) {
		t.Fatalf("expected > prefix on body line: %q", out)
	}
}

func TestWithUserPromptPointerIfNeeded(t *testing.T) {
	raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": "hello"}})
	m := types.Message{Type: types.MessageTypeUser, Content: raw}
	out := withUserPromptPointerIfNeeded(m, "body line")
	if !strings.Contains(out, "body line") {
		t.Fatalf("missing body: %q", out)
	}
	if !strings.Contains(out, UserPromptPointerGlyph()) {
		t.Fatalf("missing pointer: %q", out)
	}
}

func TestUserInputViewWithPromptPrefix_firstLine(t *testing.T) {
	m := &model{pr: prompt.New()}
	m.pr.SetValue("hi")
	v := userInputViewWithPromptPrefix(m)
	if !strings.Contains(v, ">") || !strings.Contains(v, "hi") {
		t.Fatalf("want > prefix on same line as text: %q", v)
	}
	if strings.HasPrefix(v, "hi") {
		t.Fatalf("expected prefix before body: %q", v)
	}
}
