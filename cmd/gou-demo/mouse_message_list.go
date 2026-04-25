package main

import (
	tea "charm.land/bubbletea/v2"

	"goc/types"
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

// clampScrollTopForVirtualList pins scrollTop to [0, max(0, totalContentHeight−viewportH)] when not sticky.
// After sticky-bottom (scrollTop sentinel ~1<<30), the first manual scroll leaves a huge scrollTop; without
// clamping, the new renderer's ComputeVisibleRange cannot scroll back toward the tail.
func (m *model) clampScrollTopForVirtualList() {
	if m.sticky {
		return
	}
	vpH := listViewportH(m)
	if vpH < 1 {
		return
	}
	m.integrateMessageRenderer()
	messages := m.store.Messages
	var messagesPtr []*types.Message
	for i := range messages {
		messagesPtr = append(messagesPtr, &messages[i])
	}
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	width := m.messageBodyColsForLayout()

	totalHeight := m.msgRenderer.ComputeTotalHeight(messagesPtr, isTranscript, verbose, width)
	maxTop := totalHeight - vpH
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

// tryHandleMessageListMouse maps wheel to virtual scroll; plain left-drag scrolls the list.
// go-tui/main/test.go: when bubbles viewport is at top, wheel-up in the pane can disable SGR mouse so the
// terminal scrollback wheel works (GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE).
// Returns whether the event was consumed and an optional tea.Cmd (e.g. Println hint; mouse mode is declarative via View).
func (m *model) tryHandleMessageListMouse(msg tea.Msg) (bool, tea.Cmd) {
	if gouDemoEnvTruthy("GOU_DEMO_DISABLE_MOUSE_SCROLL") {
		return false, nil
	}
	if m.permAsk != nil || m.slashListVisible() {
		return false, nil
	}
	if m.uiScreen == gouDemoScreenTranscript && (m.transcriptSearchOpen || m.transcriptDumpMode) {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		if !m.mouseYInMessageListPane(msg.Y) {
			return false, nil
		}
		if m.msgViewportWanted() && gouDemoMsgHistoryBrowseReleaseEnabled() && gouDemoMouseCellMotionEnabled() &&
			msg.Button == tea.MouseWheelUp && !msg.Mod.Contains(tea.ModShift) && m.msgViewport.AtTop() {
			m.msgHistoryBrowseMouseOff = true
			return true, tea.Println("\n📜 History browse: mouse wheel uses host buffer; press any key to return…")
		}
		if m.msgViewportWanted() {
			//diaglog.Line("[mouse] tryHandleMessageListMouse: using viewport, button=%v, viewport height=%d", msg.Button, m.msgViewport.Height())
			switch msg.Button {
			case tea.MouseWheelUp:
				m.handleMsgViewportMouseWheel(1)
			case tea.MouseWheelDown:
				m.handleMsgViewportMouseWheel(-1)
			case tea.MouseWheelLeft:
				for range max(1, listViewportH(m)/24) {
					m.msgViewport.HalfPageUp()
				}
			case tea.MouseWheelRight:
				for range max(1, listViewportH(m)/24) {
					m.msgViewport.HalfPageDown()
				}
			default:
				return false, nil
			}
			//diaglog.Line("[mouse] tryHandleMessageListMouse: viewport scrolled, yOffset=%d, totalLines=%d", m.msgViewport.YOffset(), m.msgViewport.TotalLineCount())
			return true, nil
		}
		step := messageListMouseWheelStep(listViewportH(m))
		switch msg.Button {
		case tea.MouseWheelUp:
			m.sticky = false
			m.scrollTop = max(0, m.scrollTop-step)
		case tea.MouseWheelDown:
			m.scrollTop += step
		case tea.MouseWheelLeft:
			m.sticky = false
			m.scrollTop = max(0, m.scrollTop-listViewportH(m)/4)
		case tea.MouseWheelRight:
			m.scrollTop += listViewportH(m) / 4
		default:
			return false, nil
		}
		// Clamp scrollTop if it's too large (e.g., from sticky-bottom sentinel 1<<30)
		if !m.sticky && m.scrollTop >= 1<<20 {
			m.clampScrollTopForVirtualList()
		}
		return true, nil

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && !msg.Mod.Contains(tea.ModShift) {
			if m.mouseYInMessageListPane(msg.Y) {
				m.msgListMouseDragging = true
				m.msgListMouseLastY = msg.Y
				return true, nil
			}
			m.msgListMouseDragging = false
		}
		return false, nil

	case tea.MouseMotionMsg:
		if m.msgListMouseDragging && msg.Button == tea.MouseLeft && !msg.Mod.Contains(tea.ModShift) {
			dy := msg.Y - m.msgListMouseLastY
			if dy != 0 {
				if m.msgViewportWanted() {
					n := min(4, max(1, absInt(dy)))
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
		return false, nil

	case tea.MouseReleaseMsg:
		if m.msgListMouseDragging {
			m.msgListMouseDragging = false
			return true, nil
		}
		return false, nil
	}

	return false, nil
}
