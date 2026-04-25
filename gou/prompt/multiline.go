// Package prompt holds a minimal multiline prompt model for gou TUI (Bubble Tea).
//
// Two input styles (see [Model.SetEnterSubmits]):
//   - REPL (default): Enter submits; newline via Alt+Enter (Option+Enter on macOS when mapped to Meta), Ctrl+J (LF),
//     or Shift+Enter when the terminal sends LF instead of CR.
//   - Chat: Enter inserts newline when the terminal sends CR for both Enter and Shift+Enter; Alt+Enter submits.
package prompt

import (
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// Model is a lightweight multiline text buffer with cursor.
type Model struct {
	value     []rune
	cursor    int // index in value (0..len)
	width     int
	focused   bool
	submitted bool // true after Enter submit; caller clears buffer
	// enterSubmits: true = Enter (\r) submits, Alt+Enter inserts newline (REPL).
	// false = Enter/Shift+Enter insert newline when indistinguishable, Alt+Enter submits (chat / IDE terminals).
	enterSubmits bool
}

// New returns an empty focused model (REPL-style: Enter submits).
func New() Model {
	return Model{focused: true, width: 60, enterSubmits: true}
}

// SetEnterSubmits sets REPL vs chat input (see package doc). Default true.
func (m *Model) SetEnterSubmits(replEnterSubmits bool) {
	m.enterSubmits = replEnterSubmits
}

// EnterSubmits reports whether bare Enter submits (REPL mode).
func (m Model) EnterSubmits() bool {
	return m.enterSubmits
}

// Focused reports whether the field accepts editing keys.
func (m Model) Focused() bool { return m.focused }

// Focus enables editing.
func (m *Model) Focus() { m.focused = true }

// Blur disables editing (keys are ignored).
func (m *Model) Blur() { m.focused = false }

// Value returns the full UTF-8 text (interior newlines preserved).
func (m Model) Value() string {
	return string(m.value)
}

// CursorRuneIndex is the insert position in the buffer as a 0..len(rune value) rune index
// (alias of internal cursor; matches the slice index over UTF-8 runes in [Value] for ASCII text).
func (m Model) CursorRuneIndex() int {
	return m.cursor
}

// SetValue replaces buffer content and places cursor at end.
func (m *Model) SetValue(s string) {
	m.value = []rune(s)
	m.cursor = len(m.value)
	m.clampCursor()
	m.submitted = false
}

// SetWidth hints max runes per row for View truncation.
func (m *Model) SetWidth(w int) {
	if w < 1 {
		w = 1
	}
	m.width = w
}

// Submitted is true after Update handled Enter as submit.
func (m Model) Submitted() bool { return m.submitted }

// Update handles key and window messages.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	m.submitted = false
	if !m.focused {
		return nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width - 4)
		return nil
	case tea.PasteMsg:
		m.insertRunes([]rune(msg.Content))
		return nil
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	}
	return nil
}

func (m *Model) updateKey(msg tea.KeyPressMsg) tea.Cmd {
	repl := m.enterSubmits
	key := msg.Key()

	// Ctrl+J is LF (\n). Many terminals send Shift+Enter as LF; macOS Option+Enter
	// often arrives as ESC+LF (alt+ctrl+j), not alt+enter — handle before msg.String().
	if key.Code == 'j' && key.Mod.Contains(tea.ModCtrl) {
		m.insertRune('\n')
		return nil
	}

	if key.Code == tea.KeyEnter {
		if key.Mod.Contains(tea.ModAlt) {
			if repl {
				m.insertRune('\n')
			} else if strings.TrimSpace(m.Value()) != "" {
				m.submitted = true
			}
			return nil
		}
	}

	switch msg.String() {
	case "shift+enter":
		m.insertRune('\n')
		return nil
	case "alt+enter":
		if repl {
			m.insertRune('\n')
		} else if strings.TrimSpace(m.Value()) != "" {
			m.submitted = true
		}
		return nil
	case "enter":
		if repl {
			if strings.TrimSpace(m.Value()) != "" {
				m.submitted = true
			}
		} else {
			m.insertRune('\n')
		}
		return nil
	}

	if key.Code == tea.KeyBackspace || msg.String() == "backspace" {
		m.deleteBefore()
		return nil
	}
	if key.Code == tea.KeySpace || msg.String() == "space" {
		m.insertRune(' ')
		return nil
	}
	if key.Code == tea.KeyDelete || msg.String() == "ctrl+d" {
		m.deleteAfter()
		return nil
	}
	if key.Code == tea.KeyLeft || msg.String() == "ctrl+b" {
		m.moveRune(-1)
		return nil
	}
	if key.Code == tea.KeyRight || msg.String() == "ctrl+f" {
		m.moveRune(1)
		return nil
	}
	if key.Code == tea.KeyUp && key.Mod.Contains(tea.ModShift) {
		m.moveLine(-1)
		return nil
	}
	if key.Code == tea.KeyDown && key.Mod.Contains(tea.ModShift) {
		m.moveLine(1)
		return nil
	}
	if key.Code == tea.KeyHome || msg.String() == "ctrl+a" {
		m.cursorLineStart()
		return nil
	}
	if key.Code == tea.KeyEnd || msg.String() == "ctrl+e" {
		m.cursorLineEnd()
		return nil
	}
	if key.Text != "" {
		for _, r := range key.Text {
			switch r {
			case '\n', '\u0085', '\u2028', '\u2029':
				m.insertRune('\n')
				return nil
			}
		}
		m.insertRunes([]rune(key.Text))
		return nil
	}
	return nil
}

