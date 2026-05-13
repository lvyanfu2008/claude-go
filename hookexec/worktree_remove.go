package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventWorktreeRemove = "WorktreeRemove"

type worktreeRemoveHookInput struct {
	BaseHookInput
	WorktreePath string `json:"worktree_path"`
}

// RunWorktreeRemoveHooks executes WorktreeRemove command hooks.
// Mirrors TS execute*WorktreeRemove* in src/utils/hooks.ts.
func RunWorktreeRemoveHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	worktreePath string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := worktreeRemoveHookInput{
		BaseHookInput: base,
		WorktreePath:  worktreePath,
	}
	in.HookEventName = hookEventWorktreeRemove

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventWorktreeRemove, r.Command)...)
	}
	return agg, nil
}
