package toolexecution

import (
	"context"
	"encoding/json"

	"goc/tools/toolresultpersist"
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
		// PostToolUseHookRunner mirrors executePostToolHooks: runs after a tool completes.
		// When nil, post-tool-use hooks are skipped.
		PostToolUseHookRunner PostToolUseHookRunner
	PreToolHookPermission *PermissionDecision
	// ToolPermission optional deny/ask rules for [RuleBasedDecisionForTool] after the query gate.
	ToolPermission *types.ToolPermissionContextData
	// SandboxingEnabled and AutoAllowBashWholeToolAskWhenSandboxed together enable permissions.ts 1b:
	// whole-tool alwaysAsk on [BashToolName] is skipped when [BashInputUsesSandboxForRule1b] is true (see [WholeToolAskSkippedForBash1b]).
	SandboxingEnabled                      bool
	AutoAllowBashWholeToolAskWhenSandboxed bool

	// MainLoopModel and ReadTool* supply context for Read → tool_result.content mapping
	// (mirrors TS FileReadTool.mapToolResultToToolResultBlockParam + memory freshness roots).
	MainLoopModel  string
	ReadToolRoots  []string
	ReadToolMemCWD string

	// SchemaHintBuilder returns a deferred-tool discovery hint string for the given tool name.
	// Returned hint is appended to InputValidationError tool_results. When nil or returns "",
	// no hint is appended. Mirrors buildSchemaNotSentHint in TS.
	SchemaHintBuilder func(toolName string) string

	// MultiMessageToolHandler, when set, allows a tool to produce multiple result
	// messages (e.g., Skill tool metadata + content). Checked before InvokeTool.
	// Returns (messages, true) if handled, (nil, false) to fall through to InvokeTool.
	MultiMessageToolHandler func(ctx context.Context, name, toolUseID string, input json.RawMessage, assistantUUID string) (messages []types.Message, handled bool)

	// ToolResultPersistConfig, when non-nil, enables per-tool result persistence to disk
	// (mirrors TS toolResultStorage.ts). When a tool result exceeds its MaxResultSizeChars
	// threshold, the content is saved to {sessionDir}/tool-results/ and replaced with a
	// preview message in the tool_result block.
	ToolResultPersistConfig *ToolResultPersistConfig
}

// ToolResultPersistConfig bundles the configuration for tool result persistence.
// Mirrors the combination of SessionInfo + ProcessOptions + ContentReplacementState from TS.
type ToolResultPersistConfig struct {
	// SessionInfo identifies the session for path computation.
	SessionInfo toolresultpersist.SessionInfo

	// ProcessOptions configures per-tool persistence thresholds.
	ProcessOptions toolresultpersist.ProcessOptions

	// ContentReplacementState for per-message aggregate budget enforcement.
	// When nil, only per-tool persistence is active; when non-nil, the budget
	// enforcement path is also enabled (mirrors TS ContentReplacementState gate).
	ContentReplacementState *toolresultpersist.ContentReplacementState

	// PerMessageBudgetLimit overrides the default aggregate budget limit.
	// 0 uses the default (MaxToolResultsPerMessageChars).
	PerMessageBudgetLimit int

	// SkipToolNames is a set of tool names to never persist (e.g., "Read").
	// Maps to TS skipToolNames in enforceToolResultBudget.
	SkipToolNames map[string]bool

	// ToolMaxResultSizes maps tool name → declared MaxResultSizeChars (from ToolSpec).
	// Used by ProcessToolResultBlock to look up per-tool thresholds.
	// When nil, tools are looked up via the Registry.
	ToolMaxResultSizes map[string]int64
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
