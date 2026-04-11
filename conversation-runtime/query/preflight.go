package query

import (
	"context"
	"encoding/json"

	"goc/types"
)

// MicrocompactResult mirrors src/services/compact/microCompact.ts MicrocompactResult.
type MicrocompactResult struct {
	Messages       []types.Message
	CompactionInfo json.RawMessage `json:"compactionInfo,omitempty"`
}

// MicrocompactInput mirrors microcompactMessages(messages, toolUseContext, querySource).
type MicrocompactInput struct {
	Messages       []types.Message
	ToolUseContext *types.ToolUseContext
	QuerySource    types.QuerySource
}

// CacheSafeParams mirrors the third argument to autoCompactIfNeeded in query.ts.
type CacheSafeParams struct {
	SystemPrompt        SystemPrompt
	UserContext         map[string]string
	SystemContext       map[string]string
	ToolUseContext      *types.ToolUseContext
	ForkContextMessages []types.Message
}

// AutocompactInput mirrors autoCompactIfNeeded arguments (messages, tcx, cacheSafeParams, querySource, tracking, snipTokensFreed).
type AutocompactInput struct {
	Messages        []types.Message
	ToolUseContext  *types.ToolUseContext
	CacheSafe       CacheSafeParams
	QuerySource     types.QuerySource
	Tracking        json.RawMessage // TS AutoCompactTrackingState | undefined
	SnipTokensFreed *int
}

// AutocompactResult mirrors the Promise return of autoCompactIfNeeded.
type AutocompactResult struct {
	WasCompacted        bool
	PostMessages        []types.Message // when non-empty, replaces model-facing transcript (TS buildPostCompactMessages)
	ConsecutiveFailures int
	CompactionResult    json.RawMessage
	// UpdatedTracking when non-empty is merged into [State.AutoCompactTracking] after this autocompact call.
	UpdatedTracking json.RawMessage `json:"updatedTracking,omitempty"`
	// UpdatedContentReplacementState when non-empty replaces [types.ToolUseContext.ContentReplacementState] on [State].
	UpdatedContentReplacementState json.RawMessage `json:"updatedContentReplacementState,omitempty"`
}

func runMicrocompact(
	ctx context.Context,
	deps *QueryDeps,
	in *MicrocompactInput,
) ([]types.Message, error) {
	if deps == nil || deps.Microcompact == nil {
		return in.Messages, nil
	}
	res, err := deps.Microcompact(ctx, in)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Messages == nil || len(res.Messages) == 0 {
		return in.Messages, nil
	}
	return res.Messages, nil
}

func runAutocompact(
	ctx context.Context,
	deps *QueryDeps,
	in *AutocompactInput,
) ([]types.Message, *AutocompactResult, error) {
	if deps == nil || deps.Autocompact == nil {
		return in.Messages, nil, nil
	}
	res, err := deps.Autocompact(ctx, in)
	if err != nil {
		return nil, nil, err
	}
	if res == nil {
		return in.Messages, nil, nil
	}
	out := in.Messages
	if res.WasCompacted && res.PostMessages != nil && len(res.PostMessages) > 0 {
		out = res.PostMessages
	}
	return out, res, nil
}

func runApplyToolResultBudget(
	ctx context.Context,
	deps *QueryDeps,
	in *ToolResultBudgetInput,
) ([]types.Message, error) {
	if deps != nil && deps.ApplyToolResultBudget != nil {
		out, err := deps.ApplyToolResultBudget(ctx, in)
		if err != nil {
			return nil, err
		}
		if out == nil {
			return in.Messages, nil
		}
		return out, nil
	}
	// Default: JSON replacement map re-apply (TS mustReapply / resume parity); no disk persist.
	return ReapplyToolResultReplacementsFromState(in.Messages, in.ContentReplacementState), nil
}

func runSnipCompact(ctx context.Context, deps *QueryDeps, in *SnipCompactInput) (*SnipCompactResult, error) {
	if deps == nil || deps.SnipCompact == nil {
		return nil, nil
	}
	return deps.SnipCompact(ctx, in)
}

// applyAutocompactSideEffects merges [AutocompactResult] tracking / replacement blobs into [State].
// Mutates [State] only; does not write back to [QueryParams] (callers snapshot state separately if needed).
func applyAutocompactSideEffects(state *State, res *AutocompactResult) {
	if state == nil || res == nil {
		return
	}
	if len(res.UpdatedTracking) > 0 {
		state.AutoCompactTracking = append(json.RawMessage(nil), res.UpdatedTracking...)
	}
	if len(res.UpdatedContentReplacementState) > 0 {
		tcx := state.ToolUseContext
		tcx.ContentReplacementState = append(json.RawMessage(nil), res.UpdatedContentReplacementState...)
		state.ToolUseContext = tcx
	}
}
