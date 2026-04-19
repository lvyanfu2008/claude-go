package main

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/conversation"
	"goc/gou/layout"
	"goc/gou/messagerow"
	"goc/gou/virtualscroll"
	"goc/types"
)

// messageBodyColsForLayout returns wrap width for message rows (excludes TUI scrollbar column when active).
func (m *model) messageBodyColsForLayout() int {
	if m.msgBodyCols > 0 {
		return m.msgBodyCols
	}
	c := m.cols
	if c < 1 {
		return 40
	}
	return c
}

// messageListMouseWheelStep returns how many terminal rows one wheel notch moves (message pane).
// Smaller than viewport height / 6 so scrolling feels less jumpy.
func messageListMouseWheelStep(vpH int) int {
	if vpH < 1 {
		return 1
	}
	return max(1, vpH/12)
}

// messageScrollContentHeight returns total wrapped height (terminal rows) of the virtual message list.
func (m *model) messageScrollContentHeight() int {
	keys := m.scrollItemKeys()
	if len(keys) == 0 {
		return 0
	}
	off := virtualscroll.BuildOffsets(keys, m.heightCache, virtualscroll.DefaultEstimate)
	return off[len(keys)]
}

// messageScrollbarThumb returns [start, length) in viewport rows [0, vpH) for a proportional thumb.
func messageScrollbarThumb(vpH, totalH, scrollTop int) (start, length int) {
	if vpH < 1 {
		return 0, 0
	}
	if totalH <= vpH {
		return 0, vpH
	}
	maxTop := totalH - vpH
	st := scrollTop
	if st < 0 {
		st = 0
	}
	if st > maxTop {
		st = maxTop
	}
	length = max(1, vpH*vpH/max(1, totalH))
	if length > vpH {
		length = vpH
	}
	start = (st * (vpH - length)) / max(1, maxTop)
	if start < 0 {
		start = 0
	}
	if start+length > vpH {
		start = vpH - length
	}
	return start, length
}

// joinMessagePaneLinesWithScrollbar pads each line to bodyCols cells and appends one scrollbar column when barW==1.
func joinMessagePaneLinesWithScrollbar(lines []string, bodyCols, vpH, totalH, scrollTop int, barW int) string {
	if barW != 1 || vpH < 1 {
		return strings.Join(lines, "\n")
	}
	thumbStart, thumbLen := messageScrollbarThumb(vpH, totalH, scrollTop)
	trackStyle := lipgloss.NewStyle().Faint(true)
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	out := make([]string, 0, vpH)
	for r := 0; r < vpH; r++ {
		ln := ""
		if r < len(lines) {
			ln = lines[r]
		}
		pad := bodyCols - layout.VisualWidth(ln)
		if pad > 0 {
			ln += strings.Repeat(" ", pad)
		}
		ch := "│"
		if r >= thumbStart && r < thumbStart+thumbLen {
			out = append(out, ln+thumbStyle.Render("┃"))
		} else {
			out = append(out, ln+trackStyle.Render(ch))
		}
	}
	return strings.Join(out, "\n")
}

// fillMessageHeightCache fills heightCache for all scroll keys at the given wrap width (hl = search needle).
func (m *model) fillMessageHeightCache(cols int, hl string) {
	if m.heightCache == nil {
		m.heightCache = make(map[string]int)
	}
	m.resolvedToolIDs = messagerow.CollectResolvedToolUseIDs(m.store.Messages)
	allKeys := m.scrollItemKeys()
	virtualscroll.PruneHeightCache(m.heightCache, allKeys)
	if cols < 1 {
		cols = 40
	}
	msgView := m.messagesForScroll()
	for i := range msgView {
		k := conversation.ItemKey(msgView[i], m.store.ConversationID)
		h := m.measureMessageRows(msgView[i], cols, hl)
		if i > 0 && userAssistantPairBlankLine(msgView[i-1], msgView[i]) {
			h++
		}
		if i > 0 && transcriptAssistantPairBlankLine(m, msgView[i-1], msgView[i]) {
			h++
		}
		m.heightCache[k] = h
	}
	streamKeys := m.transcriptStreamingToolScrollKeys()
	st := m.transcriptStreamingToolsForView()
	for i, sk := range streamKeys {
		if i < len(st) {
			h := m.measureTranscriptStreamingToolRow(st[i], cols, hl)
			if i == 0 && len(msgView) > 0 && msgView[len(msgView)-1].Type == types.MessageTypeUser {
				h++
			}
			m.heightCache[sk] = h
		}
	}
}
