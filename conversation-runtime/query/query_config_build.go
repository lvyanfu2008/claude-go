package query

import (
	"os"
	"strings"
)

// BuildQueryConfig mirrors buildQueryConfig at query.ts entry (subset: env + session id).
// TS also pulls GrowthBook / feature flags; those stay TS-only until ported.
func BuildQueryConfig() QueryConfig {
	sessionID := strings.TrimSpace(os.Getenv("GOU_DEMO_SESSION_ID"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(os.Getenv("CLAUDE_CODE_SESSION_ID"))
	}
	return QueryConfig{
		SessionID: sessionID,
		Gates: QueryConfigGates{
			StreamingToolExecution: envTruthy("GOU_DEMO_STREAMING_TOOL_EXECUTION"),
			EmitToolUseSummaries:   envTruthy("GOU_DEMO_EMIT_TOOL_USE_SUMMARIES"),
			IsAnt:                  envTruthy("ANT_ONLY") || strings.EqualFold(os.Getenv("CLAUDE_CODE_VENDOR"), "ant"),
			FastModeEnabled:        envTruthy("GOU_DEMO_FAST_MODE_ENABLED"),
		},
	}
}

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
