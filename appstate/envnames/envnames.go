// Package envnames centralizes CLAUDE_CODE_* env var name constants to reduce string duplication
// across the codebase. Only the most commonly referenced names are listed here; one-off usages
// may continue to use string literals.
package envnames

// DISABLE_ series — mirrors TS CLAUDE_CODE_DISABLE_* pattern for feature disablement.
const (
	DisableAutoMemory           = "CLAUDE_CODE_DISABLE_AUTO_MEMORY"
	DisableBackgroundTasks      = "CLAUDE_CODE_DISABLE_BACKGROUND_TASKS"
	DisableClaudeMds            = "CLAUDE_CODE_DISABLE_CLAUDE_MDS"
	DisableCron                 = "CLAUDE_CODE_DISABLE_CRON"
	DisableExperimentalBetas    = "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS"
	DisableFastMode             = "CLAUDE_CODE_DISABLE_FAST_MODE"
	DisableFileCheckpointing    = "CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING"
	DisableGitInstructions      = "CLAUDE_CODE_DISABLE_GIT_INSTRUCTIONS"
	DisableLocalGates           = "CLAUDE_CODE_DISABLE_LOCAL_GATES"
	DisableLocalMemory          = "CLAUDE_CODE_DISABLE_LOCAL_MEMORY"
	DisableMouse                = "CLAUDE_CODE_DISABLE_MOUSE"
	DisableNonessentialTraffic  = "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"
	DisablePolicySkills         = "CLAUDE_CODE_DISABLE_POLICY_SKILLS"
	DisableProjectMemory        = "CLAUDE_CODE_DISABLE_PROJECT_MEMORY"
	DisableSlashCommands        = "CLAUDE_CODE_DISABLE_SLASH_COMMANDS"
	DisableTerminalTitle        = "CLAUDE_CODE_DISABLE_TERMINAL_TITLE"
	DisableUserMemory           = "CLAUDE_CODE_DISABLE_USER_MEMORY"
	DisableVirtualScroll        = "CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL"
	Disable1MContext            = "CLAUDE_CODE_DISABLE_1M_CONTEXT"
)

// ENABLE_ series — mirrors TS CLAUDE_CODE_ENABLE_* pattern.
const (
	EnableFineGrainedToolStreaming = "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING"
	EnablePromptSuggestion         = "CLAUDE_CODE_ENABLE_PROMPT_SUGGESTION"
	EnableTasks                    = "CLAUDE_CODE_ENABLE_TASKS"
)

// Feature flags — used with featuregates.Feature() across the codebase.
const (
	FeatureAgentTriggersRemote  = "AGENT_TRIGGERS_REMOTE"
	FeatureBridgeMode           = "BRIDGE_MODE"
	FeatureBuddy                = "BUDDY"
	FeatureBuildingClaudeApps   = "BUILDING_CLAUDE_APPS"
	FeatureCachedMicrocompact   = "CACHED_MICROCOMPACT"
	FeatureCCRRemoteSetup       = "CCR_REMOTE_SETUP"
	FeatureChicagoMCP           = "CHICAGO_MCP"
	FeatureCoordinatorMode      = "COORDINATOR_MODE"
	FeatureCoralFern            = "CORAL_FERN"
	FeatureDaemon               = "DAEMON"
	FeatureExplorePlanAgents    = "BUILTIN_EXPLORE_PLAN_AGENTS"
	FeatureExperimentalSkill    = "EXPERIMENTAL_SKILL_SEARCH"
	FeatureKairos               = "KAIROS"
	FeatureKairosGitHubWebhooks = "KAIROS_GITHUB_WEBHOOKS"
	FeatureMCPSkills            = "MCP_SKILLS"
	FeatureMonitorTool          = "MONITOR_TOOL"
	FeatureMothCopse            = "MOTH_COPSE"
	FeatureOverflowTestTool     = "OVERFLOW_TEST_TOOL"
	FeatureReviewArtifact       = "REVIEW_ARTIFACT"
	FeatureRunSkillGenerator    = "RUN_SKILL_GENERATOR"
	FeatureTeamMem              = "TEAMMEM"
	FeatureTenguCobaltLantern   = "TENGU_COBALT_LANTERN"
	FeatureTerminalPanel        = "TERMINAL_PANEL"
	FeatureTokenBudget          = "TOKEN_BUDGET"
	FeatureTorch                = "TORCH"
	FeatureUDSInbox             = "UDS_INBOX"
	FeatureVerificationAgent    = "VERIFICATION_AGENT"
	FeatureVoiceMode            = "VOICE_MODE"
	FeatureWebBrowserTool       = "WEB_BROWSER_TOOL"
	FeatureWorkflowScripts      = "WORKFLOW_SCRIPTS"
)

// Session / operational — commonly referenced operational env vars.
const (
	DiscoverSkillsToolName  = "CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"
	PermissionMode          = "CLAUDE_CODE_PERMISSION_MODE"
	SessionID               = "CLAUDE_CODE_SESSION_ID"
	Thinking                = "CLAUDE_CODE_THINKING"
	GoAssume3P              = "CLAUDE_CODE_GO_ASSUME_3P"
	GoBundledChromeSkill    = "CLAUDE_CODE_GO_BUNDLED_CHROME_SKILL"
	GoEmbeddedSearchTools   = "CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS"
	GoExplorePlanAgents     = "CLAUDE_CODE_GO_EXPLORE_PLAN_AGENTS"
	GoMemorySearchPastCtx   = "CLAUDE_CODE_GO_MEMORY_SEARCH_PAST_CONTEXT"
	GoDisableMemorySkipIdx  = "CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX"
)
