package prompt

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMultiline_newlineAndSubmit(t *testing.T) {
	m := New()
	m.SetWidth(40)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if got := m.Value(); got != "h\ni" {
		t.Fatalf("value %q", got)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.Submitted() {
		t.Fatal("expected submit")
	}
}

func TestMultiline_shiftLines(t *testing.T) {
	m := New()
	m.SetValue("ab\ncd")
	m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	if m.Value() != "ab\ncd" {
		t.Fatalf("value changed: %q", m.Value())
	}
}

func TestMultiline_emptyNoSubmit(t *testing.T) {
	var m Model = New()
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.Submitted() {
		t.Fatal("unexpected submit on empty")
	}
}
