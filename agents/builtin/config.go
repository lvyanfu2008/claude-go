package builtin

import (
	"os"
	"strings"

	"goc/commands/featuregates"
)

// Config carries runtime flags that mirror TS process.env + feature() + GrowthBook shims.
type Config struct {
	EmbeddedSearchTools bool // TS hasEmbeddedSearchTools(); Go: FEATURE_CHICAGO_MCP or CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS=1
	UserTypeAnt         bool // USER_TYPE=ant
	NonInteractive      bool // CLAUDE_CODE_NONINTERACTIVE=1 or HEADLESS=1 (SDK-style)
	Entrypoint          string
	Using3PServices     bool   // Bedrock/Vertex/Foundry — affects guide feedback line
	IssuesExplainer     string // replaces MACRO.ISSUES_EXPLAINER for 3P guide text
}

// ConfigFromEnv builds Config from environment (best-effort parity with TS bootstrap).
func ConfigFromEnv() Config {
	return Config{
		EmbeddedSearchTools: featuregates.Feature("CHICAGO_MCP") ||
			envTruthy("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS"),
		UserTypeAnt:     strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant",
		NonInteractive:  envTruthy("CLAUDE_CODE_NONINTERACTIVE") || envTruthy("HEADLESS"),
		Entrypoint:        strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT")),
		Using3PServices:   featuregates.IsUsing3PServicesFromEnv() || envUsing3P(),
		IssuesExplainer:   strings.TrimSpace(os.Getenv("CLAUDE_CODE_ISSUES_EXPLAINER")),
	}
}

func envUsing3P() bool {
	return envTruthy("CLAUDE_CODE_USE_BEDROCK") ||
		envTruthy("CLAUDE_CODE_USE_VERTEX") ||
		envTruthy("CLAUDE_CODE_USE_FOUNDRY")
}

func envTruthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(s)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// AreExplorePlanAgentsEnabled mirrors builtInAgents.ts areExplorePlanAgentsEnabled.
func AreExplorePlanAgentsEnabled() bool {
	if !featuregates.Feature("BUILTIN_EXPLORE_PLAN_AGENTS") {
		return false
	}
	// TS: getFeatureValue_CACHED_MAY_BE_STALE('tengu_amber_stoat', true) — default true
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TENGU_AMBER_STOAT")); v != "" {
		return envTruthy("CLAUDE_CODE_TENGU_AMBER_STOAT")
	}
	return true
}

func includeVerificationAgent() bool {
	if !featuregates.Feature("VERIFICATION_AGENT") {
		return false
	}
	return envTruthy("CLAUDE_CODE_TENGU_HIVE_EVIDENCE")
}

func disableAllBuiltinsForSDK(cfg Config) bool {
	return envTruthy("CLAUDE_AGENT_SDK_DISABLE_BUILTIN_AGENTS") && cfg.NonInteractive
}

func isNonSdkEntrypoint(entry string) bool {
	switch entry {
	case "sdk-ts", "sdk-py", "sdk-cli":
		return false
	default:
		return true
	}
}
