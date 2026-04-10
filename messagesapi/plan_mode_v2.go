package messagesapi

import (
	"fmt"
	"strings"
)

// Mirrors src/utils/messages.ts PLAN_PHASE4_* + getPlanPhase4Section (tengu_pewter_ledger arms).
const (
	planPhase4Control = `### Phase 4: Final Plan
Goal: Write your final plan to the plan file (the only file you can edit).
- Begin with a **Context** section: explain why this change is being made — the problem or need it addresses, what prompted it, and the intended outcome
- Include only your recommended approach, not all alternatives
- Ensure that the plan file is concise enough to scan quickly, but detailed enough to execute effectively
- Include the paths of critical files to be modified
- Reference existing functions and utilities you found that should be reused, with their file paths
- Include a verification section describing how to test the changes end-to-end (run the code, use MCP tools, run tests)`

	planPhase4Trim = `### Phase 4: Final Plan
Goal: Write your final plan to the plan file (the only file you can edit).
- One-line **Context**: what is being changed and why
- Include only your recommended approach, not all alternatives
- List the paths of files to be modified
- Reference existing functions and utilities to reuse, with their file paths
- End with **Verification**: the single command to run to confirm the change works (no numbered test procedures)`

	planPhase4Cut = `### Phase 4: Final Plan
Goal: Write your final plan to the plan file (the only file you can edit).
- Do NOT write a Context or Background section. The user just told you what they want.
- List the paths of files to be modified and what changes in each (one line per file)
- Reference existing functions and utilities to reuse, with their file paths
- End with **Verification**: the single command that confirms the change works
- Most good plans are under 40 lines. Prose is a sign you are padding.`

	planPhase4Cap = `### Phase 4: Final Plan
Goal: Write your final plan to the plan file (the only file you can edit).
- Do NOT write a Context, Background, or Overview section. The user just told you what they want.
- Do NOT restate the user's request. Do NOT write prose paragraphs.
- List the paths of files to be modified and what changes in each (one bullet per file)
- Reference existing functions to reuse, with their file:line
- End with the single verification command
- **Hard limit: 40 lines.** If the plan is longer, delete prose — not file paths.`
)

const (
	exploreAgentType = "Explore"
	planAgentType    = "Plan"
)

func planPhase4Section(variant string) string {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "trim":
		return planPhase4Trim
	case "cut":
		return planPhase4Cut
	case "cap":
		return planPhase4Cap
	default:
		return planPhase4Control
	}
}

func clampPlanAgentCount(n int) int {
	if n <= 0 {
		return 1
	}
	if n > 10 {
		return 10
	}
	return n
}

func clampExploreAgentCount(n int) int {
	if n <= 0 {
		return 3
	}
	if n > 10 {
		return 10
	}
	return n
}

// readOnlyToolNamesForPlanInterview mirrors getReadOnlyToolNames() when allowedTools is not applied (Gou path).
func readOnlyToolNamesForPlanInterview(o Options) string {
	if o.PlanModeEmbeddedSearchTools {
		return fmt.Sprintf("%s, `find`, `grep`", fileReadToolName)
	}
	return fmt.Sprintf("%s, Glob, Grep", fileReadToolName)
}

