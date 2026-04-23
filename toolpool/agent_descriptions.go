package toolpool

import (
	"goc/commands/featuregates"
	"goc/utils"
	"strings"
)

const agentToolName = "Agent"

// agentToolDescriptionBase contains the base description for the Agent tool,
// extracted from TypeScript implementation to avoid dependency on embedded JSON.
const agentToolDescriptionBase = `Launch a new agent to handle complex, multi-step tasks autonomously.

The Agent tool launches specialized agents (subprocesses) that autonomously handle complex tasks. Each agent type has specific capabilities and tools available to it.

Available agent types and the tools they have access to:

__AGENT_LIST_PLACEHOLDER__

__AGENT_TYPE_SELECTION_PLACEHOLDER__

__AGENT_TYPE_NOT_SELECTION_PLACEHOLDER__


Usage notes:
- Always include a short description (3-5 words) summarizing what the agent will do
- Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses
- When the agent is done, it will return a single message back to you. The result returned by the agent is not visible to the user. To show the user the result, you should send a text message back to the user with a concise summary of the result.__BACKGROUND_TASKS_PLACEHOLDER__
- To continue a previously spawned agent, use SendMessage with the agent's ID or name as the ` + "`to`" + ` field. The agent resumes with its full context preserved. Each Agent invocation starts fresh — provide a complete task description.
- The agent's outputs should generally be trusted
- Clearly tell the agent whether you expect it to write code or just to do research (search, file reads, web fetches, etc.), since it is not aware of the user's intent
- If the agent description mentions that it should be used proactively, then you should try your best to use it without the user having to ask for it first. Use your judgement.
- If the user specifies that they want you to run agents "in parallel", you MUST send a single message with multiple Agent tool use content blocks. For example, if you need to launch both a build-validator agent and a test-runner agent in parallel, send a single message with both tool calls.
- You can optionally set ` + "`isolation: \"worktree\"`" + ` to run the agent in a temporary git worktree, giving it an isolated copy of the repository. The worktree is automatically cleaned up if the agent makes no changes; if changes are made, the worktree path and branch are returned in the result.

## Writing the prompt

Brief the agent like a smart colleague who just walked into the room — it hasn't seen this conversation, doesn't know what you've tried, doesn't understand why this task matters.
- Explain what you're trying to accomplish and why.
- Describe what you've already learned or ruled out.
- Give enough context about the surrounding problem that the agent can make judgment calls rather than just following a narrow instruction.
- If you need a short response, say so ("report in under 200 words").
- Lookups: hand over the exact command. Investigations: hand over the question — prescribed steps become dead weight when the premise is wrong.

Terse command-style prompts produce shallow, generic work.

**Never delegate understanding.** Don't write "based on your findings, fix the bug" or "based on the research, implement it." Those phrases push synthesis onto the agent instead of doing it yourself. Write prompts that prove you understood: include file paths, line numbers, what specifically to change.


__EXAMPLES_PLACEHOLDER__`

// AgentInfo holds information about an agent for description formatting
type AgentInfo struct {
	AgentType       string
	WhenToUse       string
	Tools           []string
	DisallowedTools []string
}

// formatAgentLine formats one agent line for the agent tool description:
// `- type: whenToUse (Tools: ...)`.
func formatAgentLine(agent AgentInfo) string {
	var toolsDesc string
	if len(agent.Tools) > 0 && len(agent.DisallowedTools) > 0 {
		// Both defined: filter allowlist by denylist to match runtime behavior
		denySet := make(map[string]bool)
		for _, d := range agent.DisallowedTools {
			denySet[d] = true
		}
		var effectiveTools []string
		for _, t := range agent.Tools {
			if !denySet[t] {
				effectiveTools = append(effectiveTools, t)
			}
		}
		if len(effectiveTools) == 0 {
			toolsDesc = "None"
		} else {
			toolsDesc = strings.Join(effectiveTools, ", ")
		}
	} else if len(agent.Tools) > 0 {
		// Allowlist only: show the specific tools available
		toolsDesc = strings.Join(agent.Tools, ", ")
	} else if len(agent.DisallowedTools) > 0 {
		// Denylist only: show "All tools except X, Y, Z"
		toolsDesc = "All tools except " + strings.Join(agent.DisallowedTools, ", ")
	} else {
		// No restrictions
		toolsDesc = "All tools"
	}
	return "- " + agent.AgentType + ": " + agent.WhenToUse + " (Tools: " + toolsDesc + ")"
}

