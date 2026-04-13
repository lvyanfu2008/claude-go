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

func TestMultiline_altEnterInsertsNewline(t *testing.T) {
	m := New()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if got := m.Value(); got != "a\n" {
		t.Fatalf("value %q", got)
	}
	if m.Submitted() {
		t.Fatal("alt+enter must not submit")
	}
}

func TestMultiline_altEnterStringKey(t *testing.T) {
	// Same bytes bubbletea maps to alt+enter (see bubbletea key_test).
	m := New()
	m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	if m.Value() != "\n" {
		t.Fatalf("value %q", m.Value())
	}
}

func TestMultiline_altCtrlJ_insertsNewline(t *testing.T) {
	// ESC + LF: bubbletea uses KeyCtrlJ with Alt (not KeyEnter).
	m := New()
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ, Alt: true})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if got := m.Value(); got != "\nx" {
		t.Fatalf("value %q", got)
	}
	if m.Submitted() {
		t.Fatal("alt+ctrl+j must not submit")
	}
}
