package commands

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"goc/commands/featuregates"
	"goc/modelenv"
	"goc/types"
)

// GouDemoSystemOpts drives phase-2 system string assembly (mirrors TS getSystemPrompt; see prompts.go for constants/helpers).
type GouDemoSystemOpts struct {
	EnabledToolNames  map[string]struct{}
	SkillToolCommands []types.Command // merged skill listing slice (TS getSkillListingAttachments sources)
	ModelID           string
	// EnvReportModelID overrides ModelID only in # Environment model line + knowledge cutoff (TS session modelId vs API id).
	EnvReportModelID       string
	Cwd                    string
	Language               string // TS settings.language
	OutputStyleName        string
	OutputStylePrompt      string
	DiscoverSkillsToolName string // from CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME when registered
	NonInteractiveSession  bool
	// EmbeddedSearchTools mirrors hasEmbeddedSearchTools() — omits Glob/Grep bullets in # Using your tools and changes explore search hint.
	EmbeddedSearchTools bool
	// UserTypeAnt mirrors USER_TYPE=ant (tone, output efficiency, optional numeric length anchors).
	UserTypeAnt bool
	// ReplModeEnabled mirrors isReplModeEnabled() — short # Using your tools (REPL path).
	ReplModeEnabled bool
	// ExplorePlanAgentsEnabled mirrors areExplorePlanAgentsEnabled() when Agent is enabled.
	ExplorePlanAgentsEnabled bool
	// VerificationAgentGuidance mirrors feature VERIFICATION_AGENT + GrowthBook; Go enables via FEATURE_VERIFICATION_AGENT + CLAUDE_CODE_GO_VERIFICATION_AGENT_GUIDANCE=1.
	VerificationAgentGuidance bool
	// ScratchpadDir when non-empty appends # Scratchpad Directory (TS getScratchpadInstructions shape).
	ScratchpadDir string
	// MemorySkipIndex mirrors tengu_moth_copse (buildMemoryLines skipIndex) — shorter "How to save" without MEMORY.md index step.
	MemorySkipIndex bool
	// MemorySearchPastContext mirrors tengu_coral_fern buildSearchingPastContextSection.
	MemorySearchPastContext bool
	// KairosActive mirrors STATE.kairosActive / getKairosActive — set CLAUDE_CODE_GO_KAIROS_ACTIVE=1 when FEATURE_KAIROS.
	KairosActive bool
	// UserMsgOptIn mirrors getUserMsgOptIn — Brief / SendUserMessage opt-in (CLAUDE_CODE_GO_USER_MSG_OPT_IN=1).
	UserMsgOptIn bool
	// CoordinatorMode mirrors isCoordinatorMode() — disables fork subagent when true.
	CoordinatorMode bool
	// ParityGOOS and ParityGOARCH when both non-empty override runtime.GOOS/GOARCH in # Environment
	// so API parity snapshots and golden hashes are stable across macOS/Linux/ARM.
	ParityGOOS   string
	ParityGOARCH string
	// AdditionalWorkingDirs mirrors additionalWorkingDirectories in computeSimpleEnvInfo.
	AdditionalWorkingDirs []string
	// Undercover mirrors isUndercover() — strips model marketing lines from env when true with USER_TYPE=ant.
	Undercover bool
	// KeepCodingInstructions when false and OutputStyleName is non-empty omits # Doing tasks (TS keepCodingInstructions).
	// ApplyGouDemoRuntimeEnv sets this to true; set false after Apply when settings disable coding instructions.
	KeepCodingInstructions bool
	// AntModelOverrideSuffix mirrors getAntModelOverrideConfig()?.defaultSystemPromptSuffix (non-interactive / ant).
	AntModelOverrideSuffix string
	// MCPInstructionsMarkdown mirrors getMcpInstructionsSection when delta is off.
	MCPInstructionsMarkdown string
	// FunctionResultClearingMarkdown mirrors getFunctionResultClearingSection body when enabled.
	FunctionResultClearingMarkdown string
	// BriefSectionMarkdown mirrors getBriefSection when KAIROS/BRIEF gates pass.
	BriefSectionMarkdown string
	// PromptIsGitRepo / PromptGitWorktree when nil are filled via PromptGitHints(Cwd) unless SkipPromptGitDetect.
	SkipPromptGitDetect bool
	PromptIsGitRepo     *bool
	PromptGitWorktree   *bool
}

