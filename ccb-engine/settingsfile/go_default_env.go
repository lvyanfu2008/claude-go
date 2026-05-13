package settingsfile

// GoProjectSettingsEnvDefaults returns the baseline "env" map for Go binaries, merged
// first in [ApplyMergedClaudeSettingsEnv] (lowest precedence for duplicate keys: user
// and project files override; process environment still wins first via
// [applyEnvMapSkipExisting]).
func GoProjectSettingsEnvDefaults() map[string]string {
	return map[string]string{
		"FEATURE_WORKFLOW_SCRIPTS":                 "0",
		"USER_TYPE":                                "lvyanfu",
		"CLAUDE_CODE_USE_OPENAI":                   "1",
		"CLAUDE_CODE_DEBUG_LOGS_DIR":               "debug/goc",
		"CLAUDE_CODE_LOG_API_REQUEST_BODY":         "0",
		"CLAUDE_CODE_LOG_API_RESPONSE_BODY":        "0",
		"GOU_DEMO_LOG":                             "1",
		"GOU_DEMO_USE_EMBEDDED_TOOLS_API":          "1",
		"FEATURE_MCP_SKILLS":                       "1",
		"CLAUDE_CODE_LOG_TOOL_USE_CONTEXT":         "summary",
		"FEATURE_BUDDY":                            "1",
		"CLAUDE_CODE_GO_TOOL_SEARCH_CONTEXT":       "0",
		"GOU_TOOLEXEC_BASH_SANDBOX_1B":             "1",
		"GO_TOOL_INPUT_VALIDATOR":                  "zog",
		"GOU_QUERY_STREAMING_FORCE_ANTHROPIC":      "0",
		"GOU_QUERY_OPENAI_CHAT_NO_STREAM":          "1",
		"GOU_DEMO_DISALLOW_DISABLE_MOUSE":          "1",
		"GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS":       "1000",
		"GOU_DEMO_ALT_SCREEN":                      "1",
		"GOU_DEMO_DUMP_ON_EXIT":                    "1",
		"GOC_AUTOCOMPACT_MAX_CONTEXT_WINDOW":       "100000",
		"FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS":      "1",
		"CLAUDE_CODE_TENGU_HIVE_EVIDENCE":          "1",
		"CLAUDE_CODE_TENGU_CORAL_FERN":             "1",
		"FEATURE_TOKEN_BUDGET":                     "1",
		"FEATURE_VERIFICATION_AGENT":               "1",
		"FEATURE_MONITOR_TOOL":                     "1",
		"FEATURE_REVIEW_ARTIFACT":                  "0",
		"FEATURE_AGENT_TRIGGERS_REMOTE":            "0",
		"FEATURE_BUILDING_CLAUDE_APPS":             "0",
		"FEATURE_CHICAGO_MCP":                      "0",
		"FEATURE_RUN_SKILL_GENERATOR":              "0",
		"CLAUDE_CODE_DISABLE_FAST_MODE":            "1",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
		"CLAUDE_CODE_SIMPLE":                       "0",
		"CLAUDE_DEBUG_PROCESS_USER_INPUT":          "0",
		"CLAUDE_CODE_GO_DEBUG_AGENT_TOOL_SCHEMA":   "1",
		"GOU_DEMO_NO_ASK_AUTO_FIRST":               "1",
		"GOC_EXTRACT_MEMORIES_LOG_FILE":            "/Users/lvyanfu/.cache/claude/extract-memories.log",
		"GOC_EXTRACT_MEMORIES_RELAX_THRESHOLD":     "0",
		"CLAUDE_CODE_GO_DEBUG_SYSTEM_PROMPT":       "/Users/lvyanfu/.cache/claude/system_prompt_debug.log",
	}
}
