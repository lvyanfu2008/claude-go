package querycontext

import (
	"context"
	"os"
	"strings"
	"sync"

	"goc/commands"
	"goc/hookexec"
	"goc/modelenv"
	"goc/tscontext"
	"goc/types"
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

	// TSSnapshot when non-nil uses the snapshot for default/system prompt slices (Go harness only).
	// UserContext always comes from [BuildUserContext] like TS getUserContext() in fetchSystemPromptParts — never from snap.UserContext.
	TSSnapshot *tscontext.Snapshot

	// SessionStartSource when non-empty (startup|resume|clear|compact) runs SessionStart command hooks like TS processSessionStartHooks.
	SessionStartSource string
	// HooksSessionID / HooksTranscriptPath feed BaseHookInput for hook stdin JSON (optional).
	HooksSessionID      string
	HooksTranscriptPath string
}

// FetchResult mirrors the Promise return type of fetchSystemPromptParts.
type FetchResult struct {
	DefaultSystemPrompt []string
	UserContext         map[string]string
	SystemContext       map[string]string
	// SessionStartHookMessages holds hook_additional_context attachment rows from SessionStart hooks (optional).
	SessionStartHookMessages []types.Message `json:"-"`
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

// userContextLikeTS is the Go equivalent of TS getUserContext() in src/utils/queryContext.ts
// fetchSystemPromptParts: always live from cwd/CLAUDE.md, never merged from tscontext.Snapshot.
func userContextLikeTS(opts FetchOpts) (map[string]string, error) {
	return BuildUserContext(opts.Gou.Cwd, opts.ExtraClaudeMdRoots)
}

func sessionStartHookMessages(ctx context.Context, opts FetchOpts) ([]types.Message, error) {
	src := strings.TrimSpace(opts.SessionStartSource)
	if src == "" {
		return nil, nil
	}
	cwd := strings.TrimSpace(opts.Gou.Cwd)
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}
	tab, err := hookexec.MergedHooksForCwd(cwd)
	if err != nil {
		return nil, err
	}
	sid := strings.TrimSpace(opts.HooksSessionID)
	if sid == "" {
		sid = "local"
	}
	base := hookexec.BaseHookInput{
		SessionID:      sid,
		TranscriptPath: strings.TrimSpace(opts.HooksTranscriptPath),
		Cwd:            cwd,
	}
	return hookexec.RunSessionStartHooks(ctx, tab, cwd, base, hookexec.SessionStartExtra{
		Source: src,
		Model:  strings.TrimSpace(opts.Gou.ModelID),
	}, hookexec.DefaultHookTimeoutMs)
}

// FetchSystemPromptParts mirrors src/utils/queryContext.ts fetchSystemPromptParts (parallel fan-in).
func FetchSystemPromptParts(ctx context.Context, opts FetchOpts) (FetchResult, error) {
	custom := strings.TrimSpace(opts.CustomSystemPrompt)
	useCustom := custom != ""

	if snap := opts.TSSnapshot; snap != nil {
		uc, errUC := userContextLikeTS(opts)
		if errUC != nil {
			return FetchResult{}, errUC
		}
		if !useCustom {
			// Bun snapshot freezes # Environment for whatever model TS saw at bridge time.
			// When the process sets ANTHROPIC_MODEL / CCB_ENGINE_MODEL / ANTHROPIC_DEFAULT_* or
			// CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID (or Gou.EnvReportModelID), rebuild default system in Go
			// so the model line matches live env. UserContext is always live (getUserContext parity).
			if useGoDefaultSystemInsteadOfTSSnapshot(opts) {
				s := strings.TrimSpace(commands.BuildGouDemoSystemPrompt(opts.Gou))
				res := FetchResult{
					DefaultSystemPrompt: []string{s},
					UserContext:         uc,
					SystemContext:       cloneStringMap(snap.SystemContext),
				}
				ss, errSS := sessionStartHookMessages(ctx, opts)
				if errSS != nil {
					return FetchResult{}, errSS
				}
				res.SessionStartHookMessages = ss
				return res, nil
			}
			res := FetchResult{
				DefaultSystemPrompt: cloneStringSlice(snap.DefaultSystemPrompt),
				UserContext:         uc,
				SystemContext:       cloneStringMap(snap.SystemContext),
			}
			ss, errSS := sessionStartHookMessages(ctx, opts)
			if errSS != nil {
				return FetchResult{}, errSS
			}
			res.SessionStartHookMessages = ss
			return res, nil
		}
		// TS: customSystemPrompt skips default system + getSystemContext, but getUserContext() still runs.
		res := FetchResult{
			DefaultSystemPrompt: []string{},
			UserContext:         uc,
			SystemContext:       map[string]string{},
		}
		ss, errSS := sessionStartHookMessages(ctx, opts)
		if errSS != nil {
			return FetchResult{}, errSS
		}
		res.SessionStartHookMessages = ss
		return res, nil
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
	res := FetchResult{
		DefaultSystemPrompt: defaultParts,
		UserContext:         userCtx,
		SystemContext:       sysCtx,
	}
	ss, errSS := sessionStartHookMessages(ctx, opts)
	if errSS != nil {
		return FetchResult{}, errSS
	}
	res.SessionStartHookMessages = ss
	return res, nil
}
