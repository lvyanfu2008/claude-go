// Gates and optional sections from src/constants/prompts.ts and dependencies (copied verbatim; env mirrors GrowthBook / bootstrap).
package commands

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"

	"goc/commands/featuregates"
)

// BriefToolName matches BRIEF_TOOL_NAME in src/tools/BriefTool/prompt.ts.
const BriefToolName = "SendUserMessage"

// BriefToolDescription matches DESCRIPTION in src/tools/BriefTool/prompt.ts.
const BriefToolDescription = "Send a message to the user"

// BriefToolUserPrompt matches BRIEF_TOOL_PROMPT in src/tools/BriefTool/prompt.ts.
const BriefToolUserPrompt = `Send a message the user will read. Text outside this tool is visible in the detail view, but most won't open it — the answer lives here.

` + "`message`" + ` supports markdown. ` + "`attachments`" + ` takes file paths (absolute or cwd-relative) for images, diffs, logs.

` + "`status`" + ` labels intent: 'normal' when replying to what they just asked; 'proactive' when you're initiating — a scheduled task finished, a blocker surfaced during background work, you need input on something they haven't asked about. Set it honestly; downstream routing uses it.`

// BriefProactiveSectionBody matches BRIEF_PROACTIVE_SECTION in prompt.ts (template uses BriefToolName).
func BriefProactiveSectionBody() string {
	t := BriefToolName
	return `## Talking to the user

` + t + ` is where your replies go. Text outside it is visible if the user expands the detail view, but most won't — assume unread. Anything you want them to actually see goes through ` + t + `. The failure mode: the real answer lives in plain text while ` + t + ` just says "done!" — they see "done!" and miss everything.

So: every time the user says something, the reply they actually read comes through ` + t + `. Even for "hi". Even for "thanks".

If you can answer right away, send the answer. If you need to go look — run a command, read files, check something — ack first in one line ("On it — checking the test output"), then work, then send the result. Without the ack they're staring at a spinner.

For longer work: ack → work → result. Between those, send a checkpoint when something useful happened — a decision you made, a surprise you hit, a phase boundary. Skip the filler ("running tests...") — a checkpoint earns its place by carrying information.

Keep messages tight — the decision, the file:line, the PR number. Second person always ("your config"), never third.`
}

// BriefEntitled mirrors isBriefEntitled() in BriefTool.ts (kairosActive + env + GrowthBook shim).
func BriefEntitled(o GouDemoSystemOpts) bool {
	if !featuregates.Feature("KAIROS") && !featuregates.Feature("KAIROS_BRIEF") {
		return false
	}
	if o.KairosActive {
		return true
	}
	if envTruthyGo("CLAUDE_CODE_BRIEF") {
		return true
	}
	return envTruthyGo("CLAUDE_CODE_GO_TENGU_KAIROS_BRIEF")
}

// BriefEnabled mirrors isBriefEnabled() in BriefTool.ts.
func BriefEnabled(o GouDemoSystemOpts) bool {
	if !featuregates.Feature("KAIROS") && !featuregates.Feature("KAIROS_BRIEF") {
		return false
	}
	if !(o.KairosActive || o.UserMsgOptIn) {
		return false
	}
	return BriefEntitled(o)
}

// GetBriefSection mirrors getBriefSection() in prompts.ts (non-proactive main path).
func GetBriefSection(o GouDemoSystemOpts) string {
	if !featuregates.Feature("KAIROS") && !featuregates.Feature("KAIROS_BRIEF") {
		return ""
	}
	if !BriefEnabled(o) {
		return ""
	}
	if ProactiveModeActive() {
		return ""
	}
	return BriefProactiveSectionBody()
}

