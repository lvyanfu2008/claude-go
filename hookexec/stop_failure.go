package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventStopFailure = "StopFailure"

type stopFailureHookInput struct {
	BaseHookInput
	Error string `json:"error"`
}

// RunStopFailureHooks executes StopFailure command hooks.
// Mirrors TS executeStopFailureHooks in src/utils/hooks.ts.
func RunStopFailureHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	errorStr string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := stopFailureHookInput{
		BaseHookInput: base,
		Error:         errorStr,
	}
	in.HookEventName = hookEventStopFailure

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventStopFailure, r.Command)...)
	}
	return agg, nil
}