// getBuiltinAgentInfos returns static agent info to avoid circular dependencies
func getBuiltinAgentInfos() []AgentInfo {
	agents := []AgentInfo{
		{
			AgentType: "general-purpose",
			WhenToUse: "General-purpose agent for researching complex questions, searching for code, and executing multi-step tasks. When you are searching for a keyword or file and are not confident that you will find the right match in the first few tries use this agent to perform the search for you.",
			Tools:     []string{"*"},
		},
		{
			AgentType: "statusline-setup",
			WhenToUse: "Use this agent to configure the user's Claude Code status line setting.",
			Tools:     []string{"Read", "Edit"},
		},
	}

	// Add claude-code-guide agent (conditionally based on entry point, but we'll include it for now)
	agents = append(agents, AgentInfo{
		AgentType: "claude-code-guide",
		WhenToUse: "Use this agent when the user asks questions (\"Can Claude...\", \"Does Claude...\", \"How do I...\") about: (1) Claude Code (the CLI tool) - features, hooks, slash commands, MCP servers, settings, IDE integrations, keyboard shortcuts; (2) Claude Agent SDK - building custom agents; (3) Claude API (formerly Anthropic API) - API usage, tool use, Anthropic SDK usage. **IMPORTANT:** Before spawning a new agent, check if there is already a running or recently completed claude-code-guide agent that you can continue via SendMessage.",
		Tools:     []string{"Glob", "Grep", "Read", "WebFetch", "WebSearch"},
	})

	// Add verification agent if enabled
	if featuregates.Feature("VERIFICATION_AGENT") {
		agents = append(agents, AgentInfo{
			AgentType:       "verification",
			WhenToUse:       "Use this agent to verify that implementation work is correct before reporting completion. Invoke after non-trivial tasks (3+ file edits, backend/API changes, infrastructure changes). Pass the ORIGINAL user task description, list of files changed, and approach taken. The agent runs builds, tests, linters, and checks to produce a PASS/FAIL/PARTIAL verdict with evidence.",
			DisallowedTools: []string{"Agent", "ExitPlanMode", "Edit", "Write", "NotebookEdit"},
		})
	}

	// Add explore and plan agents if enabled
	if featuregates.Feature("BUILTIN_EXPLORE_PLAN_AGENTS") {
		agents = append(agents, AgentInfo{
			AgentType:       "Explore",
			WhenToUse:       "Fast agent specialized for exploring codebases. Use this when you need to quickly find files by patterns (eg. \"src/components/**/*.tsx\"), search code for keywords (eg. \"API endpoints\"), or answer questions about the codebase (eg. \"how do API endpoints work?\"). When calling this agent, specify the desired thoroughness level: \"quick\" for basic searches, \"medium\" for moderate exploration, or \"very thorough\" for comprehensive analysis across multiple locations and naming conventions.",
			DisallowedTools: []string{"Agent", "ExitPlanMode", "Edit", "Write", "NotebookEdit"},
		})
		agents = append(agents, AgentInfo{
			AgentType:       "Plan",
			WhenToUse:       "Software architect agent for designing implementation plans. Use this when you need to plan the implementation strategy for a task. Returns step-by-step plans, identifies critical files, and considers architectural trade-offs.",
			DisallowedTools: []string{"Agent", "ExitPlanMode", "Edit", "Write", "NotebookEdit"},
		})
	}

	return agents
}

// getAgentTypeSelection returns the dynamic text about subagent_type usage
func getAgentTypeSelection(isForkEnabled bool) string {
	if isForkEnabled {
		return "When using the " + agentToolName + " tool, specify a subagent_type to use a specialized agent, or omit it to fork yourself — a fork inherits your full conversation context."
	}
	return "When using the " + agentToolName + " tool, specify a subagent_type parameter to select which agent type to use. If omitted, the general-purpose agent is used."
}

