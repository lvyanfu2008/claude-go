package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventPermissionRequest = "PermissionRequest"

type permissionRequestHookInput struct {
	BaseHookInput
	ToolName  string `json:"tool_name"`
	ToolUseID string `json:"tool_use_id"`
	ToolInput any    `json:"tool_input"`
}

// RunPermissionRequestHooks executes PermissionRequest command hooks.
// Mirrors TS executePermissionRequestHooks in src/utils/hooks.ts.
func RunPermissionRequestHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	toolName, toolUseID, toolInputJSON string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	var toolInput any
	if toolInputJSON != "" {
		json.Unmarshal([]byte(toolInputJSON), &toolInput)
	}

	in := permissionRequestHookInput{
		BaseHookInput: base,
		ToolName:      toolName,
		ToolUseID:     toolUseID,
		ToolInput:     toolInput,
	}
	in.HookEventName = hookEventPermissionRequest

	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return nil, err
	}

	var hookInput map[string]any
	if err := json.Unmarshal([]byte(jsonIn), &hookInput); err != nil {
		return nil, err
	}
	if len(CommandHooksForHookInput(table, hookInput)) == 0 {
		return nil, nil
	}

	wd := trimOrDot(workDir)
	results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
		Ctx:       ctx,
		WorkDir:   wd,
		Hooks:     table,
		JSONInput: jsonIn,
		TimeoutMs: batchTimeoutMs,
	})

	var agg []types.AggregatedHookResult
	for _, r := range results {
		agg = append(agg, hookAggregate(r, toolUseID, hookEventPermissionRequest, r.Command)...)
	}
	return agg, nil
}