func planModeV2FullUserMessageContent(planExists bool, planFilePath string, o Options) string {
	exploreN := clampExploreAgentCount(o.PlanModeV2ExploreAgentCount)
	agentN := clampPlanAgentCount(o.PlanModeV2AgentCount)
	phase4 := planPhase4Section(o.PlanPhase4Variant)

	planFileInfo := planFileInfoParagraph(planExists, planFilePath)

	phase2Multi := ""
	if agentN > 1 {
		phase2Multi = fmt.Sprintf(`- **Multiple agents**: Use up to %d agents for complex tasks that benefit from different perspectives

Examples of when to use multiple agents:
- The task touches multiple parts of the codebase
- It's a large refactor or architectural change
- There are many edge cases to consider
- You'd benefit from exploring different approaches

Example perspectives by task type:
- New feature: simplicity vs performance vs maintainability
- Bug fix: root cause vs workaround vs prevention
- Refactoring: minimal change vs clean architecture
`, agentN)
	}

	return fmt.Sprintf(`Plan mode is active. The user indicated that they do not want you to execute yet -- you MUST NOT make any edits (with the exception of the plan file mentioned below), run any non-readonly tools (including changing configs or making commits), or otherwise make any changes to the system. This supercedes any other instructions you have received.

## Plan File Info:
%s
You should build your plan incrementally by writing to or editing this file. NOTE that this is the only file you are allowed to edit - other than this you are only allowed to take READ-ONLY actions.

## Plan Workflow

### Phase 1: Initial Understanding
Goal: Gain a comprehensive understanding of the user's request by reading through code and asking them questions. Critical: In this phase you should only use the %s subagent type.

1. Focus on understanding the user's request and the code associated with their request. Actively search for existing functions, utilities, and patterns that can be reused — avoid proposing new code when suitable implementations already exist.

2. **Launch up to %d %s agents IN PARALLEL** (single message, multiple tool calls) to efficiently explore the codebase.
   - Use 1 agent when the task is isolated to known files, the user provided specific file paths, or you're making a small targeted change.
   - Use multiple agents when: the scope is uncertain, multiple areas of the codebase are involved, or you need to understand existing patterns before planning.
   - Quality over quantity - %d agents maximum, but you should try to use the minimum number of agents necessary (usually just 1)
   - If using multiple agents: Provide each agent with a specific search focus or area to explore. Example: One agent searches for existing implementations, another explores related components, a third investigating testing patterns

### Phase 2: Design
Goal: Design an implementation approach.

Launch %s agent(s) to design the implementation based on the user's intent and your exploration results from Phase 1.

You can launch up to %d agent(s) in parallel.

**Guidelines:**
- **Default**: Launch at least 1 Plan agent for most tasks - it helps validate your understanding and consider alternatives
- **Skip agents**: Only for truly trivial tasks (typo fixes, single-line changes, simple renames)
%s
In the agent prompt:
- Provide comprehensive background context from Phase 1 exploration including filenames and code path traces
- Describe requirements and constraints
- Request a detailed implementation plan

### Phase 3: Review
Goal: Review the plan(s) from Phase 2 and ensure alignment with the user's intentions.
1. Read the critical files identified by agents to deepen your understanding
2. Ensure that the plans align with the user's original request
3. Use %s to clarify any remaining questions with the user

%s

### Phase 5: Call %s
At the very end of your turn, once you have asked the user questions and are happy with your final plan file - you should always call %s to indicate to the user that you are done planning.
This is critical - your turn should only end with either using the %s tool OR calling %s. Do not stop unless it's for these 2 reasons

**Important:** Use %s ONLY to clarify requirements or choose between approaches. Use %s to request plan approval. Do NOT ask about plan approval in any other way - no text questions, no AskUserQuestion. Phrases like "Is this plan okay?", "Should I proceed?", "How does this plan look?", "Any changes before we start?", or similar MUST use %s.

NOTE: At any point in time through this workflow you should feel free to ask the user questions or clarifications using the %s tool. Don't make large assumptions about user intent. The goal is to present a well researched plan to the user, and tie any loose ends before implementation begins.`,
		planFileInfo,
		exploreAgentType,
		exploreN, exploreAgentType,
		exploreN,
		planAgentType,
		agentN,
		phase2Multi,
		askUserQuestionToolName,
		phase4,
		exitPlanModeV2ToolName,
		exitPlanModeV2ToolName,
		askUserQuestionToolName,
		exitPlanModeV2ToolName,
		askUserQuestionToolName,
		exitPlanModeV2ToolName,
		exitPlanModeV2ToolName,
		askUserQuestionToolName,
	)
}