func envTruthyGo(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// EnvModelForSystemPrompt returns the model id used in ComputeSimpleEnvInfo / ComputeEnvInfo (TS computeSimpleEnvInfo modelId).
// Priority: EnvReportModelID → CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID → same process-env chain as HTTP
// ([modelenv.FirstNonEmpty]: CCB_ENGINE_MODEL, ANTHROPIC_MODEL, ANTHROPIC_DEFAULT_* — includes values
// applied from ~/.claude/settings.json and project .claude/settings.go.json when the process env was empty) → ModelID.
func (o GouDemoSystemOpts) EnvModelForSystemPrompt() string {
	if v := strings.TrimSpace(o.EnvReportModelID); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID")); v != "" {
		return v
	}
	if v := modelenv.FirstNonEmpty(); v != "" {
		return v
	}
	return strings.TrimSpace(o.ModelID)
}

// GouDemoReplModeFromEnv mirrors src/tools/REPLTool/constants.ts isReplModeEnabled (approximation for gou-demo).
func GouDemoReplModeFromEnv() bool {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REPL")); v == "0" || strings.EqualFold(v, "false") {
		return false
	}
	if envTruthyGo("CLAUDE_REPL_MODE") {
		return true
	}
	return featuregates.UserTypeAnt() && strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENTRYPOINT")) == "cli"
}

// GouDemoExplorePlanAgentsFromEnv mirrors areExplorePlanAgentsEnabled when FEATURE_BUILTIN_EXPLORE_PLAN_AGENTS is on (GrowthBook default true → opt out with CLAUDE_CODE_GO_EXPLORE_PLAN_AGENTS=0).
func GouDemoExplorePlanAgentsFromEnv() bool {
	if !featuregates.Feature("BUILTIN_EXPLORE_PLAN_AGENTS") {
		return false
	}
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_GO_EXPLORE_PLAN_AGENTS"))); v == "0" || v == "false" {
		return false
	}
	return true
}

// ApplyGouDemoRuntimeEnv mutates o with TS-parity shims from environment (embedded search, ant, REPL, explore plan, verification guidance, memory flags, kairos active).
func ApplyGouDemoRuntimeEnv(o *GouDemoSystemOpts) {
	if o == nil {
		return
	}
	o.EmbeddedSearchTools = featuregates.Feature("CHICAGO_MCP") || envTruthyGo("CLAUDE_CODE_GO_EMBEDDED_SEARCH_TOOLS")
	o.UserTypeAnt = featuregates.UserTypeAnt()
	o.ReplModeEnabled = GouDemoReplModeFromEnv()
	o.ExplorePlanAgentsEnabled = GouDemoExplorePlanAgentsFromEnv()
	o.VerificationAgentGuidance = featuregates.Feature("VERIFICATION_AGENT") && envTruthyGo("CLAUDE_CODE_GO_VERIFICATION_AGENT_GUIDANCE")
	o.MemorySkipIndex = featuregates.Feature("MOTH_COPSE")
	o.MemorySearchPastContext = featuregates.Feature("CORAL_FERN") || envTruthyGo("CLAUDE_CODE_GO_MEMORY_SEARCH_PAST_CONTEXT")
	o.KairosActive = envTruthyGo("CLAUDE_CODE_GO_KAIROS_ACTIVE")
	o.UserMsgOptIn = envTruthyGo("CLAUDE_CODE_GO_USER_MSG_OPT_IN")
	o.CoordinatorMode = envTruthyGo("CLAUDE_CODE_GO_COORDINATOR_MODE")
	o.Undercover = envTruthyGo("CLAUDE_CODE_GO_UNDERCOVER")
	o.KeepCodingInstructions = true
}

// issuesExplainerDoingTasks mirrors TS prompts.ts MACRO.ISSUES_EXPLAINER (build-time in claude-code; empty in scripts/defines.ts would leave an incomplete sentence).
const issuesExplainerDoingTasks = "use `/issue` or the options described in `/help`."

