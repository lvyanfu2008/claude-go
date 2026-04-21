package hookexec

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// OutsideReplCommandResult mirrors a single command outcome from TS executeHooksOutsideREPL (command branch).
type OutsideReplCommandResult struct {
	Command   string
	Succeeded bool
	Output    string // stdout and stderr joined (same as historical behavior for compact aggregation).
	Stdout    string
	Stderr    string
	ExitCode  int
	DurationMs int64
	Blocked   bool
}

// OutsideReplCommandParams is input for parallel command-hook execution (TS executeHooksOutsideREPL user hook batch).
type OutsideReplCommandParams struct {
	Ctx       context.Context
	WorkDir   string
	Hooks     HooksTable
	JSONInput string // full hook stdin JSON (validated by caller)
	TimeoutMs int    // per-hook cap when hook has no timeout; 0 → DefaultHookTimeoutMs
}

// ExecuteCommandHooksOutsideREPLParallel runs all matching **command** hooks concurrently (TS Promise.all on matchingHooks).
// Non-command hook types are skipped here (prompt/agent/http/callback/function — see TS executeHooksOutsideREPL).
func ExecuteCommandHooksOutsideREPLParallel(p OutsideReplCommandParams) []OutsideReplCommandResult {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil
	}
	var hookInput map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(p.JSONInput)), &hookInput); err != nil {
		return nil
	}
	hooks := CommandHooksForHookInput(p.Hooks, hookInput)
	if len(hooks) == 0 {
		return nil
	}
	wd := trimOrDot(p.WorkDir)
	batch := p.TimeoutMs
	if batch <= 0 {
		batch = DefaultHookTimeoutMs
	}
	ctx := p.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	out := make([]OutsideReplCommandResult, len(hooks))
	var wg sync.WaitGroup
	for i := range hooks {
		i := i
		h := hooks[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			ms := hookTimeoutMS(h, batch)
			start := time.Now()
			stdout, stderr, exitCode, err := RunCommandHook(ctx, wd, h.Command, strings.TrimSpace(p.JSONInput), ms)
			res := OutsideReplCommandResult{
				Command:    h.Command,
				Output:     stdout,
				Stdout:     stdout,
				Stderr:     stderr,
				ExitCode:   exitCode,
				DurationMs: time.Since(start).Milliseconds(),
			}
			if stderr != "" {
				if res.Output != "" {
					res.Output += "\n"
				}
				res.Output += stderr
			}
			// TS: succeeded = result.status === 0; stderr used for failed hooks' output.
			if err != nil {
				if res.Output == "" {
					res.Output = err.Error()
				}
			}
			res.Succeeded = err == nil && exitCode == 0
			jsonBlocked, _ := parseHookBlocked(stdout)
			res.Blocked = exitCode == 2 || jsonBlocked
			out[i] = res
		}()
	}
	wg.Wait()
	return out
}

func parseHookBlocked(stdout string) (bool, error) {
	s := strings.TrimSpace(stdout)
	if s == "" {
		return false, nil
	}
	var top map[string]any
	if err := json.Unmarshal([]byte(s), &top); err != nil {
		return false, err
	}
	if dec, ok := top["decision"].(string); ok && dec == "block" {
		return true, nil
	}
	return false, nil
}
