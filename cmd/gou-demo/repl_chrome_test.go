package main

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

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
	if UserPromptPointerGlyph() != "\u276f" {
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
