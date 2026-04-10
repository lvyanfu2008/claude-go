package toolpool

import (
	"os"
	"strings"

	"goc/permissionrules"
	"goc/types"
)

// Special tool names excluded from the model-facing list (src/tools.ts getTools, lines 299–303).
const (
	ListMcpResourcesToolName = "ListMcpResourcesTool"
	ReadMcpResourceToolName  = "ReadMcpResourceTool"
	SyntheticOutputToolName  = "StructuredOutput"
)

// REPL-only tool names hidden when REPL mode is on (src/tools/REPLTool/constants.ts REPL_ONLY_TOOLS).
var replOnlyToolNames = map[string]struct{}{
	"Read": {}, "Write": {}, "Edit": {}, "Glob": {}, "Grep": {},
	"Bash": {}, "NotebookEdit": {}, "Agent": {},
}

// IsReplModeEnabled mirrors isReplModeEnabled in src/tools/REPLTool/constants.ts (lines 23–29).
func IsReplModeEnabled() bool {
	repl := os.Getenv("CLAUDE_CODE_REPL")
	if isEnvDefinedFalsyString(repl) {
		return false
	}
	if envTruthy(os.Getenv("CLAUDE_REPL_MODE")) {
		return true
	}
	return strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" &&
		strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT")) == "cli"
}

func isEnvDefinedFalsyString(envVar string) bool {
	if envVar == "" {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(envVar))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

func envTruthy(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// IsEnvTruthyClaudeCodeSimple mirrors isEnvTruthy(process.env.CLAUDE_CODE_SIMPLE) (src/tools.ts getTools).
func IsEnvTruthyClaudeCodeSimple() bool {
	return envTruthy(os.Getenv("CLAUDE_CODE_SIMPLE"))
}

var specialToolNamesForGetTools = map[string]struct{}{
	ListMcpResourcesToolName: {},
	ReadMcpResourceToolName:  {},
	SyntheticOutputToolName:  {},
}

// GetTools mirrors getTools in src/tools.ts (lines 269–324) for logic that does not require live
// Tool.isEnabled() or dynamic getAllBaseTools. Pass exportedBase from ParseToolsAPIDocumentJSON
// (same snapshot as getTools + toolToAPISchema export for a given export-time environment).
//
// TS order: simple branch → remove special → filterToolsByDenyRules → REPL hide → isEnabled().
// isEnabled is omitted here; re-export tools JSON if you need a different enabled set.
func GetTools(permissionContext types.ToolPermissionContextData, exportedBase []types.ToolSpec) []types.ToolSpec {
	if IsEnvTruthyClaudeCodeSimple() {
		return getToolsSimpleMode(permissionContext, exportedBase)
	}
	out := make([]types.ToolSpec, 0, len(exportedBase))
	for _, t := range exportedBase {
		if _, skip := specialToolNamesForGetTools[t.Name]; skip {
			continue
		}
		out = append(out, t)
	}
	out = permissionrules.FilterToolsByDenyRules(out, permissionContext)
	if IsReplModeEnabled() {
		replPresent := false
		for _, t := range out {
			if t.Name == "REPL" {
				replPresent = true
				break
			}
		}
		if replPresent {
			filtered := make([]types.ToolSpec, 0, len(out))
			for _, t := range out {
				if _, hide := replOnlyToolNames[t.Name]; hide {
					continue
				}
				filtered = append(filtered, t)
			}
			out = filtered
		}
	}
	return out
}

func getToolsSimpleMode(permissionContext types.ToolPermissionContextData, exportedBase []types.ToolSpec) []types.ToolSpec {
	if IsReplModeEnabled() {
		for i := range exportedBase {
			if exportedBase[i].Name == "REPL" {
				repl := exportedBase[i]
				replSimple := []types.ToolSpec{repl}
				if CoordinatorMergeFilterActive() {
					replSimple = appendToolsIfPresent(replSimple, exportedBase, []string{"TaskStop", "SendMessage"})
				}
				return permissionrules.FilterToolsByDenyRules(replSimple, permissionContext)
			}
		}
	}
	simple := pickToolsByName(exportedBase, []string{"Bash", "Read", "Edit"})
	if CoordinatorMergeFilterActive() {
		simple = appendToolsIfPresent(simple, exportedBase, []string{"Agent", "TaskStop", "SendMessage"})
	}
	return permissionrules.FilterToolsByDenyRules(simple, permissionContext)
}

func pickToolsByName(all []types.ToolSpec, names []string) []types.ToolSpec {
	out := make([]types.ToolSpec, 0, len(names))
	for _, want := range names {
		for _, t := range all {
			if t.Name == want {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

func appendToolsIfPresent(base []types.ToolSpec, all []types.ToolSpec, names []string) []types.ToolSpec {
	for _, want := range names {
		for _, t := range all {
			if t.Name == want {
				base = append(base, t)
				break
			}
		}
	}
	return base
}
