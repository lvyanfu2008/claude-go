package prompt

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNormalizeTTYNewlineKey_mapsNELToCtrlJ(t *testing.T) {
	nel := tea.KeyPressMsg(tea.Key{Text: "\u0085", Code: '\u0085'})
	got := NormalizeTTYNewlineKey(nel)
	if got.String() != "ctrl+j" {
		t.Fatalf("expected ctrl+j, got %q", got.String())
	}
	unchanged := tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'})
	if NormalizeTTYNewlineKey(unchanged).String() != "x" {
		t.Fatal("expected plain x unchanged")
	}
}
