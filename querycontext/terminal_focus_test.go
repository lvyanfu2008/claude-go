package querycontext

import (
	"strings"
	"testing"
)

func TestSetTerminalFocused(t *testing.T) {
	ResetTerminalFocusState()

	// Default: unknown → treated as focused
	if !GetTerminalFocused() {
		t.Error("expected focused when state is unknown")
	}
	if GetTerminalFocusState() != "unknown" {
		t.Errorf("expected 'unknown', got %q", GetTerminalFocusState())
	}

	// Blur
	SetTerminalFocused(false)
	if GetTerminalFocused() {
		t.Error("expected not focused after blur")
	}
	if GetTerminalFocusState() != "blurred" {
		t.Errorf("expected 'blurred', got %q", GetTerminalFocusState())
	}

	// Refocus
	SetTerminalFocused(true)
	if !GetTerminalFocused() {
		t.Error("expected focused after refocus")
	}
	if GetTerminalFocusState() != "focused" {
		t.Errorf("expected 'focused', got %q", GetTerminalFocusState())
	}
}

func TestResetTerminalFocusState(t *testing.T) {
	SetTerminalFocused(false)
	ResetTerminalFocusState()

	if !GetTerminalFocused() {
		t.Error("expected focused after reset (unknown treated as focused)")
	}
	if GetTerminalFocusState() != "unknown" {
		t.Errorf("expected 'unknown' after reset, got %q", GetTerminalFocusState())
	}
}

func TestTerminalFocusContextValue_Focused(t *testing.T) {
	ResetTerminalFocusState()
	SetTerminalFocused(true)
	if v := TerminalFocusContextValue(); v != "" {
		t.Errorf("expected empty when focused, got %q", v)
	}
}

func TestTerminalFocusContextValue_Unknown(t *testing.T) {
	ResetTerminalFocusState()
	// unknown → treated as focused → empty value
	if v := TerminalFocusContextValue(); v != "" {
		t.Errorf("expected empty when unknown, got %q", v)
	}
}

func TestTerminalFocusContextValue_Blurred(t *testing.T) {
	ResetTerminalFocusState()
	SetTerminalFocused(false)
	v := TerminalFocusContextValue()
	if v == "" {
		t.Error("expected non-empty when blurred")
	}
	if !strings.Contains(v, "unfocused") {
		t.Errorf("expected 'unfocused' in value, got %q", v)
	}
}

func TestEnrichUserContext_TerminalFocus(t *testing.T) {
	ResetTerminalFocusState()
	SetTerminalFocused(false)
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "1")

	uc := map[string]string{"currentDate": "2026-05-19"}
	enrichUserContext(uc, FetchOpts{})

	tf, ok := uc["terminalFocus"]
	if !ok {
		t.Error("expected terminalFocus in user context when proactive active and unfocused")
	}
	if !strings.Contains(tf, "unfocused") {
		t.Errorf("unexpected terminalFocus value: %q", tf)
	}
}

func TestEnrichUserContext_TerminalFocus_NotActive(t *testing.T) {
	ResetTerminalFocusState()
	SetTerminalFocused(false)
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "")

	uc := map[string]string{"currentDate": "2026-05-19"}
	enrichUserContext(uc, FetchOpts{})

	if _, ok := uc["terminalFocus"]; ok {
		t.Error("expected no terminalFocus when proactive not active")
	}
}

func TestEnrichUserContext_TerminalFocus_Focused(t *testing.T) {
	ResetTerminalFocusState()
	SetTerminalFocused(true)
	t.Setenv("CLAUDE_CODE_GO_PROACTIVE_ACTIVE", "1")

	uc := map[string]string{"currentDate": "2026-05-19"}
	enrichUserContext(uc, FetchOpts{})

	if _, ok := uc["terminalFocus"]; ok {
		t.Error("expected no terminalFocus when focused, even with proactive active")
	}
}
