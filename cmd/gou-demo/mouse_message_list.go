package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/virtualscroll"
)

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

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
// Plain left-drag scrolls the list. go-tui/main/test.go: when bubbles viewport is at top, wheel-up in the pane can
// disable SGR mouse so the terminal scrollback wheel works (GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE).
// Returns whether the event was consumed and an optional tea.Cmd (e.g. DisableMouse sequence).
func (m *model) tryHandleMessageListMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	if gouDemoEnvTruthy("GOU_DEMO_DISABLE_MOUSE_SCROLL") {
		return false, nil
	}
	if m.permAsk != nil || m.slashPick != nil {
		return false, nil
	}
	if m.uiScreen == gouDemoScreenTranscript && (m.transcriptSearchOpen || m.transcriptDumpMode) {
		return false, nil
	}

	ev := tea.MouseEvent(msg)

	if ev.IsWheel() {
		if !m.mouseYInMessageListPane(msg.Y) {
			return false, nil
		}
		if m.msgViewportWanted() && gouDemoMsgHistoryBrowseReleaseEnabled() && gouDemoMouseCellMotionEnabled() &&
			msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp && !ev.Shift && m.msgViewport.AtTop() {
			m.clearMsgSelection()
			m.msgHistoryBrowseMouseOff = true
			return true, tea.Sequence(
				tea.DisableMouse,
				tea.Println("\n📜 History browse: mouse wheel uses host buffer; you can drag-select to copy like go-tui test.go; press any key to return…"),
			)
		}
		m.clearMsgSelection()
		if m.msgViewportWanted() {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.handleMsgViewportMouseWheel(1)
			case tea.MouseButtonWheelDown:
				m.handleMsgViewportMouseWheel(-1)
			case tea.MouseButtonWheelLeft:
				for range max(1, listViewportH(m)/12) {
					m.msgViewport.HalfPageUp()
				}
			case tea.MouseButtonWheelRight:
				for range max(1, listViewportH(m)/12) {
					m.msgViewport.HalfPageDown()
				}
			default:
				return false, nil
			}
			return true, nil
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
			return false, nil
		}
		return true, nil
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
				return true, nil
			}
		}
		if msg.Button == tea.MouseButtonLeft && !ev.Shift {
			if m.mouseYInMessageListPane(msg.Y) {
				m.msgListMouseDragging = true
				m.msgListMouseLastY = msg.Y
				return true, nil
			}
			m.msgListMouseDragging = false
		}
	case tea.MouseActionMotion:
		if m.selDragging && ev.Shift && msg.Button == tea.MouseButtonLeft {
			if pr, pc, ok := m.msgPaneCell(msg); ok {
				m.selFocusR, m.selFocusC = pr, pc
				return true, nil
			}
		}
		if m.msgListMouseDragging && msg.Button == tea.MouseButtonLeft && !ev.Shift {
			dy := msg.Y - m.msgListMouseLastY
			if dy != 0 {
				if m.msgViewportWanted() {
					n := min(8, max(1, absInt(dy)))
					if dy > 0 {
						m.msgViewport.ScrollUp(n)
					} else {
						m.msgViewport.ScrollDown(n)
					}
					if !m.msgViewport.AtBottom() {
						m.sticky = false
					}
				} else {
					m.sticky = false
					m.scrollTop = max(0, m.scrollTop-dy)
				}
				m.msgListMouseLastY = msg.Y
			}
			return true, nil
		}
	case tea.MouseActionRelease:
		if m.selDragging {
			m.selDragging = false
			m.selHas = true
			return true, nil
		}
		if m.msgListMouseDragging {
			m.msgListMouseDragging = false
			return true, nil
		}
	}
	return false, nil
}
