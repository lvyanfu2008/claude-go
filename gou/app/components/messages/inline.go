package messages

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/markdown"
	"goc/gou/theme"
)

// StyleMarkdownInlineSegments renders paragraph/list_item runs with TS-style
// inline `code` color and strong/emphasis (terminal lipgloss).
func StyleMarkdownInlineSegments(segs []markdown.InlineSegment, linePrefix string, userRow bool) string {
	if len(segs) == 0 {
		return ""
	}
	stCode := BaseMsgStyle(userRow).Foreground(theme.MarkdownInlineCode())
	var stPlain, stBold, stItalic, stBoldItalic lipgloss.Style
	if userRow {
		ut := theme.UserMessageText()
		stPlain = BaseMsgStyle(userRow).Foreground(ut).Bold(true)
		stBold = BaseMsgStyle(userRow).Foreground(ut).Bold(true)
		stItalic = BaseMsgStyle(userRow).Foreground(ut).Italic(true)
		stBoldItalic = BaseMsgStyle(userRow).Foreground(ut).Bold(true).Italic(true)
	} else {
		stPlain = BaseMsgStyle(userRow)
		stBold = BaseMsgStyle(userRow).Bold(true)
		stItalic = BaseMsgStyle(userRow).Italic(true)
		stBoldItalic = BaseMsgStyle(userRow).Bold(true).Italic(true)
	}
	var b strings.Builder
	for i, seg := range segs {
		txt := seg.Text
		if i == 0 && linePrefix != "" {
			txt = linePrefix + txt
		}
		if seg.Code {
			b.WriteString(stCode.Render(txt))
			continue
		}
		var st lipgloss.Style
		switch {
		case seg.Bold && seg.Italic:
			st = stBoldItalic
		case seg.Bold:
			st = stBold
		case seg.Italic:
			st = stItalic
		default:
			st = stPlain
		}
		b.WriteString(st.Render(txt))
	}
	return b.String()
}
