// Viewport line copy: keyboard-driven range copy inspired by go-tui/main/test_ignore.go
// (space toggles mode there; here ctrl+; / f3 toggles because space scrolls the message pane).
package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *model) clearMsgLineCopyMode() {
	m.msgLineCopyMode = false
	m.msgLineCopyStart = 0
	m.msgLineCopyEnd = 0
}

func (m *model) msgViewportPlainLineCount() int {
	if m.msgVpPlainDoc == "" {
		return 0
	}
	return strings.Count(m.msgVpPlainDoc, "\n") + 1
}

func (m *model) enterMsgLineCopyMode() {
	n := m.msgViewportPlainLineCount()
	if n < 1 {
		m.copyStatus = "line copy: empty pane"
		return
	}
	m.msgLineCopyMode = true
	center := m.msgViewport.YOffset + listViewportH(m)/2
	if center < 0 {
		center = 0
	}
	if center >= n {
		center = n - 1
	}
	m.msgLineCopyStart = center
	m.msgLineCopyEnd = center
	m.copyStatus = "line copy: ↑/↓ j/k move, c copy lines, ctrl+a copy all, space/esc/f3/ctrl+; exit"
}

// handleMsgViewportLineCopyKeys returns (cmd, true) when the key was consumed (test_ignore.go parity subset).
func (m *model) handleMsgViewportLineCopyKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !m.msgViewportWanted() {
		return nil, false
	}
	s := msg.String()
	if !m.msgLineCopyMode {
		if s == "ctrl+;" || s == "f3" {
			m.enterMsgLineCopyMode()
			return nil, true
		}
		return nil, false
	}

	switch s {
	case "ctrl+;", "f3":
		m.clearMsgLineCopyMode()
		m.copyStatus = ""
		return nil, true
	case " ":
		m.clearMsgLineCopyMode()
		m.copyStatus = ""
		return nil, true
	case "esc":
		m.clearMsgLineCopyMode()
		m.copyStatus = ""
		return nil, true
	case "c":
		return m.msgViewportLineCopyYank(false), true
	case "ctrl+a":
		return m.msgViewportLineCopyYank(true), true
	case "up", "k":
		m.msgLineCopyStart--
		if m.msgLineCopyStart < 0 {
			m.msgLineCopyStart = 0
		}
		return nil, true
	case "down", "j":
		n := m.msgViewportPlainLineCount()
		if n > 0 {
			m.msgLineCopyStart++
			if m.msgLineCopyStart >= n {
				m.msgLineCopyStart = n - 1
			}
		}
		return nil, true
	default:
		return nil, false
	}
}

func (m *model) msgViewportLineCopyYank(all bool) tea.Cmd {
	lines := strings.Split(m.msgVpPlainDoc, "\n")
	if len(lines) == 0 {
		m.clearMsgLineCopyMode()
		return nil
	}
	lo, hi := 0, len(lines)-1
	if !all {
		lo, hi = m.msgLineCopyStart, m.msgLineCopyEnd
		if lo > hi {
			lo, hi = hi, lo
		}
		if lo < 0 {
			lo = 0
		}
		if hi >= len(lines) {
			hi = len(lines) - 1
		}
	}
	var b strings.Builder
	for i := lo; i <= hi; i++ {
		if i > lo {
			b.WriteByte('\n')
		}
		b.WriteString(ansi.Strip(lines[i]))
	}
	text := strings.TrimRight(b.String(), "\n")
	m.clearMsgLineCopyMode()
	if text == "" {
		return nil
	}
	return m.selectionCopyToClipboardCmd(text)
}

func (m *model) msgLineCopyHighlightRange() (lo, hi int, ok bool) {
	if !m.msgLineCopyMode {
		return 0, 0, false
	}
	lo, hi = m.msgLineCopyStart, m.msgLineCopyEnd
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

// applyMsgLineCopyRowHighlight highlights absolute document lines [lo,hi] in the visible viewport slice.
func applyMsgLineCopyRowHighlight(lines []string, yOffset, lo, hi int) {
	for i := range lines {
		abs := yOffset + i
		if abs >= lo && abs <= hi {
			lines[i] = lipgloss.NewStyle().
				Background(lipgloss.Color("202")).
				Foreground(lipgloss.Color("0")).
				Render("▶ " + lines[i])
		}
	}
}
