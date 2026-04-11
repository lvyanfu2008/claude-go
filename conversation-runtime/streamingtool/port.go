package streamingtool

import (
	"goc/types"
)

// ToolUseContextPort mirrors the ToolUseContext fields StreamingToolExecutor.ts reads/writes
// (abortController, setInProgressToolUseIDs, setHasInterruptibleToolInProgress).
type ToolUseContextPort interface {
	QueryAbort() *AbortController
	SetInProgressToolUseIDs(updater func(prev map[string]struct{}) map[string]struct{})
	SetHasInterruptibleToolInProgress(v bool)
}

// ToolBehavior mirrors the Tool fields used in addTool / getToolInterruptBehavior (name, inputSchema, isConcurrencySafe, interruptBehavior).
type ToolBehavior interface {
	Name() string
	// InputOK mirrors inputSchema.safeParse(block.input); return (parsed, true) on success.
	InputOK(input []byte) (parsed any, ok bool)
	IsConcurrencySafe(parsed any) bool
	// InterruptBehavior returns "cancel" or "block" (TS default when missing: block).
	InterruptBehavior() string
}

// ToolRunner mirrors runToolUse async generator: host sends updates on ch then closes.
type ToolRunner interface {
	RunToolUpdates(
		block ToolUseBlock,
		assistant types.Message,
		canUseTool any,
		toolCtx ToolUseContextPort,
		toolAbort *AbortController,
	) <-chan ToolRunUpdate
}
