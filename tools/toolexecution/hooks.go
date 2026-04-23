package toolexecution

import (
	"context"
	"encoding/json"
	"iter"
)

// RunPreToolUseHooks mirrors the hook phase invoked from checkPermissionsAndCallTool (toolHooks.ts / utils/hooks.ts).
// When [ExecutionDeps.PreToolUseHook] is set, it runs first; otherwise this is a no-op until the full hook port lands.
func RunPreToolUseHooks(ctx context.Context, deps ExecutionDeps, toolName, toolUseID string, input json.RawMessage) error {
	if deps.PreToolUseHook != nil {
		return deps.PreToolUseHook(ctx, toolName, toolUseID, input)
	}
	return nil
}

// RunPostToolUseHooks mirrors runPostToolUseHooks in toolHooks.ts — skeleton yields nothing.
// TODO(toolHooks.ts): port executePostToolHooks and attachment/progress yields.
func RunPostToolUseHooks(ctx context.Context, deps ExecutionDeps, toolName, toolUseID string) iter.Seq2[MessageUpdate, error] {
	_ = ctx
	_ = deps
	_ = toolName
	_ = toolUseID
	return func(yield func(MessageUpdate, error) bool) {
		_ = yield
	}
}