func envVarDefinedFalsy(val string) bool {
	if strings.TrimSpace(val) == "" {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

// IsMcpInstructionsDeltaEnabled mirrors isMcpInstructionsDelta in mcpInstructionsDelta.ts.
func IsMcpInstructionsDeltaEnabled() bool {
	raw := os.Getenv("CLAUDE_CODE_MCP_INSTR_DELTA")
	if envTruthyGoRaw(raw) {
		return true
	}
	if envVarDefinedFalsy(raw) {
		return false
	}
	if featuregates.UserTypeAnt() {
		return true
	}
	return envTruthyGo("CLAUDE_CODE_GO_TENGU_BASALT_3KR")
}

func envTruthyGoRaw(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

// ProactiveModeActive mirrors proactiveModule?.isProactiveActive() when PROACTIVE|KAIROS features are on.
func ProactiveModeActive() bool {
	if !featuregates.Feature("PROACTIVE") && !featuregates.Feature("KAIROS") {
		return false
	}
	return envTruthyGo("CLAUDE_CODE_GO_PROACTIVE_ACTIVE")
}

// coordinatorModeLikeTS matches src/coordinator/coordinatorMode.ts isCoordinatorMode:
// feature('COORDINATOR_MODE') && truthy CLAUDE_CODE_COORDINATOR_MODE.
func coordinatorModeLikeTS() bool {
	if !featuregates.Feature("COORDINATOR_MODE") {
		return false
	}
	return envTruthyGo("CLAUDE_CODE_COORDINATOR_MODE")
}

// nonInteractiveSessionEnvShim approximates getIsNonInteractiveSession() when no session
// struct is in scope (static tool schemas). Covers SDK/parity env used across Go.
// TS checks: -p/--print, --init-only, --sdk-url, or !process.stdout.isTTY.
func nonInteractiveSessionEnvShim() bool {
	nonInt := envTruthyGo("CLAUDE_CODE_NONINTERACTIVE")
	headless := envTruthyGo("HEADLESS")
	gouNonInt := envTruthyGo("GOU_DEMO_NON_INTERACTIVE")
	stdoutTTY := isatty.IsTerminal(os.Stdout.Fd())
	result := nonInt || headless || gouNonInt || !stdoutTTY
	//if envTruthyGo("CLAUDE_CODE_GO_DEBUG_AGENT_TOOL_SCHEMA") {
	//	diaglog.Line("[agent-tool-schema] nonInteractiveSessionEnvShim: CLAUDE_CODE_NONINTERACTIVE=%v HEADLESS=%v GOU_DEMO_NON_INTERACTIVE=%v stdoutIsTTY=%v result=%v",
	//		nonInt, headless, gouNonInt, stdoutTTY, result)
	//}
	return result
}

// ForkSubagentEnabled mirrors isForkSubagentEnabled in forkSubagent.ts.
// Coordinator uses coordinatorModeLikeTS, not o.CoordinatorMode (see CLAUDE_CODE_GO_COORDINATOR_MODE vs TS).
// Non-interactive: session bit or env shim, matching "fork off" in CI / headless / gou-demo.
func ForkSubagentEnabled(o GouDemoSystemOpts) bool {
	if !featuregates.Feature("FORK_SUBAGENT") {
		return false
	}
	if coordinatorModeLikeTS() {
		return false
	}
	if o.NonInteractiveSession || nonInteractiveSessionEnvShim() {
		return false
	}
	return true
}

// CachedMicrocompactFRCConfig mirrors fields read by getFunctionResultClearingSection in prompts.ts.
type CachedMicrocompactFRCConfig struct {
	Enabled                      bool
	SystemPromptSuggestSummaries bool
	SupportedModels              []string // modelID must contain one of these substrings
	KeepRecent                   int
}

// CachedMicrocompactFRCFromEnv loads FRC gate from env (GrowthBook stand-in). Empty supported list = any model when enabled.
func CachedMicrocompactFRCFromEnv() CachedMicrocompactFRCConfig {
	cfg := CachedMicrocompactFRCConfig{
		Enabled:                      envTruthyGo("CLAUDE_CODE_GO_CACHED_MC_FRC_ENABLED"),
		SystemPromptSuggestSummaries: envTruthyGo("CLAUDE_CODE_GO_CACHED_MC_SYSTEM_PROMPT_SUGGEST_SUMMARIES"),
		KeepRecent:                   8,
	}
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_GO_CACHED_MC_KEEP_RECENT")); v != "" {
		var n int
		for _, c := range v {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			cfg.KeepRecent = n
		}
	}
	if raw := strings.TrimSpace(os.Getenv("CLAUDE_CODE_GO_CACHED_MC_SUPPORTED_MODELS")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.SupportedModels = append(cfg.SupportedModels, p)
			}
		}
	}
	return cfg
}

