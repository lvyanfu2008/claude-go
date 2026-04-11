package toolexecution

import (
	"context"
	"encoding/json"

	"goc/types"
)

type ctxKey struct{}

// ExecutionDeps carries injectable behavior for hooks and [RunToolUseChan] (mirrors globals in toolExecution.ts).
type ExecutionDeps struct {
	RandomUUID func() string
	// QueryCanUseTool mirrors query.ts CanUseToolFn (PermissionDecision); nil skips the gate.
	QueryCanUseTool QueryCanUseToolFn
	// AskResolver when [QueryCanUseTool] returns [PermissionAsk]; nil uses [ResolveAskWithDeps] headless deny.
	AskResolver func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (PermissionDecision, error)
	// Registry optional JSON-derived tools ([NewJSONToolRegistry]); used when [InvokeTool] is nil or tool not handled by InvokeTool.
	Registry ToolRegistry
	// InvokeTool runs a host-registered tool; when non-nil it takes precedence over [Registry] for execution (see [InvokeToolFunc]).
	InvokeTool InvokeToolFunc
	// PreToolUseHook mirrors executePreToolHooks deny path: non-nil return blocks the tool with a synthetic tool_result.
	PreToolUseHook func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error
	// PreToolHookPermission optional hook-phase decision (toolHooks.ts hookPermissionResult) before resolve.
	PreToolHookPermission *PermissionDecision
	// ToolPermission optional deny/ask rules for [RuleBasedDecisionForTool] after the query gate.
	ToolPermission *types.ToolPermissionContextData
}

// WithExecutionDeps attaches deps for [DepsFromContext] (used by check_permissions path).
func WithExecutionDeps(ctx context.Context, d ExecutionDeps) context.Context {
	return context.WithValue(ctx, ctxKey{}, d)
}

// DepsFromContext returns deps from ctx or zero values.
func DepsFromContext(ctx context.Context) ExecutionDeps {
	if v := ctx.Value(ctxKey{}); v != nil {
		return v.(ExecutionDeps)
	}
	return ExecutionDeps{}
}
