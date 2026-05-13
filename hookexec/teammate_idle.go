package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventTeammateIdle = "TeammateIdle"

type teammateIdleHookInput struct {
	BaseHookInput
	TeammateName string `json:"teammate_name"`
	TeamName     string `json:"team_name"`
}

// RunTeammateIdleHooks executes TeammateIdle command hooks.
// Mirrors TS executeTeammateIdleHooks in src/utils/hooks.ts.
func RunTeammateIdleHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	teammateName, teamName string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := teammateIdleHookInput{
		BaseHookInput: base,
		TeammateName:  teammateName,
		TeamName:      teamName,
	}
	in.HookEventName = hookEventTeammateIdle

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventTeammateIdle, r.Command)...)
	}
	return agg, nil
}
