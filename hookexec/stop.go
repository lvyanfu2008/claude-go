package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

type stopHookInput struct {
	BaseHookInput
	StopHookActive       bool   `json:"stop_hook_active"`
	AgentID              string `json:"agent_id,omitempty"`
	AgentType            string `json:"agent_type,omitempty"`
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
	AgentTranscriptPath  string `json:"agent_transcript_path,omitempty"`
}

// RunStopHooks executes Stop (or SubagentStop) command hooks and returns aggregated results.
// Mirrors TS executeStopHooks in src/utils/hooks.ts.
//
// When subagentID is non-empty, hooks fire as SubagentStop instead of Stop.
func RunStopHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	stopHookActive bool,
	subagentID string,
	agentType string,
	lastAssistantText string,
	agentTranscriptPath string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	hookEvent := hookEventStop
	if strings.TrimSpace(subagentID) != "" {
		hookEvent = hookEventSubagentStop
	}

	type hookInputWire struct {
		BaseHookInput
		HookEventName        string `json:"hook_event_name"`
		StopHookActive       bool   `json:"stop_hook_active"`
		AgentID              string `json:"agent_id,omitempty"`
		AgentType            string `json:"agent_type,omitempty"`
		LastAssistantMessage string `json:"last_assistant_message,omitempty"`
		AgentTranscriptPath  string `json:"agent_transcript_path,omitempty"`
	}
	wire := hookInputWire{
		BaseHookInput:        base,
		HookEventName:        hookEvent,
		StopHookActive:       stopHookActive,
		LastAssistantMessage: lastAssistantText,
	}
	if strings.TrimSpace(subagentID) != "" {
		wire.AgentID = subagentID
		wire.AgentType = agentType
		wire.AgentTranscriptPath = agentTranscriptPath
	}

	jsonIn, err := marshalHookInput(wire)
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
		agg = append(agg, stopAggregates(r, randomUUID(), hookEvent, r.Command)...)
	}
	return agg, nil
}

func stopAggregates(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	if stdout == "" {
		if r.ExitCode == 2 {
			s := strings.TrimSpace(r.Stderr)
			if s == "" {
				s = "No stderr output"
			}
			return []types.AggregatedHookResult{{
				BlockingError: &types.HookBlockingError{
					BlockingError: fmt.Sprintf("Stop hook blocking error from command %q: %s", r.Command, s),
					Command:       r.Command,
				},
			}}
		}
		return nil
	}

	if !strings.HasPrefix(stdout, "{") {
		return stopNonJSONPath(r, toolUseID, hookEvent, hookName)
	}

	parsed, err := parseSyncHookStdoutJSON(stdout)
	if err != nil {
		return stopValidationError(r, toolUseID, hookEvent, hookName, err.Error())
	}

	var out []types.AggregatedHookResult
	top := parsedSyncHookToLegacyTop(parsed)

	if strings.TrimSpace(top.Decision) == "block" {
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by stop hook"
		}
		out = append(out, types.AggregatedHookResult{
			BlockingError: &types.HookBlockingError{
				BlockingError: fmt.Sprintf("Stop hook blocking error from command %q: %s", r.Command, reason),
				Command:       r.Command,
			},
		})
	}

	if strings.TrimSpace(top.SystemMessage) != "" {
		msg, err := serializedHookSystemMessage(toolUseID, hookName, hookEvent, strings.TrimSpace(top.SystemMessage))
		if err == nil && len(msg) > 0 {
			out = append(out, types.AggregatedHookResult{Message: msg})
		}
	}

	if len(top.HookSpecificOutput) > 0 {
		var hso struct {
			AdditionalContext string `json:"additionalContext"`
		}
		if err := json.Unmarshal(top.HookSpecificOutput, &hso); err == nil && strings.TrimSpace(hso.AdditionalContext) != "" {
			out = append(out, types.AggregatedHookResult{
				AdditionalContexts: []string{hso.AdditionalContext},
			})
		}
	}

	return out
}

func stopNonJSONPath(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	exit := r.ExitCode
	if r.Succeeded && exit == 0 {
		return nil
	}
	if exit == 2 {
		s := strings.TrimSpace(r.Stderr)
		if s == "" {
			s = "No stderr output"
		}
		return []types.AggregatedHookResult{{
			BlockingError: &types.HookBlockingError{
				BlockingError: fmt.Sprintf("Stop hook blocking error from command %q: %s", r.Command, s),
				Command:       r.Command,
			},
		}}
	}
	return nil
}

func stopValidationError(r OutsideReplCommandResult, toolUseID, hookEvent, hookName, detail string) []types.AggregatedHookResult {
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEvent, "JSON validation failed: "+detail, r.Stdout, 1, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}