func getAgentTypeNotSelection(isForkEnabled bool) string {
	if isForkEnabled {
		return ""
	}
	return "When NOT to use the " + agentToolName + " tool, - If you want to read a specific file path, use the ${FILE_READ_TOOL_NAME} tool or ${fileSearchHint} instead of the ${AGENT_TOOL_NAME} tool, to find the match more quickly\n- If you are searching for a specific class definition like \"class Foo\", use ${contentSearchHint} instead, to find the match more quickly\n- If you are searching for code within a specific file or set of 2-3 files, use the ${FILE_READ_TOOL_NAME} tool instead of the ${AGENT_TOOL_NAME} tool, to find the match more quickly\n- Other tasks that are not related to the agent descriptions above"
}

func getBackgroundTasksText() string {
	// Check conditions similar to TypeScript version:
	// !isEnvTruthy(process.env.CLAUDE_CODE_DISABLE_BACKGROUND_TASKS) &&
	// !isInProcessTeammate() &&  // Not available in Go version, assume false
	// !forkEnabled

	forkEnabled := featuregates.Feature("FORK_SUBAGENT")
	backgroundDisabled := utils.IsEnvTruthy("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS")

	// Include background tasks documentation only if:
	// - Background tasks are not disabled
	// - Fork is not enabled (matching TypeScript logic)
	if !backgroundDisabled && !forkEnabled {
		return `
- You can optionally run agents in the background using the run_in_background parameter. When an agent runs in the background, you will be automatically notified when it completes — do NOT sleep, poll, or proactively check on its progress. Continue with other work or respond to the user instead.
- **Foreground vs background**: Use foreground (default) when you need the agent's results before you can proceed — e.g., research agents whose findings inform your next steps. Use background when you have genuinely independent work to do in parallel.`
	}

	return ""
}

