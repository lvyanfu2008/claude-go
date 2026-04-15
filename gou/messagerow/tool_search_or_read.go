// Mirrors claude-code/src/utils/collapseReadSearch.ts getToolSearchOrReadInfo (no TS Tools registry).
package messagerow

import (
	"os"
	"strings"
)

const replToolName = "REPL"

// searchOrReadResult matches TS SearchOrReadResult (subset used by collapse).
type searchOrReadResult struct {
	isCollapsible      bool
	isSearch           bool
	isRead             bool
	isList             bool
	isREPL             bool
	isMemoryWrite      bool
	isAbsorbedSilently bool
	mcpServerName      string
	isBash             bool
}

func snipToolNameFromEnv() string {
	return strings.TrimSpace(os.Getenv("GOU_DEMO_SNIP_TOOL_NAME"))
}

func toolSearchToolNameFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("GOU_DEMO_TOOL_SEARCH_NAME")); v != "" {
		return v
	}
	return "ToolSearch"
}

func mcpServerNameFromToolName(name string) string {
	if !strings.HasPrefix(name, "mcp__") {
		return ""
	}
	parts := strings.Split(name, "__")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

// getToolSearchOrReadInfoGo mirrors TS getToolSearchOrReadInfo for collapse eligibility and categorization.
func getToolSearchOrReadInfoGo(toolName string, toolInput map[string]any) searchOrReadResult {
	name := strings.TrimSpace(toolName)
	if name == replToolName {
		return searchOrReadResult{
			isCollapsible:      true,
			isREPL:             true,
			isAbsorbedSilently: true,
		}
	}
	if isMemoryWriteOrEditGo(name, toolInput) {
		return searchOrReadResult{isCollapsible: true, isMemoryWrite: true}
	}
	if sn := snipToolNameFromEnv(); sn != "" && name == sn {
		return searchOrReadResult{isCollapsible: true, isAbsorbedSilently: true}
	}
	if CollapseAllBashFromEnv() && name == toolSearchToolNameFromEnv() {
		return searchOrReadResult{isCollapsible: true, isAbsorbedSilently: true}
	}
	if srv := mcpServerNameFromToolName(name); srv != "" {
		return searchOrReadResult{
			isCollapsible: true,
			isSearch:      true,
			mcpServerName: srv,
		}
	}

	switch name {
	case "Read":
		return searchOrReadResult{isCollapsible: true, isRead: true}
	case "Grep":
		return searchOrReadResult{isCollapsible: true, isSearch: true}
	case "Glob":
		return searchOrReadResult{isCollapsible: true, isSearch: true}
	case "Bash", "BashZog":
		cmd := strFromMap(toolInput, "command")
		isS, isR, isL := IsSearchOrReadBashCommand(cmd)
		if isL {
			return searchOrReadResult{isCollapsible: true, isList: true}
		}
		if isS {
			return searchOrReadResult{isCollapsible: true, isSearch: true}
		}
		if isR {
			return searchOrReadResult{isCollapsible: true, isRead: true}
		}
		if CollapseAllBashFromEnv() {
			return searchOrReadResult{isCollapsible: true, isBash: true}
		}
		return searchOrReadResult{}
	default:
		return searchOrReadResult{}
	}
}

func isToolSearchOrReadGo(toolName string, toolInput map[string]any) bool {
	return getToolSearchOrReadInfoGo(toolName, toolInput).isCollapsible
}
