package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/tools/hookstypes"
)

// DefaultPromptHookTimeoutMs mirrors default 30s timeout for prompt hooks in TS execPromptHook.ts.
const DefaultPromptHookTimeoutMs = 30 * 1000

// PromptHookDeps carries injectable dependencies for prompt hook execution.
type PromptHookDeps struct {
	// CallModel synchronously calls an LLM and returns the text response.
	// The system prompt instructs the model to return {"ok": true} or {"ok": false, "reason": "..."}.
	CallModel func(ctx context.Context, systemPrompt, userPrompt, model string) (string, error)
}

// RunPromptHook executes a prompt hook: single-turn LLM evaluation.
// Mirrors TS execPromptHook in execPromptHook.ts.
//
// The model is instructed to return a JSON object with {ok: boolean, reason?: string}.
// If ok is false, the hook blocks; if ok is true, the hook succeeds.
func RunPromptHook(ctx context.Context, h hookstypes.HookCommand, jsonInput string, deps PromptHookDeps) OutsideReplHookResult {
	res := OutsideReplHookResult{
		HookType: "prompt",
		Command:  "[prompt] " + truncateString(h.Prompt, 80),
	}

	if deps.CallModel == nil {
		res.ErrorMessage = "prompt hook: CallModel not configured"
		return res
	}

	// Substitute $ARGUMENTS with the JSON input.
	prompt := substituteArguments(h.Prompt, jsonInput)

	timeoutMs := DefaultPromptHookTimeoutMs
	if h.Timeout > 0 {
		timeoutMs = int(h.Timeout * 1000)
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()

	systemPrompt := `You are evaluating a hook in Claude Code. Your response must be a JSON object matching one of the following schemas:
1. If the condition is met, return: {"ok": true}
2. If the condition is not met, return: {"ok": false, "reason": "..."}

Do not include any text outside the JSON object.`

	model := strings.TrimSpace(h.Model)
	if model == "" {
		model = "claude-haiku-4-5" // default small fast model
	}

	output, err := deps.CallModel(cctx, systemPrompt, prompt, model)
	res.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		if cctx.Err() != nil {
			res.Cancelled = true
			res.ErrorMessage = "prompt hook: cancelled or timed out"
		} else {
			res.ErrorMessage = fmt.Sprintf("prompt hook: %v", err)
		}
		return res
	}

	res.Stdout = output
	res.Output = output

	// Parse and validate the LLM response against {ok: boolean, reason?: string}.
	parsed, validationErr := parseHookResponseJSON(output)
	if validationErr != nil {
		res.ErrorMessage = fmt.Sprintf("prompt hook: JSON validation failed: %v", validationErr)
		return res
	}

	res.JSONParsed = &syncHookStdoutParsed{}

	if parsed.Ok {
		res.Succeeded = true
	} else {
		res.Blocked = true
		reason := strings.TrimSpace(parsed.Reason)
		if reason == "" {
			reason = "Blocked by prompt hook"
		}
		res.ErrorMessage = reason
	}

	return res
}

// hookResponseParsed holds the parsed output from a prompt/agent hook response.
type hookResponseParsed struct {
	Ok     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

// parseHookResponseJSON parses and validates a prompt/agent hook JSON response.
func parseHookResponseJSON(raw string) (*hookResponseParsed, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty response")
	}
	var p hookResponseParsed
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}
	return &p, nil
}

// substituteArguments replaces $ARGUMENTS in a prompt string with the given JSON input.
// Mirrors TS addArgumentsToPrompt / substituteArguments in hookHelpers.ts.
func substituteArguments(prompt, jsonInput string) string {
	prompt = strings.ReplaceAll(prompt, "$ARGUMENTS", jsonInput)
	prompt = strings.ReplaceAll(prompt, "${ARGUMENTS}", jsonInput)
	return prompt
}
