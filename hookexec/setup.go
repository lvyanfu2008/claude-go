package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventSetup = "Setup"

type setupHookInput struct {
	BaseHookInput
	Trigger string `json:"trigger"`
}

// RunSetupHooks executes Setup command hooks.
// Mirrors TS executeSetupHooks in src/utils/hooks.ts.
func RunSetupHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	trigger string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := setupHookInput{
		BaseHookInput: base,
		Trigger:       trigger,
	}
	in.HookEventName = hookEventSetup

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventSetup, r.Command)...)
	}
	return agg, nil
}
