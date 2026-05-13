package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventPermissionDenied = "PermissionDenied"

type permissionDeniedHookInput struct {
	BaseHookInput
	ToolName  string `json:"tool_name"`
	ToolUseID string `json:"tool_use_id"`
	ToolInput any    `json:"tool_input"`
	Reason    string `json:"reason"`
}

// RunPermissionDeniedHooks executes PermissionDenied command hooks.
// Mirrors TS executePermissionDeniedHooks in src/utils/hooks.ts.
func RunPermissionDeniedHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	toolName, toolUseID, toolInputJSON, reason string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	var toolInput any
	if toolInputJSON != "" {
		json.Unmarshal([]byte(toolInputJSON), &toolInput)
	}

	in := permissionDeniedHookInput{
		BaseHookInput: base,
		ToolName:      toolName,
		ToolUseID:     toolUseID,
		ToolInput:     toolInput,
		Reason:        reason,
	}
	in.HookEventName = hookEventPermissionDenied

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
		agg = append(agg, hookAggregate(r, toolUseID, hookEventPermissionDenied, r.Command)...)
	}
	return agg, nil
}