func (m *Model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > len(m.value) {
		m.cursor = len(m.value)
	}
}

func (m *Model) insertRune(r rune) {
	m.value = append(m.value[:m.cursor], append([]rune{r}, m.value[m.cursor:]...)...)
	m.cursor++
}

func (m *Model) insertRunes(r []rune) {
	if len(r) == 0 {
		return
	}
	m.value = append(m.value[:m.cursor], append(r, m.value[m.cursor:]...)...)
	m.cursor += len(r)
}

func (m *Model) deleteBefore() {
	if m.cursor == 0 {
		return
	}
	m.cursor--
	m.value = append(m.value[:m.cursor], m.value[m.cursor+1:]...)
}

func (m *Model) deleteAfter() {
	if m.cursor >= len(m.value) {
		return
	}
	m.value = append(m.value[:m.cursor], m.value[m.cursor+1:]...)
}

func (m *Model) moveRune(delta int) {
	m.cursor += delta
	m.clampCursor()
}

func (m *Model) lineBounds() (start, end int) {
	start = 0
	for i := 0; i < m.cursor && i < len(m.value); i++ {
		if m.value[i] == '\n' {
			start = i + 1
		}
	}
	end = len(m.value)
	for i := m.cursor; i < len(m.value); i++ {
		if m.value[i] == '\n' {
			end = i
			break
		}
	}
	return start, end
}

func (m *Model) cursorLineStart() {
	start, _ := m.lineBounds()
	m.cursor = start
}

func (m *Model) cursorLineEnd() {
	_, end := m.lineBounds()
	m.cursor = end
}

func (m *Model) moveLine(delta int) {
	if delta == 0 {
		return
	}
	lineStart, lineEnd := m.lineBounds()
	col := m.cursor - lineStart
	if delta < 0 {
		if lineStart == 0 {
			m.cursor = 0
			return
		}
		prevEnd := lineStart - 1
		prevStart := 0
		for i := prevEnd - 1; i >= 0; i-- {
			if m.value[i] == '\n' {
				prevStart = i + 1
				break
			}
		}
		prevLen := prevEnd - prevStart
		newCol := col
		if newCol > prevLen {
			newCol = prevLen
		}
		m.cursor = prevStart + newCol
		return
	}
	if lineEnd >= len(m.value) {
		m.cursor = len(m.value)
		return
	}
	nextStart := lineEnd + 1
	nextEnd := len(m.value)
	for i := nextStart; i < len(m.value); i++ {
		if m.value[i] == '\n' {
			nextEnd = i
			break
		}
	}
	nextLen := nextEnd - nextStart
	newCol := col
	if newCol > nextLen {
		newCol = nextLen
	}
	m.cursor = nextStart + newCol
}

func splitLines(v []rune) [][]rune {
	if len(v) == 0 {
		return [][]rune{{}}
	}
	var out [][]rune
	var cur []rune
	for _, r := range v {
		if r == '\n' {
			out = append(out, cur)
			cur = nil
		} else {
			cur = append(cur, r)
		}
	}
	out = append(out, cur)
	return out
}

func cursorLineCol(v []rune, cursor int) (line, col int) {
	line, col = 0, 0
	for i := 0; i < cursor && i < len(v); i++ {
		if v[i] == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}
	return line, col
}

// View renders logical lines; truncates long lines to width with "…"; shows █ at cursor.
func (m Model) View() string {
	w := m.width
	if w < 8 {
		w = 8
	}
	lines := splitLines(m.value)
	cl, cc := cursorLineCol(m.value, m.cursor)
	var parts []string
	for i, lr := range lines {
		rs := lr
		if len(rs) > w {
			rs = append(append([]rune(nil), rs[:w-1]...), '…')
		}
		s := string(rs)
		if i == cl {
			runes := []rune(s)
			c := cc
			if c > len(runes) {
				c = len(runes)
			}
			s = string(runes[:c]) + "█" + string(runes[c:])
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

// LineCount is logical lines (minimum 1).
func (m Model) LineCount() int {
	if len(m.value) == 0 {
		return 1
	}
	n := 1
	for _, r := range m.value {
		if r == '\n' {
			n++
		}
	}
	return n
}

// RuneCount returns utf8 rune count of Value.
func (m Model) RuneCount() int {
	return utf8.RuneCountInString(m.Value())
}