// BuildGouDemoSystemPrompt mirrors TS getSystemPrompt slice order: static prefix through output efficiency,
// optional SYSTEM_PROMPT_DYNAMIC_BOUNDARY, then dynamic sections (session, memory, ant override, env, …).
func BuildGouDemoSystemPrompt(o GouDemoSystemOpts) string {
	if envTruthyGo("CLAUDE_CODE_SIMPLE") {
		// Date: callers may set via session; TS uses getSessionStartDate().
		return strings.TrimSpace(SimpleModeSystemPrompt(o.Cwd, os.Getenv("CLAUDE_CODE_GO_SESSION_START_DATE")))
	}
	if ProactiveModeActive() {
		return buildProactiveGouDemoSystemPrompt(o)
	}
	var parts []string
	intro := gouDemoSimpleIntro(o.OutputStyleName, o.OutputStylePrompt)
	if strings.TrimSpace(intro) != "" {
		parts = append(parts, strings.TrimSpace(intro))
	}
	if sys := SimpleSystemSection(); strings.TrimSpace(sys) != "" {
		parts = append(parts, strings.TrimSpace(sys))
	}
	if gouDemoShouldIncludeDoingTasks(o) {
		if dt := DoingTasksSection(o.UserTypeAnt, askUserQuestionToolName, issuesExplainerDoingTasks); strings.TrimSpace(dt) != "" {
			parts = append(parts, strings.TrimSpace(dt))
		}
	}
	if ac := ActionsSection(); strings.TrimSpace(ac) != "" {
		parts = append(parts, strings.TrimSpace(ac))
	}
	if ut := gouDemoUsingYourToolsSection(o); strings.TrimSpace(ut) != "" {
		parts = append(parts, strings.TrimSpace(ut))
	}
	if ts := gouDemoToneAndStyleSection(o.UserTypeAnt); strings.TrimSpace(ts) != "" {
		parts = append(parts, strings.TrimSpace(ts))
	}
	if oe := gouDemoOutputEfficiencySection(o.UserTypeAnt); strings.TrimSpace(oe) != "" {
		parts = append(parts, strings.TrimSpace(oe))
	}
	if ShouldUseGlobalCacheScope() {
		parts = append(parts, SystemPromptDynamicBoundary)
	}
	if sg := SessionSpecificGuidanceFull(o); strings.TrimSpace(sg) != "" {
		parts = append(parts, strings.TrimSpace(sg))
	}
	if mem := BuildAutoMemoryPrompt(o); strings.TrimSpace(mem) != "" {
		parts = append(parts, strings.TrimSpace(mem))
	}
	if s := strings.TrimSpace(antModelOverrideSection(o)); s != "" {
		parts = append(parts, s)
	}
	goos := runtime.GOOS
	if strings.TrimSpace(o.ParityGOOS) != "" {
		goos = strings.TrimSpace(o.ParityGOOS)
	}
	isGit, worktree := false, false
	if !o.SkipPromptGitDetect {
		isGit, worktree = PromptGitHints(o.Cwd)
	}
	if o.PromptIsGitRepo != nil {
		isGit = *o.PromptIsGitRepo
	}
	if o.PromptGitWorktree != nil {
		worktree = *o.PromptGitWorktree
	}
	envBlock := ComputeSimpleEnvInfo(SimpleEnvInfoInput{
		ModelID:                      o.ModelID,
		EnvReportModelID:             o.EnvModelForSystemPrompt(),
		PrimaryWorkingDirectory:      o.Cwd,
		AdditionalWorkingDirectories: slicesCloneStringSlice(o.AdditionalWorkingDirs),
		IsGitRepo:                    isGit,
		GitWorktreeSession:           worktree,
		PlatformGOOS:                 goos,
		UserTypeAnt:                  o.UserTypeAnt,
		Undercover:                   o.Undercover,
	})
	if strings.TrimSpace(envBlock) != "" {
		parts = append(parts, strings.TrimSpace(envBlock))
	}
	if lang := LanguageSection(o.Language); lang != "" {
		parts = append(parts, lang)
	}
	if os := OutputStyleSection(o.OutputStyleName, o.OutputStylePrompt); os != "" {
		parts = append(parts, os)
	}
	if mcp := strings.TrimSpace(o.MCPInstructionsMarkdown); mcp != "" && !IsMcpInstructionsDeltaEnabled() {
		parts = append(parts, mcp)
	}
	if sp := GetScratchpadInstructions(strings.TrimSpace(o.ScratchpadDir)); sp != "" {
		parts = append(parts, sp)
	}
	frc := strings.TrimSpace(o.FunctionResultClearingMarkdown)
	if frc == "" {
		frc = FunctionResultClearingSection(o.ModelID)
	}
	if frc != "" {
		parts = append(parts, frc)
	}
	if sum := strings.TrimSpace(SummarizeToolResultsSection); sum != "" {
		parts = append(parts, sum)
	}
	if o.UserTypeAnt {
		parts = append(parts, NumericLengthAnchorsSection())
	}
	if featuregates.Feature("TOKEN_BUDGET") {
		parts = append(parts, TokenBudgetSection())
	}
	br := strings.TrimSpace(o.BriefSectionMarkdown)
	if br == "" {
		br = GetBriefSection(o)
	}
	if br != "" {
		parts = append(parts, br)
	}
	return strings.Join(parts, "\n\n")
}

