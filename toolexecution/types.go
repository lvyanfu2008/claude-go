package toolexecution

import (
	"context"
	"encoding/json"

	"goc/types"
)

// ToolUseBlock mirrors Anthropic ToolUseBlock / toolExecution.ts parameter shape.
type ToolUseBlock struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// AssistantMeta carries assistant message identifiers used by toolExecution.ts (uuid, message.id, requestId).
type AssistantMeta struct {
	UUID      string
	MessageID string
	RequestID string
}

// ToolUseContext is the Go stand-in for serializable + abort slices of src/Tool.ts ToolUseContext.
// TODO(toolExecution.ts): expand with messages, toolDecisions, queryTracking, mcpClients, options, …
type ToolUseContext struct {
	Abort *AbortController
	// RequireCanUseTool mirrors toolUseContext.requireCanUseTool (toolHooks.ts resolveHookPermissionDecision).
	RequireCanUseTool bool
	// ToolPermission optional merged settings (alwaysDeny / alwaysAsk); see [CheckRuleBasedPermissions].
	ToolPermission *types.ToolPermissionContextData
	// BashSandboxRule1b optional; when set with both flags true, whole-tool alwaysAsk on Bash may be skipped (permissions.ts 1b). Usually copied from [ExecutionDeps].
	BashSandboxRule1b *BashSandboxRule1b
}

// MessageUpdate mirrors MessageUpdateLazy from toolExecution.ts (message + optional context modifier).
type MessageUpdate struct {
	Message          *types.Message
	ModifyToolUseCtx func(*ToolUseContext) *ToolUseContext
}

// CanUseToolFn mirrors CanUseToolFn from src/hooks/useCanUseTool.ts (nil return = allow).
type CanUseToolFn func(toolName string, input json.RawMessage, tcx *ToolUseContext) error

// Tool is the executable tool surface used after findToolByName (src/Tool.ts + tool.call).
// Optional: [RuleBasedToolPermissionsChecker] for permissions.ts 1c–1g in [CheckRuleBasedPermissions].
// TODO(toolExecution.ts): inputSchema, isMcp, MCP branches, …
type Tool interface {
	Name() string
	// Aliases returns deprecated names that still resolve to this tool (toolExecution.ts L351–356).
	Aliases() []string
	Call(
		ctx context.Context,
		toolUseID string,
		input json.RawMessage,
		tcx *ToolUseContext,
		canUseTool CanUseToolFn,
		assistant AssistantMeta,
		onProgress func(toolUseID string, data json.RawMessage),
	) (*types.ToolRunResult, error)
}

// ToolRegistry mirrors findToolByName over toolUseContext.options.tools.
type ToolRegistry interface {
	FindToolByName(name string) (Tool, bool)
}

// ToolProgress is a minimal progress payload placeholder (toolExecution.ts ToolProgress).
type ToolProgress struct {
	ToolUseID string
	Data      json.RawMessage
}
