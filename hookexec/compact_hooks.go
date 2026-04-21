package hookexec

import (
	"context"
	"encoding/json"
	"strings"

	"goc/compactservice"
)

// PreCompactHookRunner wires settings-file PreCompact command hooks (TS executePreCompactHooks).
func PreCompactHookRunner(projectRoot, cwd, sessionID, transcriptPath string) compactservice.PreCompactHookRunner {
	return func(ctx context.Context, in compactservice.PreCompactHookInput) (compactservice.PreCompactHookResult, error) {
		root := strings.TrimSpace(projectRoot)
		table, err := MergedHooksFromPaths(root)
		if err != nil {
			return compactservice.PreCompactHookResult{}, err
		}
		sid := strings.TrimSpace(sessionID)
		if sid == "" {
			sid = "local"
		}
		wd := trimOrDot(cwd)
		var ci any
		if strings.TrimSpace(in.CustomInstructions) != "" {
			ci = in.CustomInstructions
		}
		payload := map[string]any{
			"session_id":          sid,
			"transcript_path":     strings.TrimSpace(transcriptPath),
			"cwd":                 wd,
			"hook_event_name":     "PreCompact",
			"trigger":             string(in.Trigger),
			"custom_instructions": ci,
		}
		jsonIn, err := json.Marshal(payload)
		if err != nil {
			return compactservice.PreCompactHookResult{}, err
		}
		var hookInput map[string]any
		if err := json.Unmarshal(jsonIn, &hookInput); err != nil {
			return compactservice.PreCompactHookResult{}, err
		}
		if len(CommandHooksForHookInput(table, hookInput)) == 0 {
			return compactservice.PreCompactHookResult{}, nil
		}
		results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
			Ctx:       ctx,
			WorkDir:   wd,
			Hooks:     table,
			JSONInput: string(jsonIn),
			TimeoutMs: DefaultHookTimeoutMs,
		})
		return aggregatePreCompactTS(results), nil
	}
}

// PostCompactHookRunner wires PostCompact command hooks (TS executePostCompactHooks).
func PostCompactHookRunner(projectRoot, cwd, sessionID, transcriptPath string) compactservice.PostCompactHookRunner {
	return func(ctx context.Context, in compactservice.PostCompactHookInput) (compactservice.PostCompactHookResult, error) {
		root := strings.TrimSpace(projectRoot)
		table, err := MergedHooksFromPaths(root)
		if err != nil {
			return compactservice.PostCompactHookResult{}, err
		}
		sid := strings.TrimSpace(sessionID)
		if sid == "" {
			sid = "local"
		}
		wd := trimOrDot(cwd)
		payload := map[string]any{
			"session_id":      sid,
			"transcript_path": strings.TrimSpace(transcriptPath),
			"cwd":             wd,
			"hook_event_name": "PostCompact",
			"trigger":         string(in.Trigger),
			"compact_summary": in.CompactSummary,
		}
		jsonIn, err := json.Marshal(payload)
		if err != nil {
			return compactservice.PostCompactHookResult{}, err
		}
		var hookInput map[string]any
		if err := json.Unmarshal(jsonIn, &hookInput); err != nil {
			return compactservice.PostCompactHookResult{}, err
		}
		if len(CommandHooksForHookInput(table, hookInput)) == 0 {
			return compactservice.PostCompactHookResult{}, nil
		}
		results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
			Ctx:       ctx,
			WorkDir:   wd,
			Hooks:     table,
			JSONInput: string(jsonIn),
			TimeoutMs: DefaultHookTimeoutMs,
		})
		return aggregatePostCompactTS(results), nil
	}
}

// aggregatePreCompactTS mirrors executePreCompactHooks aggregation in hooks.ts plus exit-code / JSON block → Blocked.
func aggregatePreCompactTS(results []OutsideReplCommandResult) compactservice.PreCompactHookResult {
	if len(results) == 0 {
		return compactservice.PreCompactHookResult{}
	}
	var successful []string
	var display []string
	var blocked bool
	for _, r := range results {
		if r.Blocked {
			blocked = true
		}
		if r.Succeeded && strings.TrimSpace(r.Output) != "" {
			successful = append(successful, strings.TrimSpace(r.Output))
		}
		if r.Succeeded {
			if strings.TrimSpace(r.Output) != "" {
				display = append(display, "PreCompact ["+r.Command+"] completed successfully: "+strings.TrimSpace(r.Output))
			} else {
				display = append(display, "PreCompact ["+r.Command+"] completed successfully")
			}
		} else {
			if strings.TrimSpace(r.Output) != "" {
				display = append(display, "PreCompact ["+r.Command+"] failed: "+strings.TrimSpace(r.Output))
			} else {
				display = append(display, "PreCompact ["+r.Command+"] failed")
			}
		}
	}
	out := compactservice.PreCompactHookResult{
		UserDisplayMessage: strings.Join(display, "\n"),
		Blocked:            blocked,
	}
	if len(successful) > 0 {
		out.NewCustomInstructions = strings.Join(successful, "\n\n")
	}
	return out
}

func aggregatePostCompactTS(results []OutsideReplCommandResult) compactservice.PostCompactHookResult {
	if len(results) == 0 {
		return compactservice.PostCompactHookResult{}
	}
	var display []string
	for _, r := range results {
		if r.Succeeded {
			if strings.TrimSpace(r.Output) != "" {
				display = append(display, "PostCompact ["+r.Command+"] completed successfully: "+strings.TrimSpace(r.Output))
			} else {
				display = append(display, "PostCompact ["+r.Command+"] completed successfully")
			}
		} else {
			if strings.TrimSpace(r.Output) != "" {
				display = append(display, "PostCompact ["+r.Command+"] failed: "+strings.TrimSpace(r.Output))
			} else {
				display = append(display, "PostCompact ["+r.Command+"] failed")
			}
		}
	}
	return compactservice.PostCompactHookResult{
		UserDisplayMessage: strings.Join(display, "\n"),
	}
}
