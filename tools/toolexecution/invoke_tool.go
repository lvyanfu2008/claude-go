package toolexecution

import (
	"context"
	"encoding/json"
)

// InvokeToolFunc is the contract for [ExecutionDeps.InvokeTool].
// When set, [RunToolUseChan] and [finishCheckPermissionsWithToolCall] invoke it before falling back to [Tool.Call] on the registry tool.
//
// Reference host implementation with Read / Write / Edit / Bash / Glob / Grep parity: [goc/tools/skilltools.ParityToolRunner].
type InvokeToolFunc func(ctx context.Context, name, toolUseID string, input json.RawMessage) (content string, isError bool, err error)
