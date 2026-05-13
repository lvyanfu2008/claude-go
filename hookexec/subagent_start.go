package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventSubagentStart = "SubagentStart"

type subagentStartHookInput struct {
	BaseHookInput
	AgentType string `json:"agent_type"`
}

// RunSubagentStartHooks executes SubagentStart command hooks.
// Mirrors TS executeSubagentStartHooks in src/utils/hooks.ts.
func RunSubagentStartHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	agentType string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := subagentStartHookInput{
		BaseHookInput: base,
		AgentType:     agentType,
	}
	in.HookEventName = hookEventSubagentStart

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventSubagentStart, r.Command)...)
	}
	return agg, nil
}
