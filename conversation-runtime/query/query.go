package query

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"
)

// CommandLifecycleNotifier mirrors notifyCommandLifecycle('completed', uuid) after queryLoop (query.ts tail).
// Default is a no-op; tests or hosts may replace it.
var CommandLifecycleNotifier = func(uuid string, phase string) {}

// Query mirrors src/conversation-runtime/query.ts export async function* query.
// TS order: yield* queryLoop (no Terminal in yields — Terminal is return value),
// then notifyCommandLifecycle for each consumed uuid, then return terminal.
// Go exposes Terminal as the final [QueryYield] because [iter.Seq2] has no separate return slot.
func Query(ctx context.Context, params QueryParams) iter.Seq2[QueryYield, error] {
	return func(yield func(QueryYield, error) bool) {
		consumed := make([]string, 0, 8)
		term, err := queryLoop(ctx, params, &consumed, yield)
		if err != nil {
			yield(QueryYield{}, err)
			return
		}
		for _, uuid := range consumed {
			CommandLifecycleNotifier(uuid, "completed")
		}
		yield(QueryYield{Terminal: &term}, nil)
	}
}

// queryLoop mirrors query.ts queryLoop return + yields.
// The yield callback receives stream/messages only; [Terminal] is returned, not yielded here.
func queryLoop(ctx context.Context, params QueryParams, consumedCommandUUIDs *[]string, yield func(QueryYield, error) bool) (Terminal, error) {
	_ = consumedCommandUUIDs
	deps := params.Deps
	if deps == nil {
		d := ProductionDeps()
		deps = &d
	}

	state := NewStateFromParams(params)
	_ = state

	cfg := BuildQueryConfig()
	// When params.StreamingParity is true, prefer HTTP SSE + streamingtool over CallModel.
	useStream := params.StreamingParity && StreamingParityPathEnabled(cfg)
	if deps.CallModel != nil || useStream {
		// TS order before callModel: compact slice ([MessagesForQuery]), applyToolResultBudget ([runApplyToolResultBudget]),
		// snip (may yield boundary), microcompact, collapse (N/A here), autocompact,
		// then prependUserContext(messagesForQuery, userContext).
		work := MessagesForQuery(state.Messages)

		agentID := ""
		if state.ToolUseContext.AgentID != nil {
			agentID = strings.TrimSpace(*state.ToolUseContext.AgentID)
		}

		var err error
		work, err = runApplyToolResultBudget(ctx, deps, &ToolResultBudgetInput{
			Messages:                work,
			ContentReplacementState: state.ToolUseContext.ContentReplacementState,
			QuerySource:             params.QuerySource,
			AgentID:                 agentID,
			Tools:                   state.ToolUseContext.Options.Tools,
		})
		if err != nil {
			return Terminal{Reason: TerminalReasonModelError, Error: err}, nil
		}

		snipRes, err := runSnipCompact(ctx, deps, &SnipCompactInput{Messages: work})
		if err != nil {
			return Terminal{Reason: TerminalReasonModelError, Error: err}, nil
		}
		var snipFreed *int
		if snipRes != nil {
			if len(snipRes.Messages) > 0 {
				work = snipRes.Messages
			}
			if snipRes.BoundaryMessage != nil {
				if !yield(QueryYield{Message: snipRes.BoundaryMessage}, nil) {
					return Terminal{Reason: TerminalReasonAbortedStreaming}, nil
				}
			}
			if snipRes.TokensFreed != 0 {
				v := snipRes.TokensFreed
				snipFreed = &v
			}
		}

		var microErr error
		work, microErr = runMicrocompact(ctx, deps, &MicrocompactInput{
			Messages:       work,
			ToolUseContext: &state.ToolUseContext,
			QuerySource:    params.QuerySource,
		})
		if microErr != nil {
			return Terminal{Reason: TerminalReasonModelError, Error: microErr}, nil
		}

		var autoRes *AutocompactResult
		var autoErr error
		work, autoRes, autoErr = runAutocompact(ctx, deps, &AutocompactInput{
			Messages:       work,
			ToolUseContext: &state.ToolUseContext,
			CacheSafe: CacheSafeParams{
				SystemPrompt:        params.SystemPrompt,
				UserContext:         params.UserContext,
				SystemContext:       params.SystemContext,
				ToolUseContext:      &state.ToolUseContext,
				ForkContextMessages: work,
			},
			QuerySource:     params.QuerySource,
			Tracking:        state.AutoCompactTracking,
			SnipTokensFreed: snipFreed,
		})
		if autoErr != nil {
			return Terminal{Reason: TerminalReasonModelError, Error: autoErr}, nil
		}
		applyAutocompactSideEffects(&state, autoRes)

		LogQueryUserContextIfEnabled("before_prepend", params.UserContext)
		msgs := PrependUserContext(work, params.UserContext)

		fullSystem := StripSystemPromptDynamicBoundaryForAPI(
			AppendSystemContext(params.SystemPrompt, params.SystemContext))
		cwd, _ := os.Getwd()
		in := &CallModelInput{
			Messages:       msgs,
			SystemPrompt:   fullSystem,
			ThinkingConfig: params.ToolUseContext.Options.ThinkingConfig,
			Tools:          params.ToolUseContext.Options.Tools,
			SignalDone:     ctx.Done(),
			Cwd:            cwd,
			ModelID:        strings.TrimSpace(params.ToolUseContext.Options.MainLoopModel),
		}
		if useStream {
			var streamErr error
			if StreamingUsesOpenAIChat() {
				streamErr = runOpenAIStreamingParityModelLoop(ctx, params, msgs, in, deps, yield)
			} else {
				streamErr = runStreamingParityModelLoop(ctx, params, msgs, in, deps, yield)
			}
			if streamErr != nil {
				return Terminal{Reason: TerminalReasonModelError, Error: streamErr}, nil
			}
			return Terminal{Reason: TerminalReasonCompleted}, nil
		}
		if deps.CallModel != nil {
			if err := deps.CallModel(ctx, in, func(y QueryYield) bool {
				if y.Terminal != nil {
					return false
				}
				return yield(y, nil)
			}); err != nil {
				return Terminal{Reason: TerminalReasonModelError, Error: err}, nil
			}
			return Terminal{Reason: TerminalReasonCompleted}, nil
		}
		return Terminal{Reason: TerminalReasonModelError, Error: fmt.Errorf("query: CallModel is nil")}, nil
	}

	return Terminal{Reason: TerminalReasonCompleted}, nil
}
