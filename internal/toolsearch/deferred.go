package toolsearch

import "strings"

const (
	// ToolSearchToolName matches src/tools/ToolSearchTool/constants.ts
	ToolSearchToolName = "ToolSearch"
)

// deferredBuiltin matches TS tools with shouldDefer: true (built-ins; MCP uses name prefix).
var deferredBuiltin = map[string]struct{}{
	"CronCreate":           {},
	"CronDelete":           {},
	"CronList":             {},
	"EnterPlanMode":        {},
	"EnterWorktree":        {},
	"ExitPlanMode":         {},
	"ExitWorktree":         {},
	"NotebookEdit":         {},
	"TodoWrite":            {},
	"WebFetch":             {},
	"WebSearch":            {},
	"TaskOutput":           {},
	"TaskStop":             {},
	"SendMessage":          {},
	"TeamCreate":           {},
	"TeamDelete":           {},
	"TaskCreate":           {},
	"TaskGet":              {},
	"TaskList":             {},
	"TaskUpdate":           {},
	"RemoteTrigger":        {},
	"ListMcpResourcesTool": {},
	"ReadMcpResourceTool":  {},
	"LSP":                  {},
	"Config":               {},
	"AskUserQuestion":      {},
}

// IsDeferredToolName mirrors isDeferredTool (src/tools/ToolSearchTool/prompt.ts) for wire-only name + MCP prefix.
func IsDeferredToolName(name string) bool {
	if name == ToolSearchToolName {
		return false
	}
	if strings.HasPrefix(name, "mcp__") {
		return true
	}
	_, ok := deferredBuiltin[name]
	return ok
}
