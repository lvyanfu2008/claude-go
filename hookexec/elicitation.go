package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventElicitation = "Elicitation"

type elicitationHookInput struct {
	BaseHookInput
	MCPServerName string `json:"mcp_server_name"`
	ElicitationID string `json:"elicitation_id"`
	Request       any    `json:"request"`
}

// RunElicitationHooks executes Elicitation command hooks.
// Mirrors TS executeElicitationHooks in src/utils/hooks.ts.
func RunElicitationHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	mcpServerName, elicitationID, requestJSON string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	var request any
	if requestJSON != "" {
		json.Unmarshal([]byte(requestJSON), &request)
	}

	in := elicitationHookInput{
		BaseHookInput: base,
		MCPServerName: mcpServerName,
		ElicitationID: elicitationID,
		Request:       request,
	}
	in.HookEventName = hookEventElicitation

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventElicitation, r.Command)...)
	}
	return agg, nil
}

const hookEventElicitationResult = "ElicitationResult"

type elicitationResultHookInput struct {
	BaseHookInput
	MCPServerName string `json:"mcp_server_name"`
	ElicitationID string `json:"elicitation_id"`
	Result        any    `json:"result"`
}

// RunElicitationResultHooks executes ElicitationResult command hooks.
// Mirrors TS executeElicitationResultHooks in src/utils/hooks.ts.
func RunElicitationResultHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	mcpServerName, elicitationID, resultJSON string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	var result any
	if resultJSON != "" {
		json.Unmarshal([]byte(resultJSON), &result)
	}

	in := elicitationResultHookInput{
		BaseHookInput: base,
		MCPServerName: mcpServerName,
		ElicitationID: elicitationID,
		Result:        result,
	}
	in.HookEventName = hookEventElicitationResult

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
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventElicitationResult, r.Command)...)
	}
	return agg, nil
}
