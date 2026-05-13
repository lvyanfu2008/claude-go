package hookexec

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"goc/tools/hookstypes"
)

// OutsideReplCommandResult mirrors a single command outcome from TS executeHooksOutsideREPL (command branch).
type OutsideReplCommandResult struct {
	Command   string
	Succeeded bool
	Output    string // stdout and stderr joined.
	Stdout    string
	Stderr    string
	ExitCode  int
	DurationMs int64
	Blocked   bool
}

// OutsideReplHookResult is a unified result for any hook type (command, prompt, http, agent).
type OutsideReplHookResult struct {
	HookType     string // "command", "prompt", "http", "agent"
	Command      string // command string (for display / trace)
	Succeeded    bool
	Output       string
	Stdout       string
	Stderr       string
	ExitCode     int
	DurationMs   int64
	Blocked      bool
	JSONParsed   *syncHookStdoutParsed // for prompt/http/agent typed JSON outputs
	Cancelled    bool
	HTTPStatusCode int
	ErrorMessage  string
}

// HookExecDeps bundles injectable dependencies for non-command hook types.
// When nil, prompt/http/agent hooks are skipped with a warning message.
type HookExecDeps struct {
	PromptHookDeps PromptHookDeps
	AgentHookDeps  AgentHookDeps
	// HttpAllowedURLs mirrors TS allowedHttpHookUrls setting.
	HttpAllowedURLs []string
}

// OutsideReplCommandParams is input for parallel hook execution.
type OutsideReplCommandParams struct {
	Ctx       context.Context
	WorkDir   string
	Hooks     HooksTable
	JSONInput string
	TimeoutMs int
	// Deps enables non-command hook types (prompt, http, agent).
	// When nil, only command hooks are executed.
	Deps *HookExecDeps
}

// ExecuteCommandHooksOutsideREPLParallel runs all matching **command** hooks concurrently.
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

// ExecuteAllHooksOutsideREPLParallel runs all matching hooks of any type concurrently.
// TS parity: executeHooksOutsideREPL in hooks.ts (all hook types).
//
// For command hooks, uses RunCommandHook. For prompt, http, and agent hooks, uses the respective executors.
// When the executor for a non-command type is not available, the hook is skipped with a warning.
func ExecuteAllHooksOutsideREPLParallel(p OutsideReplCommandParams) []OutsideReplHookResult {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil
	}
	var hookInput map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(p.JSONInput)), &hookInput); err != nil {
		return nil
	}
	hooks := AllHooksForHookInput(p.Hooks, hookInput)
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
	out := make([]OutsideReplHookResult, len(hooks))
	var wg sync.WaitGroup
	for i := range hooks {
		i := i
		h := hooks[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			ms := hookCommandTimeoutMS(h, batch)
			start := time.Now()
			res := OutsideReplHookResult{
				HookType:   strings.TrimSpace(h.Type),
				Command:    displayNameForHook(h),
				DurationMs: time.Since(start).Milliseconds(),
			}
			switch strings.TrimSpace(h.Type) {
			case "command":
				stdout, stderr, exitCode, err := RunCommandHook(ctx, wd, h.Command, strings.TrimSpace(p.JSONInput), ms)
				res.Stdout = stdout
				res.Stderr = stderr
				res.ExitCode = exitCode
				res.Output = stdout
				if stderr != "" {
					if res.Output != "" {
						res.Output += "\n"
					}
					res.Output += stderr
				}
				if err != nil {
					if res.Output == "" {
						res.Output = err.Error()
					}
				}
				res.Succeeded = err == nil && exitCode == 0
				jsonBlocked, _ := parseHookBlocked(stdout)
				res.Blocked = exitCode == 2 || jsonBlocked
			case "prompt":
				if p.Deps != nil {
					pr := RunPromptHook(ctx, h, strings.TrimSpace(p.JSONInput), p.Deps.PromptHookDeps)
					pr.DurationMs = 0 // will be set below
					res = pr
				} else {
					res.ErrorMessage = "prompt hooks not configured (no deps)"
				}
			case "http":
				allowed := []string(nil)
				if p.Deps != nil {
					allowed = p.Deps.HttpAllowedURLs
				}
				res = RunHttpHook(ctx, p.WorkDir, h, strings.TrimSpace(p.JSONInput), allowed)
				res.DurationMs = 0 // will be set below
			case "agent":
				if p.Deps != nil {
					ar := RunAgentHook(ctx, h, strings.TrimSpace(p.JSONInput), p.Deps.AgentHookDeps)
					ar.DurationMs = 0 // will be set below
					res = ar
				} else {
					res.ErrorMessage = "agent hooks not configured (no deps)"
				}
			default:
				res.ErrorMessage = "unknown hook type: " + h.Type
			}
			res.DurationMs = time.Since(start).Milliseconds()
			out[i] = res
		}()
	}
	wg.Wait()
	return out
}

func hookCommandTimeoutMS(h hookstypes.HookCommand, batchDefault int) int {
	if h.Timeout > 0 {
		ms := int(h.Timeout * 1000)
		if ms > 30*60*1000 {
			return 30 * 60 * 1000
		}
		if ms < 1000 {
			return 1000
		}
		return ms
	}
	if batchDefault > 0 {
		return batchDefault
	}
	return DefaultHookTimeoutMs
}

func displayNameForHook(h hookstypes.HookCommand) string {
	switch strings.TrimSpace(h.Type) {
	case "command":
		return h.Command
	case "prompt":
		return "[prompt] " + truncateString(h.Prompt, 80)
	case "http":
		return "[http] " + h.URL
	case "agent":
		return "[agent] " + truncateString(h.Prompt, 80)
	default:
		return h.Type
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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
