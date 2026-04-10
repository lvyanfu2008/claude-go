// Package permissionrules mirrors whole-tool permission rule parsing and deny matching from TypeScript:
//   - permissionRuleValueFromString, normalizeLegacyToolName, unescapeRuleContent — src/utils/permissions/permissionRuleParser.ts
//   - getMcpPrefix, buildMcpToolName, mcpInfoFromString, getToolNameForPermissionCheck — src/services/mcp/mcpStringUtils.ts
//   - normalizeNameForMCP — src/services/mcp/normalization.ts
//   - getDenyRules, getAskRules, toolMatchesRule, getDenyRuleForTool, getAskRuleForTool — src/utils/permissions/permissions.ts
//   - filterToolsByDenyRules — src/tools.ts (via FilterToolsByDenyRules)
//
// Go export names use PascalCase; behavior is intended to match the cited TS at each step.
package permissionrules
