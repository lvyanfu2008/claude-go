package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventTaskCompleted = "TaskCompleted"

type taskCompletedHookInput struct {
	BaseHookInput
	TaskID     string `json:"task_id"`
	TaskListID string `json:"task_list_id"`
}

// RunTaskCompletedHooks executes TaskCompleted command hooks.
// Mirrors TS executeTaskCompletedHooks in src/utils/hooks.ts.
func RunTaskCompletedHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	taskID, taskListID string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := taskCompletedHookInput{
		BaseHookInput: base,
		TaskID:        taskID,
		TaskListID:    taskListID,
	}
	in.HookEventName = hookEventTaskCompleted

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventTaskCompleted, r.Command)...)
	}
	return agg, nil
}
