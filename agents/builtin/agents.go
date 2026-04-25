package builtin

import "encoding/json"

// BuiltinAgent is a resolved built-in definition (system prompt expanded; mirrors AgentDefinition fields used by TS).
type BuiltinAgent struct {
	AgentType string `json:"agentType"`
	WhenToUse string `json:"whenToUse"`
	// Tools: nil or absent in JSON means wildcard (all tools except disallowed), matching TS resolveAgentTools.
	Tools           []string `json:"tools,omitempty"`
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	Source          string   `json:"source"`
	BaseDir         string   `json:"baseDir"`
	Model           string   `json:"model,omitempty"`
	Color           string   `json:"color,omitempty"`
	PermissionMode  string   `json:"permissionMode,omitempty"`
	Background      bool     `json:"background,omitempty"`
	OmitClaudeMd    bool     `json:"omitClaudeMd,omitempty"`
	// CriticalSystemReminderExperimental matches TS criticalSystemReminder_EXPERIMENTAL.
	CriticalSystemReminderExperimental string `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"`
	SystemPrompt                       string `json:"systemPrompt"`
	// Hooks mirrors TS frontmatter hooks field; nil for built-in agents.
	Hooks json.RawMessage `json:"hooks,omitempty"`
}

const (
	generalPurposeWhenToUse = `General-purpose agent for researching complex questions, searching for code, and executing multi-step tasks. When you are searching for a keyword or file and are not confident that you will find the right match in the first few tries use this agent to perform the search for you.`

	statuslineWhenToUse = `Use this agent to configure the user's Claude Code status line setting.`

	exploreWhenToUse = `Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. "src/components/**/*.tsx"), search code for keywords (eg. "API endpoints"), or answer questions about the codebase (eg. "how do API endpoints work?"). When calling this agent, specify the desired thoroughness level: "quick" for basic searches, "medium" for moderate exploration, or "very thorough" for comprehensive analysis across multiple locations and naming conventions.`

	planWhenToUse = `Software architect agent for designing implementation plans. Use this when you need to plan the implementation strategy for a task. Returns step-by-step plans, identifies critical files, and considers architectural trade-offs.`

	verificationWhenToUse = `Use this agent to verify that implementation work is correct before reporting completion. Invoke after non-trivial tasks (3+ file edits, backend/API changes, infrastructure changes). Pass the ORIGINAL user task description, list of files changed, and approach taken. The agent runs builds, tests, linters, and checks to produce a PASS/FAIL/PARTIAL verdict with evidence.`

	verificationCriticalReminder = `CRITICAL: This is a VERIFICATION-ONLY task. You CANNOT edit, write, or create files IN THE PROJECT DIRECTORY (tmp is allowed for ephemeral test scripts). You MUST end with VERDICT: PASS, VERDICT: FAIL, or VERDICT: PARTIAL.`
)

func claudeCodeGuideWhenToUse() string {
	return `Use this agent when the user asks questions ("Can Claude...", "Does Claude...", "How do I...") about: (1) Claude Code (the CLI tool) - features, hooks, slash commands, MCP servers, settings, IDE integrations, keyboard shortcuts; (2) Claude Agent SDK - building custom agents; (3) Claude API (formerly Anthropic API) - API usage, tool use, Anthropic SDK usage. **IMPORTANT:** Before spawning a new agent, check if there is already a running or recently completed claude-code-guide agent that you can continue via ` + ToolSendMessage + `.`
}

var explorePlanDisallowedTools = []string{
	ToolAgent,
	ToolExitPlanMode,
	ToolEdit,
	ToolWrite,
	ToolNotebookEdit,
}

// GetBuiltInAgents mirrors claude-code getBuiltInAgents() (coordinator branch omitted in Go).
func GetBuiltInAgents(cfg Config, guideCtx GuideContext) []BuiltinAgent {
	if disableAllBuiltinsForSDK(cfg) {
		return nil
	}
	out := make([]BuiltinAgent, 0, 8)

	out = append(out, BuiltinAgent{
		AgentType:    "general-purpose",
		WhenToUse:    generalPurposeWhenToUse,
		Tools:        []string{"*"},
		Source:       "built-in",
		BaseDir:      "built-in",
		SystemPrompt: generalPurposeSystemPrompt(),
	})

	out = append(out, BuiltinAgent{
		AgentType:    "statusline-setup",
		WhenToUse:    statuslineWhenToUse,
		Tools:        []string{ToolRead, ToolEdit},
		Source:       "built-in",
		BaseDir:      "built-in",
		Model:        "sonnet",
		Color:        "orange",
		SystemPrompt: StatuslineSystemPrompt(),
	})

	if AreExplorePlanAgentsEnabled() {
		exploreModel := "haiku"
		if cfg.UserTypeAnt {
			exploreModel = "inherit"
		}
		out = append(out, BuiltinAgent{
			AgentType:       "Explore",
			WhenToUse:       exploreWhenToUse,
			DisallowedTools: explorePlanDisallowedTools,
			Source:          "built-in",
			BaseDir:         "built-in",
			Model:           exploreModel,
			OmitClaudeMd:    true,
			SystemPrompt:    exploreSystemPrompt(cfg.EmbeddedSearchTools),
		})
		out = append(out, BuiltinAgent{
			AgentType:       "Plan",
			WhenToUse:       planWhenToUse,
			DisallowedTools: explorePlanDisallowedTools,
			Source:          "built-in",
			BaseDir:         "built-in",
			Model:           "inherit",
			OmitClaudeMd:    true,
			SystemPrompt:    planSystemPrompt(cfg.EmbeddedSearchTools),
		})
	}

	if isNonSdkEntrypoint(cfg.Entrypoint) {
		out = append(out, BuiltinAgent{
			AgentType:      "claude-code-guide",
			WhenToUse:      claudeCodeGuideWhenToUse(),
			Tools:          guideTools(cfg.EmbeddedSearchTools),
			Source:         "built-in",
			BaseDir:        "built-in",
			Model:          "haiku",
			PermissionMode: "dontAsk",
			SystemPrompt:   ClaudeCodeGuideSystemPrompt(cfg, guideCtx),
		})
	}

	if includeVerificationAgent() {
		out = append(out, BuiltinAgent{
			AgentType: "verification",
			WhenToUse: verificationWhenToUse,
			DisallowedTools: []string{
				ToolAgent,
				ToolExitPlanMode,
				ToolEdit,
				ToolWrite,
				ToolNotebookEdit,
			},
			Source:                             "built-in",
			BaseDir:                            "built-in",
			Model:                              "inherit",
			Color:                              "red",
			Background:                         true,
			CriticalSystemReminderExperimental: verificationCriticalReminder,
			SystemPrompt:                       VerificationSystemPrompt(),
		})
	}

	return out
}
