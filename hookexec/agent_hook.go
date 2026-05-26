package hookexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goc/tools/hookstypes"
)

// DefaultAgentHookTimeoutMs mirrors default 60s timeout for agent hooks in TS execAgentHook.ts.
const DefaultAgentHookTimeoutMs = 60 * 1000

// MaxAgentHookTurns mirrors TS MAX_AGENT_TURNS in execAgentHook.ts.
const MaxAgentHookTurns = 50

// AgentHookDeps carries injectable dependencies for agent hook execution.
type AgentHookDeps struct {
	// RunAgentQuery executes a multi-turn agentic verification and returns the
	// structured output result. The implementation should:
	// 1. Create a unique agent ID (hook-agent-<uuid>)
	// 2. Use "dontAsk" permission mode with pre-allowed file system reads
	// 3. Block agent-disallowed tools
	// 4. Inject SyntheticOutputTool for structured output
	// 5. Run the query loop with max 50 turns
	// 6. Parse and return the structured output
	RunAgentQuery func(ctx context.Context, params AgentQueryParams) (*hookResponseParsed, error)
}

// AgentQueryParams is the input to the agent hook query executor.
type AgentQueryParams struct {
	SystemPrompt string // the agent's system prompt
	UserPrompt   string // the user's prompt (from $ARGUMENTS substitution)
	Model        string // the model to use
	TranscriptPath string // path to the session transcript for context
	HookAgentID  string // unique ID for this hook agent
	WorkDir      string // working directory
	TimeoutMs    int    // timeout in milliseconds
}

// NewSingleTurnAgentHookRunner creates a RunAgentQuery implementation that uses
// CallModel for a single-turn call (no tool loop). For full multi-turn agentic
// verification, wire a custom RunAgentQuery that invokes tools.executeAgent.
func NewSingleTurnAgentHookRunner(callModel PromptHookDeps) AgentHookDeps {
	return AgentHookDeps{
		RunAgentQuery: func(ctx context.Context, params AgentQueryParams) (*hookResponseParsed, error) {
			resp, err := callModel.CallModel(ctx, params.SystemPrompt, params.UserPrompt, params.Model)
			if err != nil {
				return nil, err
			}
			return parseHookResponseJSON(resp)
		},
	}
}

// RunAgentHook executes an agent hook: multi-turn agentic verification.
// Mirrors TS execAgentHook in execAgentHook.ts.
//
// The agent is given tools to inspect the codebase and must call SyntheticOutputTool
// with {ok: boolean, reason?: string}. If ok is false, the hook blocks.
func RunAgentHook(ctx context.Context, h hookstypes.HookCommand, jsonInput string, deps AgentHookDeps) OutsideReplHookResult {
	res := OutsideReplHookResult{
		HookType: "agent",
		Command:  "[agent] " + truncateString(h.Prompt, 80),
	}

	if deps.RunAgentQuery == nil {
		res.ErrorMessage = "agent hook: RunAgentQuery not configured"
		return res
	}

	// Substitute $ARGUMENTS with the JSON input.
	prompt := substituteArguments(h.Prompt, jsonInput)

	timeoutMs := DefaultAgentHookTimeoutMs
	if h.Timeout > 0 {
		timeoutMs = int(h.Timeout * 1000)
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()

	model := strings.TrimSpace(h.Model)
	if model == "" {
		model = "claude-haiku-4-5" // default small fast model
	}

	// Build a system prompt instructing the agent to verify the condition.
	systemPrompt := fmt.Sprintf(
		`You are verifying a stop condition in Harness Code. Your task is to verify
that the agent completed the given plan.

Use the available tools to inspect the codebase and verify the condition.
Use as few steps as possible - be efficient and direct.

When done, return your result using the StructuredOutput tool with:
- ok: true if the condition is met
- ok: false with reason if the condition is not met`)

	params := AgentQueryParams{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
		Model:        model,
		TimeoutMs:    timeoutMs,
	}

	parsed, err := deps.RunAgentQuery(cctx, params)
	res.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		if cctx.Err() != nil {
			res.Cancelled = true
			res.ErrorMessage = "agent hook: cancelled or timed out"
		} else {
			res.ErrorMessage = fmt.Sprintf("agent hook: %v", err)
		}
		return res
	}

	if parsed == nil {
		res.ErrorMessage = "agent hook: no structured output received (max turns exceeded)"
		return res
	}

	if parsed.Ok {
		res.Succeeded = true
	} else {
		res.Blocked = true
		reason := strings.TrimSpace(parsed.Reason)
		if reason == "" {
			reason = "Blocked by agent hook"
		}
		res.ErrorMessage = reason
	}

	return res
}
