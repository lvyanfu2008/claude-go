package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventFileChanged = "FileChanged"

type fileChangedHookInput struct {
	BaseHookInput
	FilePath   string `json:"file_path"`
	ChangeType string `json:"change_type"`
}

// RunFileChangedHooks executes FileChanged command hooks.
// Mirrors TS executeFileChangedHooks in src/utils/hooks.ts.
func RunFileChangedHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	filePath, changeType string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := fileChangedHookInput{
		BaseHookInput: base,
		FilePath:      filePath,
		ChangeType:    changeType,
	}
	in.HookEventName = hookEventFileChanged

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventFileChanged, r.Command)...)
	}
	return agg, nil
}
