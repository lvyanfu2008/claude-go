package modelenv

import (
	"testing"
)

func clearLookupKeys(t *testing.T) {
	t.Helper()
	for _, k := range LookupKeys {
		t.Setenv(k, "")
	}
}

func TestResolveWithFallback_priority(t *testing.T) {
	t.Run("ccb_engine_wins", func(t *testing.T) {
		clearLookupKeys(t)
		t.Setenv("CCB_ENGINE_MODEL", "m-ccb")
		t.Setenv("ANTHROPIC_MODEL", "m-ant")
		if got := ResolveWithFallback("fb"); got != "m-ccb" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("anthropic_model_second", func(t *testing.T) {
		clearLookupKeys(t)
		t.Setenv("ANTHROPIC_MODEL", "m-ant")
		t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "m-sonnet")
		if got := ResolveWithFallback("fb"); got != "m-ant" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("default_sonnet_third", func(t *testing.T) {
		clearLookupKeys(t)
		t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "m-sonnet")
		t.Setenv("ANTHROPIC_DEFAULT_HAIKU_MODEL", "m-haiku")
		if got := ResolveWithFallback("fb"); got != "m-sonnet" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("fallback", func(t *testing.T) {
		clearLookupKeys(t)
		if got := ResolveWithFallback("fb-only"); got != "fb-only" {
			t.Fatalf("got %q", got)
		}
	})
}
