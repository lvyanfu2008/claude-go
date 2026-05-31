package app

import (
	"goc/gou/conversation"
	"goc/gou/messagerow"
	"goc/gou/virtualscroll"
	"goc/types"
)

// messageBodyColsForLayout returns wrap width for message rows (excludes TUI scrollbar column when active).
func (m *Model) messageBodyColsForLayout() int {
	if m.msgBodyCols > 0 {
		return m.msgBodyCols
	}
	c := m.cols
	if c < 1 {
		return 40
	}
	return c
}

// messageScrollContentHeight returns total wrapped height (terminal rows) of the virtual message list.
func (m *Model) messageScrollContentHeight() int {
	keys := m.scrollItemKeys()
	if len(keys) == 0 {
		return 0
	}
	off := virtualscroll.BuildOffsets(keys, m.heightCache, virtualscroll.DefaultEstimate)
	return off[len(keys)]
}

// fillMessageHeightCache fills heightCache for all scroll keys at the given wrap width (hl = search needle).
func (m *Model) fillMessageHeightCache(cols int, hl string) {
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
