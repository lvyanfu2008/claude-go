package query

import (
	"context"
	"encoding/json"
	"net/http"

	"goc/anthropicmessages"
	"goc/tools/toolexecution"
	"goc/types"
)

// CanUseToolFn mirrors src/hooks/useCanUseTool.ts usage from query.ts (PermissionDecision gate).
type CanUseToolFn = toolexecution.QueryCanUseToolFn

// TaskBudget mirrors query.ts QueryParams.taskBudget (API task_budget).
type TaskBudget struct {
	Total int
}

// QueryParams mirrors src/conversation-runtime/query.ts QueryParams.
type QueryParams struct {
	Messages                []types.Message
	SystemPrompt            SystemPrompt
	UserContext             map[string]string
	SystemContext           map[string]string
	CanUseTool              CanUseToolFn
	ToolUseContext          types.ToolUseContext
	FallbackModel           string
	QuerySource             types.QuerySource
	MaxOutputTokensOverride *int
	MaxTurns                *int
	SkipCacheWrite          *bool
	TaskBudget              *TaskBudget
	Deps                    *QueryDeps
	// StreamingParity, together with [StreamingParityPathEnabled] on [BuildQueryConfig], selects HTTP streaming
	// + streamingtool instead of [QueryDeps.CallModel]: Anthropic [runStreamingParityModelLoop] or OpenAI
	// [runOpenAIStreamingParityModelLoop] when [StreamingUsesOpenAIChat] is true. When both this and CallModel
	// are configured, the streaming path wins inside [queryLoop].
	StreamingParity bool
	// AutoCompactTracking optional seed for [State.AutoCompactTracking] (first queryLoop iteration).
	AutoCompactTracking json.RawMessage `json:"-"`
	// ToolPermissionContext optional merged deny/ask rules for [toolexecution.ExecutionDeps.ToolPermission] on streaming parity.
	ToolPermissionContext *types.ToolPermissionContextData `json:"-"`
}

// CallModelInput groups arguments passed to QueryDeps.CallModel (mirrors deps.callModel({...}) in query.ts).
type CallModelInput struct {
	Messages       []types.Message
	SystemPrompt   SystemPrompt
	ThinkingConfig types.ThinkingConfig
	Tools          json.RawMessage
	SignalDone     <-chan struct{} // TS AbortSignal — caller closes to abort
	// Cwd and ModelID are optional hints for hosts that implement [QueryDeps.CallModel].
	Cwd     string
	ModelID string
	// Options bag grows with parity; keep JSON for uncommon fields during migration.
	Options json.RawMessage
}

// QueryDeps mirrors src/conversation-runtime/queryPipeline/deps.ts QueryDeps.
type QueryDeps struct {
	// ApplyToolResultBudget overrides default JSON re-apply in [runApplyToolResultBudget] (TS live budget + persist).
	ApplyToolResultBudget func(ctx context.Context, in *ToolResultBudgetInput) ([]types.Message, error)
	// SnipCompact mirrors snipCompactIfNeeded yield path; nil = skip. May return [SnipCompactResult.BoundaryMessage].
	SnipCompact  func(ctx context.Context, in *SnipCompactInput) (*SnipCompactResult, error)
	CallModel    func(ctx context.Context, in *CallModelInput, emit func(QueryYield) bool) error
	Microcompact func(ctx context.Context, in *MicrocompactInput) (*MicrocompactResult, error)
	Autocompact  func(ctx context.Context, in *AutocompactInput) (*AutocompactResult, error)
	NewUUID      func() string
	// HTTPClient optional HTTP client for streaming parity (defaults to http.DefaultClient).
	HTTPClient *http.Client
	// StreamPost when set overrides [anthropicmessages.PostStream] (tests inject httptest).
	StreamPost func(ctx context.Context, p anthropicmessages.PostStreamParams) error
	// OpenAIPostStream when set overrides [PostOpenAIChatStream] for OpenAI-compatible streaming parity tests.
	OpenAIPostStream func(ctx context.Context, p OpenAIPostStreamParams) error
	// ToolexecutionDeps is passed to [RunToolUseToolRunner] during streaming parity (InvokeTool optional).
	ToolexecutionDeps toolexecution.ExecutionDeps
	// OnQueryYield optional; invoked after each successful streaming-parity yield (assistant and tool_result rows) so hosts can persist incrementally (e.g. [sessiontranscript.RecordTranscript]).
	OnQueryYield func(ctx context.Context, y QueryYield) error
	// OnStreamingToolUses optional; invoked during Anthropic/OpenAI streaming parity with the
	// current in-flight tool_use snapshot after each content_block_start|delta|stop. uses==nil
	// means message_stop (TS clears streamingToolUses). Non-nil slice replaces the live list (may be empty).
	OnStreamingToolUses func(ctx context.Context, uses []StreamingToolUseLive) error
}

// ToolResultBudgetInput is passed to [QueryDeps.ApplyToolResultBudget] (query.ts applyToolResultBudget).
type ToolResultBudgetInput struct {
	Messages                []types.Message
	ContentReplacementState json.RawMessage
	QuerySource             types.QuerySource
	AgentID                 string
	Tools                   json.RawMessage // optional; tool defs for maxResultSizeChars skip set
}

// SnipCompactInput mirrors snip input (messages only in Go stub; TS has more closure state).
type SnipCompactInput struct {
	Messages []types.Message
}

// SnipCompactResult carries snipped transcript + optional boundary row + token delta for autocompact.
type SnipCompactResult struct {
	Messages        []types.Message
	TokensFreed     int
	BoundaryMessage *types.Message
}
