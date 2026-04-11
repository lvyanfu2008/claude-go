package query

import (
	"context"
	"encoding/json"

	"goc/types"
)

// CanUseToolFn mirrors src/hooks/useCanUseTool.ts usage from query.ts (permission gate).
type CanUseToolFn func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (allowed bool, err error)

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
	// AutoCompactTracking optional seed for [State.AutoCompactTracking] (first queryLoop iteration).
	AutoCompactTracking json.RawMessage `json:"-"`
}

// CallModelInput groups arguments passed to QueryDeps.CallModel (mirrors deps.callModel({...}) in query.ts).
type CallModelInput struct {
	Messages       []types.Message
	SystemPrompt   SystemPrompt
	ThinkingConfig types.ThinkingConfig
	Tools          json.RawMessage
	SignalDone     <-chan struct{} // TS AbortSignal — caller closes to abort
	// Cwd and ModelID are used by [LocalTurnCallModel] / [localturn.Params] (optional).
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
