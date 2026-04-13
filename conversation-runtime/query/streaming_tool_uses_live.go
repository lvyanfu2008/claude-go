package query

import (
	"context"
)

// StreamingToolUseLive mirrors claude-code StreamingToolUse during SSE: block index, tool id/name,
// and unparsed tool JSON accumulated from input_json_delta (REPL.tsx streamingToolUses).
type StreamingToolUseLive struct {
	Index         int
	ToolUseID     string
	Name          string
	UnparsedInput string
}

// See [QueryDeps.OnStreamingToolUses] for callback semantics.

func depsOnStreamingToolUses(deps *QueryDeps) func(context.Context, []StreamingToolUseLive) error {
	if deps == nil || deps.OnStreamingToolUses == nil {
		return nil
	}
	return deps.OnStreamingToolUses
}

func notifyStreamingToolUsesSnapshot(ctx context.Context, deps *QueryDeps, acc *assistantStreamAccumulator) {
	fn := depsOnStreamingToolUses(deps)
	if fn == nil || acc == nil {
		return
	}
	_ = fn(ctx, acc.StreamingToolUsesLive())
}

func notifyStreamingToolUsesClear(ctx context.Context, deps *QueryDeps) {
	fn := depsOnStreamingToolUses(deps)
	if fn == nil {
		return
	}
	_ = fn(ctx, nil)
}
