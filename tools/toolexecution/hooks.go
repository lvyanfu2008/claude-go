package toolexecution

import (
	"context"
	"encoding/json"
	"iter"

	"goc/types"
)

// PostToolUseHookInput is the input data passed to post-tool-use hooks.
type PostToolUseHookInput struct {
	ToolName     string
	ToolUseID    string
	ToolInput    json.RawMessage
	ToolResponse string
}

// PostToolUseHookRunner is called after a tool completes successfully.
// When set on [ExecutionDeps], [RunPostToolUseHooks] delegates to it.
// Returns aggregated hook results (blocking errors, messages, additional contexts).
type PostToolUseHookRunner func(ctx context.Context, input PostToolUseHookInput, deps ExecutionDeps) ([]types.AggregatedHookResult, error)

// RunPreToolUseHooks mirrors the hook phase invoked from checkPermissionsAndCallTool (toolHooks.ts / utils/hooks.ts).
// When [ExecutionDeps.PreToolUseHook] is set, it runs first; otherwise this is a no-op until the full hook port lands.
func RunPreToolUseHooks(ctx context.Context, deps ExecutionDeps, toolName, toolUseID string, input json.RawMessage) error {
	if deps.PreToolUseHook != nil {
		return deps.PreToolUseHook(ctx, toolName, toolUseID, input)
	}
	return nil
}

// RunPostToolUseHooks mirrors runPostToolUseHooks in toolHooks.ts.
// When [ExecutionDeps.PostToolUseHookRunner] is set, it invokes the runner and yields
// messages for each hook result. Otherwise this is a no-op.
func RunPostToolUseHooks(ctx context.Context, deps ExecutionDeps, toolName, toolUseID string, toolInput json.RawMessage, toolResponse string) iter.Seq2[MessageUpdate, error] {
	return func(yield func(MessageUpdate, error) bool) {
		if deps.PostToolUseHookRunner == nil {
			return
		}
		results, err := deps.PostToolUseHookRunner(ctx, PostToolUseHookInput{
			ToolName:     toolName,
			ToolUseID:    toolUseID,
			ToolInput:    toolInput,
			ToolResponse: toolResponse,
		}, deps)
		if err != nil {
			yield(MessageUpdate{}, err)
			return
		}
		for _, r := range results {
			if len(r.Message) > 0 {
				var msg types.Message
				if err := json.Unmarshal(r.Message, &msg); err == nil {
					yield(MessageUpdate{Message: &msg}, nil)
				}
			}
		}
	}
}
