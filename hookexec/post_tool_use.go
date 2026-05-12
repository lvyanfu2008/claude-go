package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

const hookEventPostToolUse = "PostToolUse"

type postToolUseHookInput struct {
	BaseHookInput
	ToolName  string `json:"tool_name"`
	ToolUseID string `json:"tool_use_id"`
	ToolInput any    `json:"tool_input"`
}

// RunPostToolUseHooks executes PostToolUse command hooks and returns aggregated results.
// Mirrors TS executePostToolHooks in src/utils/hooks.ts.
func RunPostToolUseHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	toolName, toolUseID, toolInputJSON string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	var toolInput any
	if strings.TrimSpace(toolInputJSON) != "" {
		json.Unmarshal([]byte(toolInputJSON), &toolInput)
	}

	in := postToolUseHookInput{
		BaseHookInput: base,
		ToolName:      toolName,
		ToolUseID:     toolUseID,
		ToolInput:     toolInput,
	}
	in.HookEventName = hookEventPostToolUse

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
		agg = append(agg, postToolUseAggregates(r, toolUseID, r.Command)...)
	}
	return agg, nil
}

func postToolUseAggregates(r OutsideReplCommandResult, toolUseID, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	if stdout == "" {
		if r.ExitCode == 2 {
			s := strings.TrimSpace(r.Stderr)
			if s == "" {
				s = "No stderr output"
			}
			return []types.AggregatedHookResult{{
				BlockingError: &types.HookBlockingError{
					BlockingError: fmt.Sprintf("[%s]: %s", r.Command, s),
					Command:       r.Command,
				},
			}}
		}
		return nil
	}

	if !strings.HasPrefix(stdout, "{") {
		return postToolUseNonJSONPath(r, toolUseID, hookName)
	}

	parsed, err := parseSyncHookStdoutJSON(stdout)
	if err != nil {
		return postToolUseValidationError(r, toolUseID, hookName, err.Error())
	}

	var out []types.AggregatedHookResult
	top := parsedSyncHookToLegacyTop(parsed)

	if strings.TrimSpace(top.Decision) == "block" {
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by hook"
		}
		out = append(out, types.AggregatedHookResult{
			BlockingError: &types.HookBlockingError{BlockingError: reason, Command: r.Command},
		})
	}

	if strings.TrimSpace(top.SystemMessage) != "" {
		msg, err := serializedHookSystemMessage(toolUseID, hookName, hookEventPostToolUse, strings.TrimSpace(top.SystemMessage))
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

	if strings.TrimSpace(top.Decision) == "block" {
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by hook"
		}
		att := map[string]any{
			"type": "hook_blocking_error",
			"blockingError": map[string]any{
				"blockingError": reason,
				"command":       r.Command,
			},
			"hookName":  hookName,
			"hookEvent": hookEventPostToolUse,
		}
		msg, err := marshalAttachmentMessage(toolUseID, att)
		if err == nil && len(msg) > 0 {
			out = append(out, types.AggregatedHookResult{Message: msg})
		}
	} else {
		msg, err := serializedHookSuccess(toolUseID, hookName, hookEventPostToolUse, "", r.Stdout, r.Stderr, r.ExitCode, r.Command, r.DurationMs)
		if err == nil && len(msg) > 0 {
			out = append(out, types.AggregatedHookResult{Message: msg})
		}
	}

	return out
}

func postToolUseNonJSONPath(r OutsideReplCommandResult, toolUseID, hookName string) []types.AggregatedHookResult {
	exit := r.ExitCode
	if r.Succeeded && exit == 0 {
		msg, err := serializedHookSuccess(toolUseID, hookName, hookEventPostToolUse, r.Stdout, r.Stdout, r.Stderr, exit, r.Command, r.DurationMs)
		if err != nil || len(msg) == 0 {
			return nil
		}
		return []types.AggregatedHookResult{{Message: msg}}
	}
	if exit == 2 {
		s := strings.TrimSpace(r.Stderr)
		if s == "" {
			s = "No stderr output"
		}
		return []types.AggregatedHookResult{{
			BlockingError: &types.HookBlockingError{BlockingError: fmt.Sprintf("[%s]: %s", r.Command, s), Command: r.Command},
		}}
	}
	return nil
}

func postToolUseValidationError(r OutsideReplCommandResult, toolUseID, hookName, detail string) []types.AggregatedHookResult {
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEventPostToolUse, "JSON validation failed: "+detail, r.Stdout, 1, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}
