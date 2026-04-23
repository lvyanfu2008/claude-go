package toolpool

import (
	"strings"

	"goc/types"
)

// IsPrActivitySubscriptionTool mirrors isPrActivitySubscriptionTool in src/utils/toolPool.ts (lines 16–18).
func IsPrActivitySubscriptionTool(name string) bool {
	for _, suffix := range prActivityToolSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// ApplyCoordinatorToolFilter mirrors applyCoordinatorToolFilter in src/utils/toolPool.ts (lines 35–41).
func ApplyCoordinatorToolFilter(tools []types.ToolSpec) []types.ToolSpec {
	if len(tools) == 0 {
		return nil
	}
	out := make([]types.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if _, ok := coordinatorModeAllowedTools[t.Name]; ok {
			out = append(out, t)
			continue
		}
		if IsPrActivitySubscriptionTool(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// MergeAndFilterTools mirrors mergeAndFilterTools in src/utils/toolPool.ts (lines 55–79).
//
// mode is unused in TS (kept on the signature for useMergedTools); we accept it for API parity.
func MergeAndFilterTools(
	initialTools []types.ToolSpec,
	assembled []types.ToolSpec,
	mode types.PermissionMode,
) []types.ToolSpec {
	_ = mode // parity with src/utils/toolPool.ts: mode not read in mergeAndFilterTools

	// Merge initialTools on top — they take precedence in deduplication (TS lines 60–62).
	merged := append(append([]types.ToolSpec(nil), initialTools...), assembled...)
	merged = UniqByName(merged)

	// partition(merged, isMcpTool) → [mcp, builtIn] in lodash; then tools = builtIn.sort + mcp.sort (TS 65–70).
	var mcpPart, builtInPart []types.ToolSpec
	for _, t := range merged {
		if IsMcpTool(t) {
			mcpPart = append(mcpPart, t)
		} else {
			builtInPart = append(builtInPart, t)
		}
	}
	sortToolsByNameInPlace(builtInPart)
	sortToolsByNameInPlace(mcpPart)
	tools := append(append([]types.ToolSpec(nil), builtInPart...), mcpPart...)

	if CoordinatorMergeFilterActive() {
		return ApplyCoordinatorToolFilter(tools)
	}
	return tools
}
