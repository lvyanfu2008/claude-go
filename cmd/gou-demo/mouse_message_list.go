package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// mouseYInMessageListPane reports whether screen row y falls in the virtual message list
// (title row(s) above, stream strip / prompt below). Coords are 0-based from top of terminal.
func (m *model) mouseYInMessageListPane(y int) bool {
	if m.height <= 0 {
		return false
	}
	vp := listViewportH(m)
	top := m.titleH
	bot := top + vp
	return y >= top && y < bot
}

// tryHandleMessageListMouse maps wheel and left-drag to virtual scroll (TS ScrollBox wheel / drag).
// Returns true if the event was consumed (caller should not forward to prompt).
func (m *model) tryHandleMessageListMouse(msg tea.MouseMsg) bool {
	if gouDemoEnvTruthy("GOU_DEMO_DISABLE_MOUSE_SCROLL") {
		return false
	}
	if m.permAsk != nil || m.slashPick != nil {
		return false
	}
	if m.uiScreen == gouDemoScreenTranscript && (m.transcriptSearchOpen || m.transcriptDumpMode) {
		return false
	}

	if tea.MouseEvent(msg).IsWheel() {
		if !m.mouseYInMessageListPane(msg.Y) {
			return false
		}
		step := max(1, listViewportH(m)/6)
		m.sticky = false
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scrollTop = max(0, m.scrollTop-step)
		case tea.MouseButtonWheelDown:
			m.scrollTop += step
		case tea.MouseButtonWheelLeft:
			m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
		case tea.MouseButtonWheelRight:
			m.scrollTop += listViewportH(m) / 2
		default:
			return false
		}
		return true
	}

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			if m.mouseYInMessageListPane(msg.Y) {
				m.msgListMouseDragging = true
				m.msgListMouseLastY = msg.Y
				return true
			}
			m.msgListMouseDragging = false
		}
	case tea.MouseActionMotion:
		if m.msgListMouseDragging && msg.Button == tea.MouseButtonLeft {
			dy := msg.Y - m.msgListMouseLastY
			if dy != 0 {
				m.sticky = false
				m.scrollTop = max(0, m.scrollTop-dy)
				m.msgListMouseLastY = msg.Y
			}
			return true
		}
	case tea.MouseActionRelease:
		if m.msgListMouseDragging {
			m.msgListMouseDragging = false
			return true
		}
	}
	return false
}
