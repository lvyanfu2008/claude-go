package hookexec

import (
	"context"
	"encoding/json"
	"os"
	"strings"
)

const hookEventSessionEndName = "SessionEnd"

// SessionEndHookTimeoutMs mirrors TS SESSION_END_HOOK_TIMEOUT_MS_DEFAULT (1500ms).
// Overridable via CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS.
const SessionEndHookTimeoutMs = 1500

func getSessionEndHookTimeoutMs() int {
	raw := os.Getenv("CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS")
	if raw == "" {
		return SessionEndHookTimeoutMs
	}
	var ms int
	for _, c := range raw {
		if c < '0' || c > '9' {
			return SessionEndHookTimeoutMs
		}
		ms = ms*10 + int(c-'0')
	}
	if ms <= 0 {
		return SessionEndHookTimeoutMs
	}
	return ms
}

type sessionEndHookInput struct {
	BaseHookInput
	Reason string `json:"reason"`
}

// RunSessionEndHooks executes SessionEnd command hooks during shutdown/clear.
// Mirrors TS executeSessionEndHooks in src/utils/hooks.ts.
//
// SessionEnd hooks use a much tighter timeout (1500ms by default) since they
// run during shutdown. Hooks run in parallel; the single timeout value caps
// both per-hook and overall execution.
//
// After execution, session hooks are cleared via ClearSessionHooks.
func RunSessionEndHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	reason string,
	sessionID string,
) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return
	}

	in := sessionEndHookInput{
		BaseHookInput: base,
		Reason:        reason,
	}
	in.HookEventName = hookEventSessionEndName

	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return
	}

	var hookInput map[string]any
	if err := json.Unmarshal([]byte(jsonIn), &hookInput); err != nil {
		return
	}
	if len(CommandHooksForHookInput(table, hookInput)) == 0 {
		return
	}

	timeoutMs := getSessionEndHookTimeoutMs()
	wd := trimOrDot(workDir)
	results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
		Ctx:       ctx,
		WorkDir:   wd,
		Hooks:     table,
		JSONInput: jsonIn,
		TimeoutMs: timeoutMs,
	})

	for _, r := range results {
		if !r.Succeeded && strings.TrimSpace(r.Output) != "" {
			os.Stderr.WriteString("SessionEnd hook [" + r.Command + "] failed: " + strings.TrimSpace(r.Output) + "\n")
		}
	}

	// Clear session hooks after execution.
	if strings.TrimSpace(sessionID) != "" {
		ClearSessionHooks(sessionID)
	}
}
