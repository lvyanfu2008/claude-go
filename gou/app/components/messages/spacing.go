package messages

import (
	"strings"

	"goc/gou/messagerow"
	"goc/types"
)

// UserAssistantPairBlankLine is true when the UI inserts one empty line between adjacent
// user and assistant scroll rows (either order).
func UserAssistantPairBlankLine(a, b types.Message) bool {
	u, aType := types.MessageTypeUser, types.MessageTypeAssistant
	c := types.MessageTypeCollapsedReadSearch
	return a.Type == u && b.Type == aType || a.Type == c && b.Type == aType
}

// StreamGapAfterUserMessage is true when the StreamingText tail should be separated from the
// message list by the same blank line as user↔assistant rows (last scroll message is user).
func StreamGapAfterUserMessage(msgView []types.Message) bool {
	return len(msgView) > 0 && msgView[len(msgView)-1].Type == types.MessageTypeUser
}

// TranscriptAssistantPairBlankLine is true when the UI inserts one empty line between consecutive
// assistant rows in transcript (breathing room before the next ⏺ block).
func TranscriptAssistantPairBlankLine(isTranscript bool, a, b types.Message) bool {
	if !isTranscript {
		return false
	}
	return a.Type == types.MessageTypeAssistant && b.Type == types.MessageTypeAssistant
}

// SegmentJoinSeparator inserts an extra blank line after assistant prose before a merged
// Grep/Glob/Read summary line.
func SegmentJoinSeparator(prev, cur messagerow.Segment) string {
	if prev.Kind == messagerow.SegTextMarkdown && strings.TrimSpace(prev.Text) != "" && cur.Kind == messagerow.SegToolUseSummaryLine {
		return "\n\n"
	}
	return "\n"
}
