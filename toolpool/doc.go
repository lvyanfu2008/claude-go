// Package toolpool mirrors tool-pool assembly from TypeScript:
//   - assembleToolPool, getMergedTools — src/tools.ts
//   - mergeAndFilterTools, applyCoordinatorToolFilter, isPrActivitySubscriptionTool — src/utils/toolPool.ts
//   - isMcpTool — src/services/mcp/utils.ts
//
// Coordinator gating matches mergeAndFilterTools: FEATURE_COORDINATOR_MODE=1 and CLAUDE_CODE_COORDINATOR_MODE truthy.
// AssembleToolPool expects builtInTools equivalent to getTools(permissionContext); GetTools implements the
// non–isEnabled portions of src/tools.ts getTools over an embedded tools_api.json snapshot (goc/commands/data).
package toolpool
