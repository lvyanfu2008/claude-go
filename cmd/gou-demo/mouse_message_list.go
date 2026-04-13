package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/virtualscroll"
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

// msgPaneCell maps a mouse event to message-pane (0-based row within the vpH lines, col within body width).
func (m *model) msgPaneCell(msg tea.MouseMsg) (row, col int, ok bool) {
	if !m.mouseYInMessageListPane(msg.Y) {
		return 0, 0, false
	}
	vp := listViewportH(m)
	row = msg.Y - m.titleH
	if row < 0 || row >= vp {
		return 0, 0, false
	}
	col = msg.X
	bc := m.messageBodyColsForLayout()
	if col < 0 {
		col = 0
	}
	if col >= bc {
		col = bc - 1
	}
	return row, col, true
}

// clampScrollTopForVirtualList pins scrollTop to [0, max(0, totalContentHeight−viewportH)] when not sticky.
// After sticky-bottom (scrollTop sentinel ~1<<30), the first manual scroll leaves a huge scrollTop; without
// clamping, ComputeRange's binary search breaks and wheel/keys cannot scroll back toward the tail.
func (m *model) clampScrollTopForVirtualList() {
	if m.sticky {
		return
	}
	keys := m.scrollItemKeys()
	n := len(keys)
	if n == 0 {
		m.scrollTop = 0
		return
	}
	vpH := listViewportH(m)
	if vpH < 1 {
		return
	}
	off := virtualscroll.BuildOffsets(keys, m.heightCache, virtualscroll.DefaultEstimate)
	total := off[n]
	maxTop := total - vpH
	if maxTop < 0 {
		maxTop = 0
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
	if m.scrollTop > maxTop {
		m.scrollTop = maxTop
	}
}

// tryHandleMessageListMouse maps wheel to virtual scroll and Shift+left-drag to in-app selection (TS selection + wheel clears).
// Plain left-drag scrolls the list. Returns true if the event was consumed.
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

	ev := tea.MouseEvent(msg)

	if ev.IsWheel() {
		if !m.mouseYInMessageListPane(msg.Y) {
			return false
		}
		m.clearMsgSelection()
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
		if msg.Button == tea.MouseButtonLeft && ev.Shift {
			if pr, pc, ok := m.msgPaneCell(msg); ok {
				m.selDragging = true
				m.selHas = false
				m.selAnchorR, m.selAnchorC = pr, pc
				m.selFocusR, m.selFocusC = pr, pc
				m.msgListMouseDragging = false
				return true
			}
		}
		if msg.Button == tea.MouseButtonLeft && !ev.Shift {
			if m.mouseYInMessageListPane(msg.Y) {
				m.msgListMouseDragging = true
				m.msgListMouseLastY = msg.Y
				return true
			}
			m.msgListMouseDragging = false
		}
	case tea.MouseActionMotion:
		if m.selDragging && ev.Shift && msg.Button == tea.MouseButtonLeft {
			if pr, pc, ok := m.msgPaneCell(msg); ok {
				m.selFocusR, m.selFocusC = pr, pc
				return true
			}
		}
		if m.msgListMouseDragging && msg.Button == tea.MouseButtonLeft && !ev.Shift {
			dy := msg.Y - m.msgListMouseLastY
			if dy != 0 {
				m.sticky = false
				m.scrollTop = max(0, m.scrollTop-dy)
				m.msgListMouseLastY = msg.Y
			}
			return true
		}
	case tea.MouseActionRelease:
		if m.selDragging {
			m.selDragging = false
			m.selHas = true
			return true
		}
		if m.msgListMouseDragging {
			m.msgListMouseDragging = false
			return true
		}
	}
	return false
}