func slicesCloneStringSlice(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return append([]string(nil), s...)
}

func gouDemoShouldIncludeDoingTasks(o GouDemoSystemOpts) bool {
	if strings.TrimSpace(o.OutputStyleName) == "" {
		return true
	}
	return o.KeepCodingInstructions
}

func antModelOverrideSection(o GouDemoSystemOpts) string {
	if !o.UserTypeAnt || o.Undercover {
		return ""
	}
	s := strings.TrimSpace(o.AntModelOverrideSuffix)
	if s == "" {
		s = AntModelDefaultSystemPromptSuffixFromEnv()
	}
	return s
}

func gouDemoSimpleIntro(outputStyleName, outputStylePrompt string) string {
	task := "with software engineering tasks."
	if strings.TrimSpace(outputStyleName) != "" && strings.TrimSpace(outputStylePrompt) != "" {
		task = `according to your "Output Style" below, which describes how you should respond to user queries.`
	}
	return fmt.Sprintf(`
You are an interactive agent that helps users %s Use the instructions below and the tools available to you to assist the user.

%s
IMPORTANT: You must NEVER generate or guess URLs for the user unless you are confident that the URLs are for helping the user with programming. You may use URLs provided by the user in their messages or local files.`, task, CyberRiskInstruction)
}

func taskToolNameForUsingYourTools(enabled map[string]struct{}) string {
	if enabled == nil {
		return ""
	}
	if _, ok := enabled[taskCreateToolName]; ok {
		return taskCreateToolName
	}
	if _, ok := enabled[todoWriteToolName]; ok {
		return todoWriteToolName
	}
	return ""
}

