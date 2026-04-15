// CollapseReadSearchGroupsInList mirrors TS collapseReadSearchGroups for consecutive
// [assistant: single tool_use][user: single tool_result] runs (subset: Read/Grep/Glob/Bash rules in canRollupAssistantToolPair).
// Opt-in via GOU_DEMO_COLLAPSE_READ_SEARCH_FULL=1; display pipeline only (do not double-apply with store tail merge if both enabled).
package messagerow

import (
	"os"
	"strings"

	"goc/types"
)

// CollapseReadSearchFullFromEnv enables full-list collapse in messagesview.MessagesForScrollList.
func CollapseReadSearchFullFromEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_COLLAPSE_READ_SEARCH_FULL")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// CollapseReadSearchGroupsInList replaces each maximal run of consecutive collapsible tool pairs
// with one collapsed_read_search message. Already-collapsed rows are left unchanged.
func CollapseReadSearchGroupsInList(msgs []types.Message) []types.Message {
	if !CollapseReadSearchFullFromEnv() || len(msgs) < 2 {
		return msgs
	}
	out := make([]types.Message, 0, len(msgs))
	i := 0
	for i < len(msgs) {
		if msgs[i].Type == types.MessageTypeCollapsedReadSearch {
			out = append(out, msgs[i])
			i++
			continue
		}
		nPairs, end := countRollupPairRun(msgs, i)
		if nPairs >= 1 {
			collapsed := buildCollapsedReadSearch(msgs[i:end])
			out = append(out, collapsed)
			i = end
			continue
		}
		out = append(out, msgs[i])
		i++
	}
	return out
}

// countRollupPairRun returns the number of complete [assistant][user] pairs starting at start,
// and the exclusive end index of the run (first index after the last message consumed).
func countRollupPairRun(msgs []types.Message, start int) (pairs int, endExclusive int) {
	i := start
	for i+1 < len(msgs) {
		if msgs[i].Type != types.MessageTypeAssistant || msgs[i+1].Type != types.MessageTypeUser {
			break
		}
		tu, okA := assistantSingleToolUse(msgs[i])
		if !okA || !canRollupAssistantToolPair(tu) {
			break
		}
		tr, okU := userSingleToolResult(msgs[i+1])
		if !okU || tr.ToolUseID == "" || tu.ID == "" || tr.ToolUseID != tu.ID {
			break
		}
		pairs++
		i += 2
	}
	return pairs, i
}
