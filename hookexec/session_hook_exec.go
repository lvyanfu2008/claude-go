package hookexec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AgentPreToolUseHookFromSession builds a PreToolUseHook function that
// executes PreToolUse command hooks from the agent's session-scoped hooks.
//
// The returned function matches the toolexecution.PreToolUseHook signature:
// returns nil to allow the tool, or an error to block it.
//
// TS equivalent: executePreToolHooks → getMatchingHooks for PreToolUse event
// → executeHooks with the matched hooks.
func AgentPreToolUseHookFromSession(agentID, workDir string) func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error {
	return func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) error {
		hooks := GetSessionHooks(agentID)
		if len(hooks) == 0 {
			return nil
		}

		// Build hook input matching TS PreToolUseHookInput.
		hookInput := map[string]any{
			"hook_event_name": "PreToolUse",
			"tool_name":       toolName,
			"tool_use_id":     toolUseID,
		}

		// Get matching command hooks using the same matchQuery rules as TS.
		cmdHooks := CommandHooksForHookInput(hooks, hookInput)
		if len(cmdHooks) == 0 {
			return nil
		}

		jsonIn, _ := json.Marshal(hookInput)

		wd := trimOrDot(workDir)
		results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
			Ctx:       ctx,
			WorkDir:   wd,
			Hooks:     hooks,
			JSONInput: string(jsonIn),
			TimeoutMs: DefaultHookTimeoutMs,
		})

		// Check for blocking hooks (TS: decision === "block" or exit code 2).
		for _, r := range results {
			if r.Blocked {
				msg := strings.TrimSpace(r.Output)
				if msg == "" {
					msg = fmt.Sprintf("hook blocked tool %q", toolName)
				}
				return fmt.Errorf("%s", msg)
			}
			if !r.Succeeded {
				msg := strings.TrimSpace(r.Output)
				if msg == "" {
					msg = fmt.Sprintf("hook failed for tool %q", toolName)
				}
				return fmt.Errorf("%s", msg)
			}
		}

		return nil
	}
}

// AgentMergedHooksTable merges settings-file hooks with session hooks for the given agent ID.
// This is the Go equivalent of TS getHooksConfig() for agent-scoped execution.
func AgentMergedHooksTable(projectRoot string, agentID string) (HooksTable, error) {
	settingsTable, err := MergedHooksFromPaths(projectRoot)
	if err != nil {
		return nil, err
	}
	sessionTable := MergeSessionHookTables(agentID)
	return mergeHooksTable(settingsTable, sessionTable), nil
}
