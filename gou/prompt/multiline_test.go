package prompt

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func kpText(text string, code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

func TestMultiline_newlineAndSubmit(t *testing.T) {
	m := New()
	m.SetWidth(40)
	m.Update(kpText("h", 'h'))
	m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl}))
	m.Update(kpText("i", 'i'))
	if got := m.Value(); got != "h\ni" {
		t.Fatalf("value %q", got)
	}
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if !m.Submitted() {
		t.Fatal("expected submit")
	}
}

func TestMultiline_shiftLines(t *testing.T) {
	m := New()
	m.SetValue("ab\ncd")
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp, Mod: tea.ModShift}))
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown, Mod: tea.ModShift}))
	if m.Value() != "ab\ncd" {
		t.Fatalf("value changed: %q", m.Value())
	}
}

func TestMultiline_emptyNoSubmit(t *testing.T) {
	var m Model = New()
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if m.Submitted() {
		t.Fatal("unexpected submit on empty")
	}
}

func TestMultiline_altEnterInsertsNewline(t *testing.T) {
	m := New()
	m.Update(kpText("a", 'a'))
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModAlt}))
	if got := m.Value(); got != "a\n" {
		t.Fatalf("value %q", got)
	}
	if m.Submitted() {
		t.Fatal("alt+enter must not submit")
	}
}

func TestMultiline_altEnterStringKey(t *testing.T) {
	m := New()
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModAlt}))
	if m.Value() != "\n" {
		t.Fatalf("value %q", m.Value())
	}
}

func TestMultiline_altCtrlJ_insertsNewline(t *testing.T) {
	m := New()
	m.Update(tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModAlt | tea.ModCtrl}))
	m.Update(kpText("x", 'x'))
	if got := m.Value(); got != "\nx" {
		t.Fatalf("value %q", got)
	}
	if m.Submitted() {
		t.Fatal("alt+ctrl+j must not submit")
	}
}

func TestMultiline_chatMode_enterInsertsNewline_altEnterSubmits(t *testing.T) {
	m := New()
	m.SetEnterSubmits(false)
	m.Update(kpText("a", 'a'))
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got := m.Value(); got != "a\n" {
		t.Fatalf("value %q", got)
	}
	if m.Submitted() {
		t.Fatal("bare Enter must not submit in chat mode")
	}
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModAlt}))
	if !m.Submitted() {
		t.Fatal("alt+enter should submit")
	}
}

func TestMultiline_chatMode_shiftEnterSameAsEnter(t *testing.T) {
	m := New()
	m.SetEnterSubmits(false)
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if got := m.Value(); got != "\n\n" {
		t.Fatalf("value %q", got)
	}
}
