package prompt

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNormalizeTTYNewlineKey(t *testing.T) {
	t.Parallel()
	nel := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\u0085'}}
	got := NormalizeTTYNewlineKey(nel)
	if got.Type != tea.KeyCtrlJ || len(got.Runes) != 0 {
		t.Fatalf("NEL: got %#v want KeyCtrlJ", got)
	}
	unchanged := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	got2 := NormalizeTTYNewlineKey(unchanged)
	if got2.Type != unchanged.Type || len(got2.Runes) != 1 || got2.Runes[0] != 'x' {
		t.Fatalf("non-line-separator rune should be unchanged, got %#v", got2)
	}
}

func TestIsKittyModifiedEnterCSI(t *testing.T) {
	t.Parallel()
	cases := []struct {
		b    []byte
		want bool
	}{
		{[]byte("\x1b[13;2u"), true},
		{[]byte("\x1b[13:2u"), true},
		{[]byte("\x1b[13u"), false},
		{[]byte("\x1b[12;2u"), false},
		{[]byte("x"), false},
	}
	for _, tc := range cases {
		if got := isKittyModifiedEnterCSI(tc.b); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.b, got, tc.want)
		}
	}
}
