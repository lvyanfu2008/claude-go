package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

const hookEventUserPromptSubmit = "UserPromptSubmit"

type userPromptSubmitHookInput struct {
	BaseHookInput
	Prompt string `json:"prompt"`
}

// RunUserPromptSubmitHooks runs UserPromptSubmit **command** hooks (TS executeUserPromptSubmitHooks → executeHooks command branch)
// and returns [types.AggregatedHookResult] slices suitable for [processuserinput.ProcessUserInputParams.ExecuteUserPromptSubmitHooks].
func RunUserPromptSubmitHooks(ctx context.Context, table HooksTable, workDir string, base BaseHookInput, prompt string, batchTimeoutMs int) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}
	base.HookEventName = hookEventUserPromptSubmit
	in := userPromptSubmitHookInput{BaseHookInput: base, Prompt: prompt}
	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return nil, err
	}
	var hookInput map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonIn)), &hookInput); err != nil {
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
	toolUseID := randomUUID()
	var agg []types.AggregatedHookResult
	for _, r := range results {
		agg = append(agg, userPromptSubmitAggregates(r, toolUseID, hookEventUserPromptSubmit, r.Command)...)
	}
	return agg, nil
}

func validateUserPromptSubmitDecision(dec string) error {
	dec = strings.TrimSpace(dec)
	if dec == "" || dec == "approve" || dec == "block" {
		return nil
	}
	return fmt.Errorf("Unknown hook decision type: %s. Valid types are: approve, block", dec)
}

func userPromptSubmitAggregates(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	trimmed := strings.TrimSpace(stdout)

	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return userPromptSubmitNonJSONPath(r, toolUseID, hookEvent, hookName)
	}

	var asyncProbe struct {
		Async *bool `json:"async"`
	}
	if err := json.Unmarshal([]byte(trimmed), &asyncProbe); err != nil {
		return userPromptSubmitNonJSONPath(r, toolUseID, hookEvent, hookName)
	}
	if asyncProbe.Async != nil && *asyncProbe.Async {
		return nil
	}

	var top syncUserPromptSubmitJSON
	if err := json.Unmarshal([]byte(trimmed), &top); err != nil {
		return userPromptSubmitValidationError(r, toolUseID, hookEvent, hookName, fmt.Sprintf("Failed to parse hook output as JSON: %v", err))
	}

	if err := validateUserPromptSubmitDecision(top.Decision); err != nil {
		return userPromptSubmitValidationError(r, toolUseID, hookEvent, hookName, err.Error())
	}

	if len(top.HookSpecificOutput) > 0 {
		var hso struct {
			HookEventName string `json:"hookEventName"`
		}
		_ = json.Unmarshal(top.HookSpecificOutput, &hso)
		if hso.HookEventName != "" && hso.HookEventName != hookEventUserPromptSubmit {
			return userPromptSubmitValidationError(r, toolUseID, hookEvent, hookName,
				fmt.Sprintf("Hook returned incorrect event name: expected %q but got %q. Full stdout: %s", hookEventUserPromptSubmit, hso.HookEventName, trimmed))
		}
	}

	var out []types.AggregatedHookResult

	if top.Continue != nil && !*top.Continue {
		item := types.AggregatedHookResult{PreventContinuation: boolPtr(true)}
		if top.StopReason != nil && strings.TrimSpace(*top.StopReason) != "" {
			sr := strings.TrimSpace(*top.StopReason)
			item.StopReason = &sr
		}
		out = append(out, item)
	}

	switch strings.TrimSpace(top.Decision) {
	case "block":
		reason := strings.TrimSpace(top.Reason)
		if reason == "" {
			reason = "Blocked by hook"
		}
		out = append(out, types.AggregatedHookResult{
			BlockingError: &types.HookBlockingError{BlockingError: reason, Command: r.Command},
		})
	case "approve":
		allow := "allow"
		item := types.AggregatedHookResult{PermissionBehavior: &allow}
		if strings.TrimSpace(top.Reason) != "" {
			rs := strings.TrimSpace(top.Reason)
			item.HookPermissionDecisionReason = &rs
		}
		out = append(out, item)
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

	msg, err := serializedHookProcessJSONMessage(r, toolUseID, hookEvent, hookName, top)
	if err == nil && len(msg) > 0 {
		out = append(out, types.AggregatedHookResult{Message: msg})
	}

	return out
}

