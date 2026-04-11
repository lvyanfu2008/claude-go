package streamingtool

import (
	"encoding/json"

	"goc/types"
)

// TS: ToolUseBlock from @anthropic-ai/sdk
type ToolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolStatus mirrors TrackedTool.status in StreamingToolExecutor.ts.
type ToolStatus string

const (
	ToolQueued    ToolStatus = "queued"
	ToolExecuting ToolStatus = "executing"
	ToolCompleted ToolStatus = "completed"
	ToolYielded   ToolStatus = "yielded"
)

// MessageUpdate mirrors type MessageUpdate in StreamingToolExecutor.ts.
type MessageUpdate struct {
	Message    *types.Message
	NewContext ToolUseContextPort
}

// ToolRunUpdate mirrors each yield from runToolUse (message + optional contextModifier).
type ToolRunUpdate struct {
	Message         *types.Message
	ContextModifier func(ToolUseContextPort) ToolUseContextPort
}

// Abort reasons (TS string literals on signal.reason).
const (
	AbortReasonInterrupt    = "interrupt"
	AbortReasonSiblingError = "sibling_error"
)

// BashToolName mirrors BASH_TOOL_NAME in src/tools/BashTool/toolName.js.
const BashToolName = "Bash"
