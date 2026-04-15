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

// CollapseReadSearchGroupsInList mirrors TS collapseReadSearchGroups (Ink parity: deferred skippable messages,
// multi–tool_result user rows, grouped_tool_use, hooks, relevant_memories, MCP, generic Bash when fullscreen env).
// Already-collapsed rows are left unchanged.
func CollapseReadSearchGroupsInList(msgs []types.Message) []types.Message {
	if !CollapseReadSearchFullFromEnv() || len(msgs) < 2 {
		return msgs
	}
	return collapseReadSearchGroupsInk(msgs)
}