type syncUserPromptSubmitJSON struct {
	Continue           *bool           `json:"continue"`
	StopReason         *string         `json:"stopReason"`
	Decision           string          `json:"decision"`
	Reason             string          `json:"reason"`
	SystemMessage      string          `json:"systemMessage"`
	SuppressOutput     *bool           `json:"suppressOutput"`
	HookSpecificOutput json.RawMessage `json:"hookSpecificOutput"`
}

func userPromptSubmitNonJSONPath(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string) []types.AggregatedHookResult {
	stdout := strings.TrimSpace(r.Stdout)
	stderr := strings.TrimSpace(r.Stderr)
	exit := r.ExitCode

	if r.Succeeded && exit == 0 {
		if stdout == "" {
			return nil
		}
		msg, err := serializedHookSuccess(toolUseID, hookName, hookEvent, stdout, r.Stdout, r.Stderr, exit, r.Command, r.DurationMs)
		if err != nil || len(msg) == 0 {
			return nil
		}
		return []types.AggregatedHookResult{{Message: msg}}
	}

	if exit == 2 {
		s := stderr
		if s == "" {
			s = "No stderr output"
		}
		blocking := fmt.Sprintf("[%s]: %s", r.Command, s)
		return []types.AggregatedHookResult{{
			BlockingError: &types.HookBlockingError{BlockingError: blocking, Command: r.Command},
		}}
	}

	errLine := stderr
	if errLine == "" {
		errLine = "No stderr output"
	}
	detail := fmt.Sprintf("Failed with non-blocking status code: %s", errLine)
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEvent, detail, r.Stdout, exit, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}

func userPromptSubmitValidationError(r OutsideReplCommandResult, toolUseID, hookEvent, hookName, detail string) []types.AggregatedHookResult {
	msg, err := serializedHookNonBlockingError(toolUseID, hookName, hookEvent, "JSON validation failed: "+detail, r.Stdout, 1, r.Command, r.DurationMs)
	if err != nil || len(msg) == 0 {
		return nil
	}
	return []types.AggregatedHookResult{{Message: msg}}
}

func serializedHookProcessJSONMessage(r OutsideReplCommandResult, toolUseID, hookEvent, hookName string, top syncUserPromptSubmitJSON) (json.RawMessage, error) {
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
			"hookEvent": hookEvent,
		}
		return marshalAttachmentMessage(toolUseID, att)
	}
	content := ""
	if suppressOutputFalse(top.SuppressOutput) && strings.TrimSpace(r.Stdout) != "" && r.ExitCode == 0 && r.Succeeded {
		content = ""
	}
	return serializedHookSuccess(toolUseID, hookName, hookEvent, content, r.Stdout, r.Stderr, r.ExitCode, r.Command, r.DurationMs)
}

func suppressOutputFalse(p *bool) bool {
	return p == nil || !*p
}

func marshalAttachmentMessage(toolUseID string, attachment map[string]any) (json.RawMessage, error) {
	attachment["toolUseID"] = toolUseID
	rawAtt, err := json.Marshal(attachment)
	if err != nil {
		return nil, err
	}
	msg := map[string]any{
		"type":       string(types.MessageTypeAttachment),
		"uuid":       randomUUID(),
		"attachment": json.RawMessage(rawAtt),
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func serializedHookSuccess(toolUseID, hookName, hookEvent, content, stdout, stderr string, exitCode int, command string, durationMs int64) (json.RawMessage, error) {
	att := map[string]any{
		"type":       "hook_success",
		"content":    content,
		"hookName":   hookName,
		"hookEvent":  hookEvent,
		"stdout":     stdout,
		"stderr":     stderr,
		"exitCode":   exitCode,
		"command":    command,
		"durationMs": durationMs,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func serializedHookNonBlockingError(toolUseID, hookName, hookEvent, stderr, stdout string, exitCode int, command string, durationMs int64) (json.RawMessage, error) {
	att := map[string]any{
		"type":       "hook_non_blocking_error",
		"hookName":   hookName,
		"stderr":     stderr,
		"stdout":     stdout,
		"exitCode":   exitCode,
		"hookEvent":  hookEvent,
		"command":    command,
		"durationMs": durationMs,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func serializedHookSystemMessage(toolUseID, hookName, hookEvent, content string) (json.RawMessage, error) {
	att := map[string]any{
		"type":      "hook_system_message",
		"content":   content,
		"hookName":  hookName,
		"hookEvent": hookEvent,
	}
	return marshalAttachmentMessage(toolUseID, att)
}

func boolPtr(b bool) *bool { return &b }
