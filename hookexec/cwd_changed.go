package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventCwdChanged = "CwdChanged"

type cwdChangedHookInput struct {
	BaseHookInput
	PreviousCwd string `json:"previous_cwd"`
	NewCwd      string `json:"new_cwd"`
}

// RunCwdChangedHooks executes CwdChanged command hooks.
// Mirrors TS executeCwdChangedHooks in src/utils/hooks.ts.
func RunCwdChangedHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	previousCwd, newCwd string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := cwdChangedHookInput{
		BaseHookInput: base,
		PreviousCwd:   previousCwd,
		NewCwd:        newCwd,
	}
	in.HookEventName = hookEventCwdChanged

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventCwdChanged, r.Command)...)
	}
	return agg, nil
}
