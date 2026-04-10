package querycontext

import (
	"context"
	"os"
	"strings"
)

// BuildSystemContext mirrors src/context.ts getSystemContext (no memoization).
func BuildSystemContext(ctx context.Context, cwd string, systemPromptInjection *string) map[string]string {
	out := map[string]string{}

	var gitStatus string
	if !IsEnvTruthy(os.Getenv("CLAUDE_CODE_REMOTE")) && ShouldIncludeGitInstructions() {
		gitStatus = BuildGitStatusSnapshot(ctx, cwd)
	}
	if strings.TrimSpace(gitStatus) != "" {
		out["gitStatus"] = gitStatus
	}

	injection := resolveSystemPromptInjection(systemPromptInjection)
	if FeatureBreakCacheCommand() && strings.TrimSpace(injection) != "" {
		out["cacheBreaker"] = `[CACHE_BREAKER: ` + injection + `]`
	}
	return out
}

func resolveSystemPromptInjection(explicit *string) string {
	if explicit != nil {
		return strings.TrimSpace(*explicit)
	}
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_SYSTEM_PROMPT_INJECTION"))
}
