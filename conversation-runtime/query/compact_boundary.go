package query

import (
	"goc/types"
)

// CompactBoundaryOpts mirrors optional args for getMessagesAfterCompactBoundary (TS).
// When [ProjectSnippedView] is implemented, [ExcludeSnipped] can hide snipped rows like TS HISTORY_SNIP path.
type CompactBoundaryOpts struct {
	ExcludeSnipped bool // reserved; currently ignored (no snip projection in Go yet)
}

// IsCompactBoundaryMessage mirrors src/utils/messages.ts isCompactBoundaryMessage.
func IsCompactBoundaryMessage(m types.Message) bool {
	if m.Type != types.MessageTypeSystem || m.Subtype == nil {
		return false
	}
	return *m.Subtype == "compact_boundary"
}

// FindLastCompactBoundaryIndex mirrors src/utils/messages.ts findLastCompactBoundaryIndex.
func FindLastCompactBoundaryIndex(messages []types.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if IsCompactBoundaryMessage(messages[i]) {
			return i
		}
	}
	return -1
}

// MessagesAfterCompactBoundary mirrors getMessagesAfterCompactBoundary when HISTORY_SNIP
// projection is off (or [opts.ExcludeSnipped] is false): slice from last compact_boundary onward.
func MessagesAfterCompactBoundary(messages []types.Message, opts CompactBoundaryOpts) []types.Message {
	_ = opts
	idx := FindLastCompactBoundaryIndex(messages)
	if idx < 0 {
		out := make([]types.Message, len(messages))
		copy(out, messages)
		return out
	}
	out := make([]types.Message, len(messages)-idx)
	copy(out, messages[idx:])
	return out
}
