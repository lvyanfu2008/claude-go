package querycontext

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Session-level memoization for user/system context and git status mirrors lodash memoize
// on getUserContext, getSystemContext, and getGitStatus in src/context.ts.
//
// Invalidation (TS parity):
//   - [ClearAllContextCaches] — src/commands/clear/caches.ts clearCaches
//   - [ClearUserAndSystemContextCaches] — setSystemPromptInjection (not git)
//   - [ClearUserContextCache] — post-compact cleanup (getUserContext.cache.clear only)

var sessionCtxMu sync.Mutex

type sessionCtxCaches struct {
	gitKey string
	gitVal string

	sysKey string
	sysVal map[string]string

	userKey string
	userVal map[string]string
	userErr error
}

var sessionCtx sessionCtxCaches

func canonicalCwd(cwd string) string {
	s := strings.TrimSpace(cwd)
	if s == "" {
		s, _ = os.Getwd()
	}
	abs, err := filepath.Abs(s)
	if err != nil {
		return filepath.Clean(s)
	}
	return filepath.Clean(abs)
}

func sortedExtraRootsKey(extra []string) string {
	if len(extra) == 0 {
		return ""
	}
	norm := make([]string, 0, len(extra))
	for _, e := range extra {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if a, err := filepath.Abs(e); err == nil {
			norm = append(norm, filepath.Clean(a))
		} else {
			norm = append(norm, filepath.Clean(e))
		}
	}
	sort.Strings(norm)
	return strings.Join(norm, "\x1e")
}

func userContextMemoKey(cwd string, extra []string) string {
	cw := canonicalCwd(cwd)
	ex := sortedExtraRootsKey(extra)
	dis := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISABLE_CLAUDE_MDS"))
	bareSkip := BareModeFromEnv() && len(extra) == 0
	// currentDate is intentionally excluded from the key — first successful result wins until clear (TS memo).
	return strings.Join([]string{cw, ex, dis, boolBit(bareSkip)}, "\x1f")
}

func systemContextMemoKey(cwd string, systemPromptInjection *string) string {
	cw := canonicalCwd(cwd)
	remote := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE"))
	gitOn := boolBit(!IsEnvTruthy(os.Getenv("CLAUDE_CODE_REMOTE")) && ShouldIncludeGitInstructions())
	inj := resolveSystemPromptInjection(systemPromptInjection)
	breakOn := boolBit(FeatureBreakCacheCommand() && strings.TrimSpace(inj) != "")
	return strings.Join([]string{cw, remote, gitOn, inj, breakOn}, "\x1f")
}

func boolBit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func cloneStrMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ClearAllContextCaches clears user context, system context, and git status caches
// (TS: getUserContext.cache.clear + getSystemContext.cache.clear + getGitStatus.cache.clear).
func ClearAllContextCaches() {
	sessionCtxMu.Lock()
	defer sessionCtxMu.Unlock()
	sessionCtx = sessionCtxCaches{}
}

// ClearUserAndSystemContextCaches clears user and system context caches but keeps the
// git snapshot memo (TS: setSystemPromptInjection).
func ClearUserAndSystemContextCaches() {
	sessionCtxMu.Lock()
	defer sessionCtxMu.Unlock()
	sessionCtx.userKey = ""
	sessionCtx.userVal = nil
	sessionCtx.userErr = nil
	sessionCtx.sysKey = ""
	sessionCtx.sysVal = nil
}

// ClearUserContextCache clears only the user context memo (TS: getUserContext.cache.clear
// from runPostCompactCleanup / compact paths).
func ClearUserContextCache() {
	sessionCtxMu.Lock()
	defer sessionCtxMu.Unlock()
	sessionCtx.userKey = ""
	sessionCtx.userVal = nil
	sessionCtx.userErr = nil
}

func gitStatusForSessionCache(ctx context.Context, cwd string) string {
	key := canonicalCwd(cwd)
	sessionCtxMu.Lock()
	if sessionCtx.gitKey == key {
		v := sessionCtx.gitVal
		sessionCtxMu.Unlock()
		return v
	}
	sessionCtxMu.Unlock()

	v := buildGitStatusSnapshotUncached(ctx, cwd)

	sessionCtxMu.Lock()
	sessionCtx.gitKey = key
	sessionCtx.gitVal = v
	sessionCtxMu.Unlock()
	return v
}

func userContextMemoized(cwd string, extraClaudeMdRoots []string) (map[string]string, error) {
	key := userContextMemoKey(cwd, extraClaudeMdRoots)

	sessionCtxMu.Lock()
	if sessionCtx.userKey == key {
		out := cloneStrMap(sessionCtx.userVal)
		err := sessionCtx.userErr
		sessionCtxMu.Unlock()
		return out, err
	}
	sessionCtxMu.Unlock()

	out, err := buildUserContextUncached(cwd, extraClaudeMdRoots)
	if err != nil {
		return nil, err
	}

	sessionCtxMu.Lock()
	sessionCtx.userKey = key
	sessionCtx.userVal = cloneStrMap(out)
	sessionCtx.userErr = nil
	sessionCtxMu.Unlock()

	return cloneStrMap(out), nil
}

func systemContextMemoized(ctx context.Context, cwd string, systemPromptInjection *string) map[string]string {
	key := systemContextMemoKey(cwd, systemPromptInjection)

	sessionCtxMu.Lock()
	if sessionCtx.sysKey == key {
		out := cloneStrMap(sessionCtx.sysVal)
		sessionCtxMu.Unlock()
		return out
	}
	sessionCtxMu.Unlock()

	out := buildSystemContextUncached(ctx, cwd, systemPromptInjection)

	sessionCtxMu.Lock()
	sessionCtx.sysKey = key
	sessionCtx.sysVal = cloneStrMap(out)
	sessionCtxMu.Unlock()

	return cloneStrMap(out)
}
