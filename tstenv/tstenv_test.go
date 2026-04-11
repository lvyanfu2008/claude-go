package tstenv

import (
	"testing"
)

func TestGetToolSearchMode_defaultsTst(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "")
	if got := GetToolSearchMode(); got != "tst" {
		t.Fatalf("got %q", got)
	}
}

func TestGetToolSearchMode_autoZeroIsTst(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:0")
	if got := GetToolSearchMode(); got != "tst" {
		t.Fatalf("got %q want tst", got)
	}
}

func TestGetToolSearchMode_auto100Standard(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "auto:100")
	if got := GetToolSearchMode(); got != "standard" {
		t.Fatalf("got %q want standard", got)
	}
}
