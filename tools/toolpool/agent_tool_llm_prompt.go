package toolpool

import (
	"fmt"
	"strings"
)

// Constants for Agent tool LLM prompt text (mirrors tools/agent_prompt.go; exported for tools package re-exports).
const (
	AgentToolName       = "Agent"
	LegacyAgentToolName = "Task"
	FileReadToolName    = "Read"
	FileWriteToolName   = "Write"
	GlobToolName        = "Glob"
	SendMessageToolName = "SendMessage"
)

// AgentPromptOptions configures the agent prompt generation.
type AgentPromptOptions struct {
	IsCoordinator     bool
	AllowedAgentTypes []string
	HasEmbeddedSearch bool
	IsForkEnabled     bool
	IsProUser         bool
	IsTeammate        bool
	InProcessTeammate bool
}

// FormatAgentToolListingLine formats one `- agentType: whenToUse (Tools: ...)` row (TS formatAgentLine).
func FormatAgentToolListingLine(agent AgentInfo) string {
	return formatAgentLineForPrompt(agent)
}

func formatAgentLineForPrompt(agent AgentInfo) string {
	toolsDesc := "All tools"
	switch {
	case len(agent.Tools) > 0 && len(agent.DisallowedTools) > 0:
		effective := make([]string, 0, len(agent.Tools))
		deny := map[string]struct{}{}
		for _, d := range agent.DisallowedTools {
			deny[d] = struct{}{}
		}
		for _, t := range agent.Tools {
			if _, blocked := deny[t]; !blocked {
				effective = append(effective, t)
			}
		}
		if len(effective) == 0 {
			toolsDesc = "None"
		} else {
			toolsDesc = strings.Join(effective, ", ")
		}
	case len(agent.Tools) > 0:
		toolsDesc = strings.Join(agent.Tools, ", ")
	case len(agent.DisallowedTools) > 0:
		toolsDesc = "All tools except " + strings.Join(agent.DisallowedTools, ", ")
	}
	return fmt.Sprintf("- %s: %s (Tools: %s)", agent.AgentType, agent.WhenToUse, toolsDesc)
}

// AgentPromptWithOptions builds the full Agent tool description for the model (TS getPrompt parity).
func AgentPromptWithOptions(agentInfos []AgentInfo, opts AgentPromptOptions) string {
	effectiveAgents := agentInfos
	if len(opts.AllowedAgentTypes) > 0 {
		filtered := make([]AgentInfo, 0)
		allowedSet := make(map[string]bool)
		for _, t := range opts.AllowedAgentTypes {
			allowedSet[t] = true
		}
		for _, agent := range agentInfos {
			if allowedSet[agent.AgentType] {
				filtered = append(filtered, agent)
			}
		}
		effectiveAgents = filtered
	}

	agentListLines := make([]string, 0, len(effectiveAgents))
	for _, agent := range effectiveAgents {
		agentListLines = append(agentListLines, FormatAgentToolListingLine(agent))
	}

	agentListSection := fmt.Sprintf("Available agent types and the tools they have access to:\n%s",
		strings.Join(agentListLines, "\n"))

	shared := fmt.Sprintf(`Launch a new agent to handle complex, multi-step tasks. Each agent type has specific capabilities and tools available to it.

%s

%s`, agentListSection, getAgentTypeSelection(opts.IsForkEnabled))

	if opts.IsCoordinator {
		return shared
	}

	whenNotToUse := getWhenNotToUseSection(opts.HasEmbeddedSearch, opts.IsForkEnabled)
	usageNotes := getUsageNotesSection(opts)
	whenToFork := getWhenToForkSection(opts.IsForkEnabled)
	writingPrompt := getWritingPromptSection(opts.IsForkEnabled)
	examples := getExamplesSection(opts.IsForkEnabled)

	whenToUse := `## When to use

Reach for this when the task matches an available agent type, when you have independent work to run in parallel, or when answering would mean reading across several files — delegate it and you keep the conclusion, not the file dumps. For a single-fact lookup where you already know the file, symbol, or value, search directly. Once you've delegated a search, don't also run it yourself — wait for the result.`

	return fmt.Sprintf(`%s

%s
%s

Usage notes:
%s%s%s

%s`, shared, whenToUse, whenNotToUse, usageNotes, whenToFork, writingPrompt, examples)
}

