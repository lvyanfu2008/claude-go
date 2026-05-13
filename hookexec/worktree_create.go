package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventWorktreeCreate = "WorktreeCreate"

type worktreeCreateHookInput struct {
	BaseHookInput
	WorktreePath string `json:"worktree_path"`
	BaseRef      string `json:"base_ref"`
	Branch       string `json:"branch"`
}

// RunWorktreeCreateHooks executes WorktreeCreate command hooks.
// Mirrors TS execute*WorktreeCreate* in src/utils/hooks.ts.
func RunWorktreeCreateHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	worktreePath, baseRef, branch string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := worktreeCreateHookInput{
		BaseHookInput: base,
		WorktreePath:  worktreePath,
		BaseRef:       baseRef,
		Branch:        branch,
	}
	in.HookEventName = hookEventWorktreeCreate

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventWorktreeCreate, r.Command)...)
	}
	return agg, nil
}
