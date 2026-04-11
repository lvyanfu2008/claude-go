package query

import (
	"context"

	"goc/conversation-runtime/streamingtool"
	"goc/toolexecution"
	"goc/types"
)

// RunToolUseToolRunner implements [streamingtool.ToolRunner] by delegating to [toolexecution.RunToolUseChan].
// ParentCtx should be the query / model-call context (cancel on turn abort).
type RunToolUseToolRunner struct {
	ParentCtx context.Context
	Deps      toolexecution.ExecutionDeps
}

// RunToolUpdates implements [streamingtool.ToolRunner].
func (r RunToolUseToolRunner) RunToolUpdates(
	block streamingtool.ToolUseBlock,
	assistant types.Message,
	canUseTool any,
	toolCtx streamingtool.ToolUseContextPort,
	toolAbort *streamingtool.AbortController,
) <-chan streamingtool.ToolRunUpdate {
	parent := r.ParentCtx
	if parent == nil {
		parent = context.Background()
	}
	_ = toolCtx
	effective := r.Deps
	if fn, ok := canUseTool.(toolexecution.QueryCanUseToolFn); ok && fn != nil {
		d := r.Deps
		d.QueryCanUseTool = fn
		effective = d
	}
	return toolexecution.RunToolUseChan(parent, block, assistant, effective, toolAbort)
}