// AgentToolDescription returns the full description with comprehensive usage guidance
func AgentToolDescription() string {
	base := agentToolDescriptionBase

	// Get built-in agent information
	builtinAgents := getBuiltinAgentInfos()

	var agentLines []string
	for _, agent := range builtinAgents {
		agentLines = append(agentLines, formatAgentLine(agent))
	}

	// Replace the placeholder with actual agent list
	agentListText := strings.Join(agentLines, "\n")
	base = strings.Replace(base, "__AGENT_LIST_PLACEHOLDER__", agentListText, 1)

	// Replace the agent type selection placeholder with dynamic text
	agentTypeSelectionText := getAgentTypeSelection(featuregates.Feature("FORK_SUBAGENT"))
	base = strings.Replace(base, "__AGENT_TYPE_SELECTION_PLACEHOLDER__", agentTypeSelectionText, 1)

	agentTypeNotSelectionText := getAgentTypeNotSelection(featuregates.Feature("FORK_SUBAGENT"))
	base = strings.Replace(base, "__AGENT_TYPE_NOT_SELECTION_PLACEHOLDER__", agentTypeNotSelectionText, 1)

	// Replace background tasks placeholder based on feature flags and environment
	backgroundTasksText := getBackgroundTasksText()
	base = strings.Replace(base, "__BACKGROUND_TASKS_PLACEHOLDER__", backgroundTasksText, 1)

	// Set up examples based on whether fork is enabled
	var examplesText string

	if featuregates.Feature("FORK_SUBAGENT") {
		// Fork feature enabled - use fork examples and add fork section
		forkSection := `

## When to fork

Fork yourself (omit ` + "`subagent_type`" + `) when the intermediate tool output isn't worth keeping in your context. The criterion is qualitative — "will I need this output again" — not task size.
- **Research**: fork open-ended questions. If research can be broken into independent questions, launch parallel forks in one message. A fork beats a fresh subagent for this — it inherits context and shares your cache.
- **Implementation**: prefer to fork implementation work that requires more than a couple of edits. Do research before jumping to implementation.

Forks are cheap because they share your prompt cache. Don't set ` + "`model`" + ` on a fork — a different model can't reuse the parent's cache. Pass a short ` + "`name`" + ` (one or two words, lowercase) so the user can see the fork in the teams panel and steer it mid-run.

**Don't peek.** The tool result includes an ` + "`output_file`" + ` path — do not Read or tail it unless the user explicitly asks for a progress check. You get a completion notification; trust it. Reading the transcript mid-flight pulls the fork's tool noise into your context, which defeats the point of forking.

**Don't race.** After launching, you know nothing about what the fork found. Never fabricate or predict fork results in any format — not as prose, summary, or structured output. The notification arrives as a user-role message in a later turn; it is never something you write yourself. If the user asks a follow-up before the notification lands, tell them the fork is still running — give status, not a guess.

**Writing a fork prompt.** Since the fork inherits your context, the prompt is a *directive* — what to do, not what the situation is. Be specific about scope: what's in, what's out, what another agent is handling. Don't re-explain background.`

		base = base + forkSection

		// Use fork examples when fork is enabled
		examplesText = `Example usage:

<example>
user: "What's left on this branch before we can ship?"
assistant: <thinking>Forking this — it's a survey question. I want the punch list, not the git output in my context.</thinking>
` + "`Agent`" + `({
  name: "ship-audit",
  description: "Branch ship-readiness audit",
  prompt: "Audit what's left before this branch can ship. Check: uncommitted changes, commits ahead of main, whether tests exist, whether the GrowthBook gate is wired up, whether CI-relevant files changed. Report a punch list — done vs. missing. Under 200 words."
})
assistant: Ship-readiness audit running.
<commentary>
Turn ends here. The coordinator knows nothing about the findings yet. What follows is a SEPARATE turn — the notification arrives from outside, as a user-role message. It is not something the coordinator writes.
</commentary>
[later turn — notification arrives as user message]
assistant: Audit's back. Three blockers: no tests for the new prompt path, GrowthBook gate wired but not in build_flags.yaml, and one uncommitted file.
</example>

<example>
user: "so is the gate wired up or not"
<commentary>
User asks mid-wait. The audit fork was launched to answer exactly this, and it hasn't returned. The coordinator does not have this answer. Give status, not a fabricated result.
</commentary>
assistant: Still waiting on the audit — that's one of the things it's checking. Should land shortly.
</example>

<example>
user: "Can you get a second opinion on whether this migration is safe?"
assistant: <thinking>I'll ask the code-reviewer agent — it won't see my analysis, so it can give an independent read.</thinking>
<commentary>
A subagent_type is specified, so the agent starts fresh. It needs full context in the prompt. The briefing explains what to assess and why.
</commentary>
` + "`Agent`" + `({
  name: "migration-review",
  description: "Independent migration review",
  subagent_type: "code-reviewer",
  prompt: "Review migration 0042_user_schema.sql for safety. Context: we're adding a NOT NULL column to a 50M-row table. Existing rows get a backfill default. I want a second opinion on whether the backfill approach is safe under concurrent writes — I've checked locking behavior but want independent verification. Report: is this safe, and if not, what specifically breaks?"
})
</example>`
	} else {
		// Fork feature disabled - use current examples
		examplesText = `Example usage:

<example_agent_descriptions>
"test-runner": use this agent after you are done writing code to run tests
"greeting-responder": use this agent to respond to user greetings with a friendly joke
</example_agent_descriptions>

<example>
user: "Please write a function that checks if a number is prime"
assistant: I'm going to use the Write tool to write the following code:
<code>
function isPrime(n) {
  if (n <= 1) return false
  for (let i = 2; i * i <= n; i++) {
    if (n % i === 0) return false
  }
  return true
}
</code>
<commentary>
Since a significant piece of code was written and the task was completed, now use the test-runner agent to run the tests
</commentary>
assistant: Uses the ` + "`Agent`" + ` tool to launch the test-runner agent
</example>

<example>
user: "Hello"
<commentary>
Since the user is greeting, use the greeting-responder agent to respond with a friendly joke
</commentary>
assistant: "I'm going to use the ` + "`Agent`" + ` tool to launch the greeting-responder agent"
</example>`
	}

	// Replace the examples placeholder
	base = strings.Replace(base, "__EXAMPLES_PLACEHOLDER__", examplesText, 1)

	// Replace tool name placeholders with actual tool name
	base = strings.ReplaceAll(base, "`Agent`", agentToolName)

	return base
}