func getAgentTypeSelection(isForkEnabled bool) string {
	if isForkEnabled {
		return fmt.Sprintf("When using the %s tool, specify a subagent_type to use a specialized agent, or omit it to fork yourself — a fork inherits your full conversation context.", AgentToolName)
	}
	return fmt.Sprintf("When using the %s tool, specify a subagent_type parameter to select which agent type to use. If omitted, the general-purpose agent is used.", AgentToolName)
}

func getWhenNotToUseSection(hasEmbeddedSearch bool, isForkEnabled bool) string {
	if isForkEnabled {
		return ""
	}

	fileSearchHint := GlobToolName + " tool"
	contentSearchHint := GlobToolName + " tool"

	if hasEmbeddedSearch {
		fileSearchHint = "`find` via the Bash tool"
		contentSearchHint = "`grep` via the Bash tool"
	}

	return fmt.Sprintf(`
When NOT to use the %s tool:
- If you want to read a specific file path, use the %s tool or %s instead of the %s tool, to find the match more quickly
- If you are searching for a specific class definition like "class Foo", use %s instead, to find the match more quickly
- If you are searching for code within a specific file or set of 2-3 files, use the %s tool instead of the %s tool, to find the match more quickly
- Other tasks that are not related to the agent descriptions above`,
		AgentToolName, FileReadToolName, fileSearchHint, AgentToolName, contentSearchHint, FileReadToolName, AgentToolName)
}

func getUsageNotesSection(opts AgentPromptOptions) string {
	var notes []string

	notes = append(notes, "- The agent's final message is returned to you as the tool result; it is not shown to the user — relay what matters.")

	continuationNote := fmt.Sprintf("- Use %s with the agent's ID or name to continue a previously spawned agent with its context intact; a new Agent call starts fresh.", SendMessageToolName)
	notes = append(notes, continuationNote)

	notes = append(notes, "- `isolation: \"worktree\"` gives the agent its own git worktree (auto-cleaned if unchanged).")

	if !opts.InProcessTeammate && !opts.IsForkEnabled {
		notes = append(notes, "- `run_in_background: true` runs the agent asynchronously; you'll be notified when it completes.")
	}

	if opts.IsProUser {
		notes = append(notes, "- When you launch multiple agents for independent work, send them in a single message with multiple tool uses so they run concurrently")
	}

	if opts.InProcessTeammate {
		notes = append(notes, "- The run_in_background, name, team_name, and mode parameters are not available in this context. Only synchronous subagents are supported.")
	} else if opts.IsTeammate {
		notes = append(notes, "- The name, team_name, and mode parameters are not available in this context — teammates cannot spawn other teammates. Omit them to spawn a subagent.")
	}

	return strings.Join(notes, "\n")
}

func getWritingPromptSection(isForkEnabled bool) string {
	contextNote := ""
	if isForkEnabled {
		contextNote = "When spawning a fresh agent (with a `subagent_type`), it starts with zero context. "
	}

	terseNote := "Terse"
	if isForkEnabled {
		terseNote = "For fresh agents, terse"
	}

	return fmt.Sprintf(`

## Writing the prompt

%sBrief the agent like a smart colleague who just walked into the room — it hasn't seen this conversation, doesn't know what you've tried, doesn't understand why this task matters.
- Explain what you're trying to accomplish and why.
- Describe what you've already learned or ruled out.
- Give enough context about the surrounding problem that the agent can make judgment calls rather than just following a narrow instruction.
- If you need a short response, say so ("report in under 200 words").
- Lookups: hand over the exact command. Investigations: hand over the question — prescribed steps become dead weight when the premise is wrong.

%s command-style prompts produce shallow, generic work.

**Never delegate understanding.** Don't write "based on your findings, fix the bug" or "based on the research, implement it." Those phrases push synthesis onto the agent instead of doing it yourself. Write prompts that prove you understood: include file paths, line numbers, what specifically to change.`, contextNote, terseNote)
}

