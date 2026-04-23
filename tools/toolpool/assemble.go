package toolpool

import (
	"goc/permissionrules"
	"goc/types"
)

// AssembleToolPool mirrors assembleToolPool in src/tools.ts (lines 343–365).
//
// TS calls getTools(permissionContext) internally (line 347). Go does not port getTools here — pass
// builtInTools equivalent to that output (already deny-filtered and feature-gated inside getTools).
func AssembleToolPool(
	permissionContext types.ToolPermissionContextData,
	builtInTools []types.ToolSpec,
	mcpTools []types.ToolSpec,
) []types.ToolSpec {
	// Filter out MCP tools that are in the deny list (TS line 350: filterToolsByDenyRules).
	allowedMcpTools := permissionrules.FilterToolsByDenyRules(mcpTools, permissionContext)

	// Sort each partition for prompt-cache stability, keeping built-ins as a contiguous prefix (TS lines 352–359).
	// uniqBy preserves insertion order, so built-ins win on name conflict (TS lines 356–357).
	// Avoid mutating caller slices: copy-then-sort (TS lines 358–359).
	bi := append([]types.ToolSpec(nil), builtInTools...)
	sortToolsByNameInPlace(bi)
	mc := append([]types.ToolSpec(nil), allowedMcpTools...)
	sortToolsByNameInPlace(mc)

	return UniqByName(append(bi, mc...))
}

// GetMergedTools mirrors getMergedTools in src/tools.ts (lines 381–387).
//
// TS uses getTools(permissionContext) for the first slice; pass the same as builtInTools here.
func GetMergedTools(
	_ types.ToolPermissionContextData,
	builtInTools []types.ToolSpec,
	mcpTools []types.ToolSpec,
) []types.ToolSpec {
	return append(append([]types.ToolSpec(nil), builtInTools...), mcpTools...)
}