// gouDemoUsingYourToolsSection mirrors getUsingYourToolsSection (prompts.ts); tool names from *Tool/prompt.ts.
func gouDemoUsingYourToolsSection(o GouDemoSystemOpts) string {
	enabled := o.EnabledToolNames
	taskName := taskToolNameForUsingYourTools(enabled)
	if o.ReplModeEnabled {
		if taskName == "" {
			return ""
		}
		return fmt.Sprintf(`# Using your tools
 - Break down and manage your work with the %s tool. These tools are helpful for planning your work and helping the user track your progress. Mark each task as completed as soon as you are done with the task. Do not batch up multiple tasks before marking them as completed.`, taskName)
	}
	var subitems []string
	subitems = append(subitems,
		fmt.Sprintf(`To read files use %s instead of cat, head, tail, or sed`, fileReadToolName),
		fmt.Sprintf(`To edit files use %s instead of sed or awk`, fileEditToolName),
		fmt.Sprintf(`To create files use %s instead of cat with heredoc or echo redirection`, fileWriteToolName),
	)
	if !o.EmbeddedSearchTools {
		subitems = append(subitems,
			fmt.Sprintf(`To search for files use %s instead of find or ls`, globToolName),
			fmt.Sprintf(`To search the content of files, use %s instead of grep or rg`, grepToolName),
		)
	}
	subitems = append(subitems, fmt.Sprintf(`Reserve using the %s exclusively for system commands and terminal operations that require shell execution. If you are unsure and there is a relevant dedicated tool, default to using the dedicated tool and only fallback on using the %s tool for these if it is absolutely necessary.`, bashToolName, bashToolName))

	var b strings.Builder
	b.WriteString("# Using your tools\n")
	b.WriteString(fmt.Sprintf(` - Do NOT use the %s to run commands when a relevant dedicated tool is provided. Using dedicated tools allows the user to better understand and review your work. This is CRITICAL to assisting the user:`, bashToolName))
	b.WriteByte('\n')
	for _, s := range subitems {
		b.WriteString("  - ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if taskName != "" {
		b.WriteString(fmt.Sprintf(` - Break down and manage your work with the %s tool. These tools are helpful for planning your work and helping the user track your progress. Mark each task as completed as soon as you are done with the task. Do not batch up multiple tasks before marking them as completed.`, taskName))
		b.WriteByte('\n')
	}
	b.WriteString(` - You can call multiple tools in a single response. If you intend to call multiple tools and there are no dependencies between them, make all independent tool calls in parallel. Maximize use of parallel tool calls where possible to increase efficiency. However, if some tool calls depend on previous calls to inform dependent values, do NOT call these tools in parallel and instead call them sequentially. For instance, if one operation must complete before another starts, run these operations sequentially instead.`)
	b.WriteByte('\n')
	return b.String()
}

// gouDemoToneAndStyleSection mirrors getSimpleToneAndStyleSection (prompts.ts).
func gouDemoToneAndStyleSection(userTypeAnt bool) string {
	var b strings.Builder
	b.WriteString("# Tone and style\n")
	b.WriteString(` - Only use emojis if the user explicitly requests it. Avoid using emojis in all communication unless asked.`)
	b.WriteByte('\n')
	if !userTypeAnt {
		b.WriteString(` - Your responses should be short and concise.`)
		b.WriteByte('\n')
	}
	b.WriteString(` - When referencing specific functions or pieces of code include the pattern file_path:line_number to allow the user to easily navigate to the source code location.`)
	b.WriteByte('\n')
	b.WriteString(` - When referencing GitHub issues or pull requests, use the owner/repo#123 format (e.g. anthropics/claude-code#100) so they render as clickable links.`)
	b.WriteByte('\n')
	b.WriteString(` - Do not use a colon before tool calls. Your tool calls may not be shown directly in the output, so text like "Let me read the file:" followed by a read tool call should just be "Let me read the file." with a period.`)
	b.WriteByte('\n')
	return b.String()
}

// gouDemoOutputEfficiencySection mirrors getOutputEfficiencySection (prompts.ts).
func gouDemoOutputEfficiencySection(userTypeAnt bool) string {
	if userTypeAnt {
		return `# Communicating with the user
When sending user-facing text, you're writing for a person, not logging to a console. Assume users can't see most tool calls or thinking - only your text output. Before your first tool call, briefly state what you're about to do. While working, give short updates at key moments: when you find something load-bearing (a bug, a root cause), when changing direction, when you've made progress without an update.

When making updates, assume the person has stepped away and lost the thread. They don't know codenames, abbreviations, or shorthand you created along the way, and didn't track your process. Write so they can pick back up cold: use complete, grammatically correct sentences without unexplained jargon. Expand technical terms. Err on the side of more explanation. Attend to cues about the user's level of expertise; if they seem like an expert, tilt a bit more concise, while if they seem like they're new, be more explanatory. 

Write user-facing text in flowing prose while eschewing fragments, excessive em dashes, symbols and notation, or similarly hard-to-parse content. Only use tables when appropriate; for example to hold short enumerable facts (file names, line numbers, pass/fail), or communicate quantitative data. Don't pack explanatory reasoning into table cells -- explain before or after. Avoid semantic backtracking: structure each sentence so a person can read it linearly, building up meaning without having to re-parse what came before. 

What's most important is the reader understanding your output without mental overhead or follow-ups, not how terse you are. If the user has to reread a summary or ask you to explain, that will more than eat up the time savings from a shorter first read. Match responses to the task: a simple question gets a direct answer in prose, not headers and numbered sections. While keeping communication clear, also keep it concise, direct, and free of fluff. Avoid filler or stating the obvious. Get straight to the point. Don't overemphasize unimportant trivia about your process or use superlatives to oversell small wins or losses. Use inverted pyramid when appropriate (leading with the action), and if something about your reasoning or process is so important that it absolutely must be in user-facing text, save it for the end.

These user-facing text instructions do not apply to code or tool calls.`
	}
	return `# Output efficiency

IMPORTANT: Go straight to the point. Try the simplest approach first without going in circles. Do not overdo it. Be extra concise.

Keep your text output brief and direct. Lead with the answer or action, not the reasoning. Skip filler words, preamble, and unnecessary transitions. Do not restate what the user said — just do it. When explaining, include only what is necessary for the user to understand.

Focus text output on:
- Decisions that need the user's input
- High-level status updates at natural milestones
- Errors or blockers that change the plan

If you can say it in one sentence, don't use three. Prefer short, direct sentences over long explanations. This does not apply to code or tool calls.`
}

// LanguageSection mirrors getLanguageSection (prompts.ts).
func LanguageSection(languagePreference string) string {
	p := strings.TrimSpace(languagePreference)
	if p == "" {
		return ""
	}
	return fmt.Sprintf(`# Language
Always respond in %s. Use %s for all explanations, comments, and communications with the user. Technical terms and code identifiers should remain in their original form.`, p, p)
}

// OutputStyleSection mirrors getOutputStyleSection (prompts.ts).
func OutputStyleSection(name, prompt string) string {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(prompt) == "" {
		return ""
	}
	return fmt.Sprintf("# Output Style: %s\n%s", strings.TrimSpace(name), strings.TrimSpace(prompt))
}
