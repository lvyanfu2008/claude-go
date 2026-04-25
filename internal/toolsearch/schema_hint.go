package toolsearch

import (
	"fmt"

	"goc/internal/anthropic"
	"goc/tstenv"
)

// BuildSchemaNotSentHint mirrors buildSchemaNotSentHint in src/services/tools/toolExecution.ts.
// It returns a hint string appended to input validation errors when a deferred tool's schema
// was never sent to the model because the tool hadn't been discovered yet.
//
// Returns empty string when the hint is not applicable (tool search disabled, tool not deferred,
// or tool was already discovered in messages).
func BuildSchemaNotSentHint(toolName string, messages []anthropic.Message, tools []anthropic.ToolDefinition) string {
	if !tstenv.ToolSearchEnabledOptimistic() {
		return ""
	}
	if !isToolSearchToolAvailable(tools) {
		return ""
	}
	if !IsDeferredToolName(toolName) {
		return ""
	}
	discovered := ExtractDiscoveredToolNames(messages)
	if _, ok := discovered[toolName]; ok {
		return ""
	}

	return fmt.Sprintf(
		"\n\nTool %q is deferred-loading and needs to be discovered before use.\n"+
			"When using OpenAI-compatible models (DeepSeek, Ollama, etc.), follow these steps:\n"+
			"1. First discover the tool with ToolSearch: %s(\"select:%s\")\n"+
			"2. Then call %s tool\n"+
			"\nExample:\n"+
			"%s(\"select:%s\") \u2192 %s({ ... })\n"+
			"\nImportant notes:\n"+
			"\u2022 Use camelCase parameter names (e.g., taskId), not snake_case (task_id)\n"+
			"\u2022 All task tools (TaskGet, TaskCreate, TaskUpdate, TaskList) need to be discovered first\n"+
			"\u2022 You can discover them all at once: %s(\"select:TaskGet,TaskCreate,TaskUpdate,TaskList\")\n",
		toolName,
		ToolSearchToolName, toolName,
		toolName,
		ToolSearchToolName, toolName, toolName,
		ToolSearchToolName,
	)
}

// HasSchemaNotSentHint checks whether BuildSchemaNotSentHint would return a non-empty string
// without constructing the full hint text. Useful as a fast check.
func HasSchemaNotSentHint(toolName string, messages []anthropic.Message, tools []anthropic.ToolDefinition) bool {
	if !tstenv.ToolSearchEnabledOptimistic() {
		return false
	}
	if !isToolSearchToolAvailable(tools) {
		return false
	}
	if !IsDeferredToolName(toolName) {
		return false
	}
	discovered := ExtractDiscoveredToolNames(messages)
	if _, ok := discovered[toolName]; ok {
		return false
	}
	return true
}

// SchemaNotSentHintForOpenAI returns a compact hint tailored for OpenAI-compatible models.
// It is a shorter variant of [BuildSchemaNotSentHint] for contexts where the long-form
// hint may consume too many tokens.
func SchemaNotSentHintForOpenAI(toolName string) string {
	return fmt.Sprintf(
		"\n\nTool %q is deferred-loading. Use %s(\"select:%s\") to discover it first, then call it.",
		toolName, ToolSearchToolName, toolName,
	)
}

// HasDeferredToolName checks if a name belongs to a known deferred tool (for pre-check).
func HasDeferredToolName(name string) bool {
	return IsDeferredToolName(name)
}