// FunctionResultClearingSection mirrors getFunctionResultClearingSection in prompts.ts.
func FunctionResultClearingSection(modelID string) string {
	if !featuregates.Feature("CACHED_MICROCOMPACT") {
		return ""
	}
	cfg := CachedMicrocompactFRCFromEnv()
	if !cfg.Enabled || !cfg.SystemPromptSuggestSummaries {
		return ""
	}
	mid := strings.TrimSpace(modelID)
	if len(cfg.SupportedModels) > 0 {
		ok := false
		low := strings.ToLower(mid)
		for _, p := range cfg.SupportedModels {
			if strings.Contains(low, strings.ToLower(strings.TrimSpace(p))) {
				ok = true
				break
			}
		}
		if !ok {
			return ""
		}
	}
	kr := cfg.KeepRecent
	if kr <= 0 {
		kr = 8
	}
	return `# Function Result Clearing

Old tool results will be automatically cleared from context to free up space. The ` + strconv.Itoa(kr) + ` most recent results are always kept.`
}

// AntModelDefaultSystemPromptSuffixFromEnv mirrors getAntModelOverrideConfig()?.defaultSystemPromptSuffix via env.
func AntModelDefaultSystemPromptSuffixFromEnv() string {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_GO_ANT_MODEL_OVERRIDE_SUFFIX"))
}

// buildProactiveGouDemoSystemPrompt mirrors getSystemPrompt simple-proactive branch (prompts.ts).
func buildProactiveGouDemoSystemPrompt(o GouDemoSystemOpts) string {
	goos := runtimeGOOSForPrompt(o)
	isGit, worktree := gouDemoGitHintsForPrompt(o)
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
	lang := LanguageSection(o.Language)
	mcp := ""
	if s := strings.TrimSpace(o.MCPInstructionsMarkdown); s != "" && !IsMcpInstructionsDeltaEnabled() {
		mcp = s
	}
	scratch := GetScratchpadInstructions(strings.TrimSpace(o.ScratchpadDir))
	frc := strings.TrimSpace(o.FunctionResultClearingMarkdown)
	if frc == "" {
		frc = FunctionResultClearingSection(o.ModelID)
	}
	briefAppend := ""
	if BriefEnabled(o) {
		briefAppend = BriefProactiveSectionBody()
	}
	proactive := ProactiveAutonomousWorkSection(briefAppend)
	parts := ProactiveSystemPromptParts(
		BuildAutoMemoryPrompt(o),
		envBlock,
		lang,
		mcp,
		scratch,
		frc,
		proactive,
	)
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func runtimeGOOSForPrompt(o GouDemoSystemOpts) string {
	goos := runtime.GOOS
	if strings.TrimSpace(o.ParityGOOS) != "" {
		goos = strings.TrimSpace(o.ParityGOOS)
	}
	return goos
}

func gouDemoGitHintsForPrompt(o GouDemoSystemOpts) (isGit, worktree bool) {
	if !o.SkipPromptGitDetect {
		isGit, worktree = PromptGitHints(o.Cwd)
	}
	if o.PromptIsGitRepo != nil {
		isGit = *o.PromptIsGitRepo
	}
	if o.PromptGitWorktree != nil {
		worktree = *o.PromptGitWorktree
	}
	return isGit, worktree
}
