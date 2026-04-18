// CollapseReadSearchGroupsInList mirrors TS collapseReadSearchGroups for consecutive
// [assistant: single tool_use][user: single tool_result] runs (subset: Read/Grep/Glob/Bash rules in canRollupAssistantToolPair).
// Display pipeline only (do not double-apply with store tail merge if both enabled).
package messagerow

import (
	"goc/types"
)

// CollapseReadSearchGroupsInList mirrors TS collapseReadSearchGroups (Ink parity: deferred skippable messages,
// multi–tool_result user rows, grouped_tool_use, hooks, relevant_memories, MCP, generic Bash when fullscreen env).
// Already-collapsed rows are left unchanged.
// If resolvedToolUseIDs is non-nil, a group is collapsed only when every tool_use_id in the group appears in the map
// (so in-flight tools keep the transcript expanded until results arrive).
func CollapseReadSearchGroupsInList(msgs []types.Message, resolvedToolUseIDs map[string]struct{}) []types.Message {
	if len(msgs) < 2 {
		return msgs
	}
	return collapseReadSearchGroupsInk(msgs, resolvedToolUseIDs)
}
