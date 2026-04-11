package commands

import (
	"fmt"
	"strings"

	"goc/types"
)

const skillToolName = "Skill"

// Tool / agent names aligned with src/tools/*/prompt.ts and AgentTool/constants.ts.
const (
	agentToolName          = "Agent"
	bashToolName           = "Bash"
	fileReadToolName       = "Read"
	fileEditToolName       = "Edit"
	fileWriteToolName      = "Write"
	globToolName           = "Glob"
	grepToolName           = "Grep"
	taskCreateToolName     = "TaskCreate"
	todoWriteToolName      = "TodoWrite"
	exploreAgentType       = "Explore"
	exploreAgentMinQueries = 3
	verificationAgentType  = "verification"
)

// SessionSpecificGuidance mirrors the skills-related branch of getSessionSpecificGuidanceSection in src/constants/prompts.ts
// when hasSkills is true (skillToolCommands non-empty and Skill in enabledTools).
// Format matches TS: "# Session-specific guidance" + prependBullets single item (" - " prefix).
func SessionSpecificGuidance(enabledToolNames map[string]struct{}, skillToolCommands []types.Command) string {
	b := skillSlashGuidanceBullet(enabledToolNames, skillToolCommands)
	if b == "" {
		return ""
	}
	return "# Session-specific guidance\n - " + b
}

func skillSlashGuidanceBullet(enabledToolNames map[string]struct{}, skillToolCommands []types.Command) string {
	if len(skillToolCommands) == 0 {
		return ""
	}
	if enabledToolNames == nil {
		return ""
	}
	if _, ok := enabledToolNames[skillToolName]; !ok {
		return ""
	}
	return `/<skill-name> (e.g., /commit) is shorthand for users to invoke a user-invocable skill. When executed, the skill gets expanded to a full prompt. Use the ` + skillToolName + ` tool to execute them. IMPORTANT: Only use ` + skillToolName + ` for skills listed in its user-invocable skills section - do not guess or use built-in CLI commands.`
}

const askUserQuestionToolName = "AskUserQuestion"

func agentToolSectionNonFork() string {
	return fmt.Sprintf(`Use the %s tool with specialized agents when the task at hand matches the agent's description. Subagents are valuable for parallelizing independent queries or for protecting the main context window from excessive results, but they should not be used excessively when not needed. Importantly, avoid duplicating work that subagents are already doing - if you delegate research to a subagent, do not also perform the same searches yourself.`, agentToolName)
}

func agentToolSectionFork() string {
	return fmt.Sprintf(`Calling %s without a subagent_type creates a fork, which runs in the background and keeps its tool output out of your context — so you can keep chatting with the user while it works. Reach for it when research or multi-step implementation work would otherwise fill your context with raw output you won't need again. **If you ARE the fork** — execute directly; do not re-delegate.`, agentToolName)
}

func agentToolSectionForOpts(o GouDemoSystemOpts) string {
	if ForkSubagentEnabled(o) {
		return agentToolSectionFork()
	}
	return agentToolSectionNonFork()
}

func verificationAgentGuidanceBullet() string {
	return fmt.Sprintf(`The contract: when non-trivial implementation happens on your turn, independent adversarial verification must happen before you report completion — regardless of who did the implementing (you directly, a fork you spawned, or a subagent). You are the one reporting to the user; you own the gate. Non-trivial means: 3+ file edits, backend/API changes, or infrastructure changes. Spawn the %s tool with subagent_type="%s". Your own checks, caveats, and a fork's self-checks do NOT substitute — only the verifier assigns a verdict; you cannot self-assign PARTIAL. Pass the original user request, all files changed (by anyone), the approach, and the plan file path if applicable. Flag concerns if you have them but do NOT share test results or claim things work. On FAIL: fix, resume the verifier with its findings plus your fix, repeat until PASS. On PASS: spot-check it — re-run 2-3 commands from its report, confirm every PASS has a Command run block with output that matches your re-run. If any PASS lacks a command block or diverges, resume the verifier with the specifics. On PARTIAL (from the verifier): report what passed and what could not be verified.`, agentToolName, verificationAgentType)
}

// SessionSpecificGuidanceFull mirrors getSessionSpecificGuidanceSection order in src/constants/prompts.ts (subset + optional explore/verification gates).
func SessionSpecificGuidanceFull(o GouDemoSystemOpts) string {
	enabledToolNames := o.EnabledToolNames
	if enabledToolNames == nil {
		enabledToolNames = map[string]struct{}{}
	}
	var bullets []string
	if _, ok := enabledToolNames[askUserQuestionToolName]; ok {
		bullets = append(bullets, fmt.Sprintf(`If you do not understand why the user has denied a tool call, use the %s tool to ask them.`, askUserQuestionToolName))
	}
	if !o.NonInteractiveSession {
		bullets = append(bullets, `If you need the user to run a shell command themselves (e.g., an interactive login like `+"`gcloud auth login`"+`), suggest they type `+"`! <command>`"+` in the prompt — the `+"`!`"+` prefix runs the command in this session so its output lands directly in the conversation.`)
	}
	if _, ok := enabledToolNames[agentToolName]; ok {
		bullets = append(bullets, agentToolSectionForOpts(o))
	}
	if _, hasAgent := enabledToolNames[agentToolName]; hasAgent && o.ExplorePlanAgentsEnabled && !o.ReplModeEnabled && !ForkSubagentEnabled(o) {
		searchTools := fmt.Sprintf("the %s or %s", globToolName, grepToolName)
		if o.EmbeddedSearchTools {
			searchTools = fmt.Sprintf("`find` or `grep` via the %s tool", bashToolName)
		}
		bullets = append(bullets,
			fmt.Sprintf(`For simple, directed codebase searches (e.g. for a specific file/class/function) use %s directly.`, searchTools),
			fmt.Sprintf(`For broader codebase exploration and deep research, use the %s tool with subagent_type=%s. This is slower than using %s directly, so use this only when a simple, directed search proves to be insufficient or when your task will clearly require more than %d queries.`, agentToolName, exploreAgentType, searchTools, exploreAgentMinQueries),
		)
	}
	if b := skillSlashGuidanceBullet(enabledToolNames, o.SkillToolCommands); b != "" {
		bullets = append(bullets, b)
	}
	ds := strings.TrimSpace(o.DiscoverSkillsToolName)
	if ds != "" && len(o.SkillToolCommands) > 0 {
		if _, ok := enabledToolNames[ds]; ok {
			bullets = append(bullets, DiscoverSkillsGuidance(ds))
		}
	}
	if _, ok := enabledToolNames[agentToolName]; ok && o.VerificationAgentGuidance {
		bullets = append(bullets, verificationAgentGuidanceBullet())
	}
	if len(bullets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Session-specific guidance\n")
	for _, bl := range bullets {
		b.WriteString(" - ")
		b.WriteString(bl)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// EnabledToolNames builds a set from tool definition names (TS Set(tools.map(t => t.name))).
func EnabledToolNames(toolNames []string) map[string]struct{} {
	m := make(map[string]struct{}, len(toolNames))
	for _, n := range toolNames {
		n = strings.TrimSpace(n)
		if n != "" {
			m[n] = struct{}{}
		}
	}
	return m
}