func getExamplesSection(isForkEnabled bool) string {
	if isForkEnabled {
		return getForkExamples()
	}
	return getCurrentExamples()
}

func getForkExamples() string {
	return fmt.Sprintf(`Example usage:

<example>
user: "What's left on this branch before we can ship?"
assistant: <thinking>Forking this — it's a survey question. I want the punch list, not the git output in my context.</thinking>
%s({
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
%s({
  name: "migration-review",
  description: "Independent migration review",
  subagent_type: "code-reviewer",
  prompt: "Review migration 0042_user_schema.sql for safety. Context: we're adding a NOT NULL column to a 50M-row table. Existing rows get a backfill default. I want a second opinion on whether the backfill approach is safe under concurrent writes — I've checked locking behavior but want independent verification. Report: is this safe, and if not, what specifically breaks?"
})
</example>`, AgentToolName, AgentToolName)
}

func getWhenToForkSection(isForkEnabled bool) string {
	if !isForkEnabled {
		return ""
	}

	return fmt.Sprintf(`

## When to fork

Fork yourself (omit `+"`subagent_type`"+`) when the intermediate tool output isn't worth keeping in your context. The criterion is qualitative — "will I need this output again" — not task size.
- **Research**: fork open-ended questions. If research can be broken into independent questions, launch parallel forks in one message. A fork beats a fresh subagent for this — it inherits context and shares your cache.
- **Implementation**: prefer to fork implementation work that requires more than a couple of edits. Do research before jumping to implementation.

Forks are cheap because they share your prompt cache. Don't set `+"`model`"+` on a fork — a different model can't reuse the parent's cache. Pass a short `+"`name`"+` (one or two words, lowercase) so the user can see the fork in the teams panel and steer it mid-run.

**Don't peek.** The tool result includes an `+"`output_file`"+` path — do not %s or tail it unless the user explicitly asks for a progress check. You get a completion notification; trust it. Reading the transcript mid-flight pulls the fork's tool noise into your context, which defeats the point of forking.

**Don't race.** After launching, you know nothing about what the fork found. Never fabricate or predict fork results in any format — not as prose, summary, or structured output. The notification arrives as a user-role message in a later turn; it is never something you write yourself. If the user asks a follow-up before the notification lands, tell them the fork is still running — give status, not a guess.

**Writing a fork prompt.** Since the fork inherits your context, the prompt is a *directive* — what to do, not what the situation is. Be specific about scope: what's in, what's out, what another agent is handling. Don't re-explain background.`, FileReadToolName)
}

func getCurrentExamples() string {
	return fmt.Sprintf(`Example usage:

<example_agent_descriptions>
"test-runner": use this agent after you are done writing code to run tests
"greeting-responder": use this agent to respond to user greetings with a friendly joke
</example_agent_descriptions>

<example>
user: "Please write a function that checks if a number is prime"
assistant: I'm going to use the %s tool to write the following code:
<code>
function isPrime(n) {
  if (n <= 1) return false
  for (let i = 2; i * i <= n; i++) {
    if (n %% i === 0) return false
  }
  return true
}
</code>
<commentary>
Since a significant piece of code was written and the task was completed, now use the test-runner agent to run the tests
</commentary>
assistant: Uses the %s tool to launch the test-runner agent
</example>

<example>
user: "Hello"
<commentary>
Since the user is greeting, use the greeting-responder agent to respond with a friendly joke
</commentary>
assistant: "I'm going to use the %s tool to launch the greeting-responder agent"
</example>`, FileWriteToolName, AgentToolName, AgentToolName)
}
