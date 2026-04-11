package querycontext

import (
	"context"
	"os"
	"strings"
	"sync"

	"goc/commands"
	"goc/modelenv"
	"goc/tscontext"
)

// FetchOpts mirrors fetchSystemPromptParts inputs from src/utils/queryContext.ts.
type FetchOpts struct {
	// CustomSystemPrompt when non-empty skips default system prompt assembly and system context (git + cache breaker), like TS.
	CustomSystemPrompt string

	// Gou builds the default system prompt when CustomSystemPrompt is empty (subset: commands.BuildGouDemoSystemPrompt).
	Gou commands.GouDemoSystemOpts

	// ExtraClaudeMdRoots is the Go equivalent of additional working dirs for CLAUDE.md when CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1.
	ExtraClaudeMdRoots []string

	// SystemPromptInjection optional override; nil falls back to CLAUDE_CODE_SYSTEM_PROMPT_INJECTION env.
	SystemPromptInjection *string

	// TSSnapshot when non-nil uses that snapshot for default/user/system prompt parts
	// (mirrors fetchSystemPromptParts) instead of building them in Go.
	TSSnapshot *tscontext.Snapshot
}

// FetchResult mirrors the Promise return type of fetchSystemPromptParts.
type FetchResult struct {
	DefaultSystemPrompt []string
	UserContext         map[string]string
	SystemContext       map[string]string
}

func useGoDefaultSystemInsteadOfTSSnapshot(opts FetchOpts) bool {
	if strings.TrimSpace(opts.Gou.EnvReportModelID) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID")) != "" {
		return true
	}
	return modelenv.FirstNonEmpty() != ""
}

func cloneStringSlice(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return append([]string(nil), s...)
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// FetchSystemPromptParts mirrors src/utils/queryContext.ts fetchSystemPromptParts (parallel fan-in).
func FetchSystemPromptParts(ctx context.Context, opts FetchOpts) (FetchResult, error) {
	custom := strings.TrimSpace(opts.CustomSystemPrompt)
	useCustom := custom != ""

	if snap := opts.TSSnapshot; snap != nil {
		if !useCustom {
			// Bun snapshot freezes # Environment for whatever model TS saw at bridge time.
			// When the process sets ANTHROPIC_MODEL / CCB_ENGINE_MODEL / ANTHROPIC_DEFAULT_* or
			// CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID (or Gou.EnvReportModelID), rebuild default system in Go
			// so the model line matches live env; keep TS userContext + systemContext slices.
			if useGoDefaultSystemInsteadOfTSSnapshot(opts) {
				s := strings.TrimSpace(commands.BuildGouDemoSystemPrompt(opts.Gou))
				return FetchResult{
					DefaultSystemPrompt: []string{s},
					UserContext:         cloneStringMap(snap.UserContext),
					SystemContext:       cloneStringMap(snap.SystemContext),
				}, nil
			}
			return FetchResult{
				DefaultSystemPrompt: cloneStringSlice(snap.DefaultSystemPrompt),
				UserContext:         cloneStringMap(snap.UserContext),
				SystemContext:       cloneStringMap(snap.SystemContext),
			}, nil
		}
		return FetchResult{
			DefaultSystemPrompt: []string{},
			UserContext:         cloneStringMap(snap.UserContext),
			SystemContext:       map[string]string{},
		}, nil
	}

	var (
		defaultParts []string
		userCtx      map[string]string
		sysCtx       map[string]string
		errUC        error
		mu           sync.Mutex
		wg           sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		if useCustom {
			return
		}
		s := commands.BuildGouDemoSystemPrompt(opts.Gou)
		mu.Lock()
		defaultParts = []string{strings.TrimSpace(s)}
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		uc, err := BuildUserContext(opts.Gou.Cwd, opts.ExtraClaudeMdRoots)
		mu.Lock()
		userCtx = uc
		errUC = err
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		if useCustom {
			return
		}
		sc := BuildSystemContext(ctx, opts.Gou.Cwd, opts.SystemPromptInjection)
		mu.Lock()
		sysCtx = sc
		mu.Unlock()
	}()

	wg.Wait()
	if errUC != nil {
		return FetchResult{}, errUC
	}
	if defaultParts == nil {
		defaultParts = []string{}
	}
	if userCtx == nil {
		userCtx = map[string]string{}
	}
	if sysCtx == nil {
		sysCtx = map[string]string{}
	}
	return FetchResult{
		DefaultSystemPrompt: defaultParts,
		UserContext:         userCtx,
		SystemContext:       sysCtx,
	}, nil
}
