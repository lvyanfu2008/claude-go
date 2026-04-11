package query

import "goc/types"

// MessagesForQuery mirrors getMessagesAfterCompactBoundary in query.ts (without HISTORY_SNIP projection).
// Tool-result re-apply / budget runs next via [runApplyToolResultBudget] (default: JSON reapply; override with [QueryDeps.ApplyToolResultBudget]).
func MessagesForQuery(messages []types.Message) []types.Message {
	out := make([]types.Message, len(messages))
	copy(out, messages)
	return MessagesAfterCompactBoundary(out, CompactBoundaryOpts{})
}
