package toolpool

import (
	"os"
	"strings"
	"time"

	"goc/commands"
)

// ─── TodoWrite ───────────────────────────────────────────────────────────────

const todoWritePrompt = `Use this tool to create and manage a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool
Use this tool proactively in these scenarios:

1. Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
2. Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
3. User explicitly requests todo list - When the user directly asks you to use the todo list
4. User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
5. After receiving new instructions - Immediately capture user requirements as todos
6. When you start working on a task - Mark it as in_progress BEFORE beginning work. Ideally you should only have one todo as in_progress at a time
7. After completing a task - Mark it as completed and add any new follow-up tasks discovered during implementation

## When NOT to Use This Tool

Skip using this tool when:
1. There is only a single, straightforward task
2. The task is trivial and tracking it provides no organizational benefit
3. The task can be completed in less than 3 trivial steps
4. The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task States and Management

1. **Task States**: Use these states to track progress:
   - pending: Task not yet started
   - in_progress: Currently working on (limit to ONE task at a time)
   - completed: Task finished successfully

2. **Task Management**:
   - Update task status in real-time as you work
   - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
   - Exactly ONE task must be in_progress at any time (not less, not more)
   - Complete current tasks before starting new ones
   - Remove tasks that are no longer relevant from the list entirely

3. **Task Completion Requirements**:
   - ONLY mark a task as completed when you have FULLY accomplished it
   - If you encounter errors, blockers, or cannot finish, keep the task as in_progress
   - When blocked, create a new task describing what needs to be resolved
   - Never mark a task as completed if:
     - Tests are failing
     - Implementation is partial
     - You encountered unresolved errors
     - You couldn't find necessary files or dependencies

When in doubt, use this tool. Being proactive with task management demonstrates attentiveness and ensures you complete all requirements successfully.`

// ─── WebFetch ─────────────────────────────────────────────────────────────────

const webFetchPrompt = `IMPORTANT: WebFetch WILL FAIL for authenticated or private URLs. Before using this tool, check if the URL points to an authenticated service (e.g. Google Docs, Confluence, Jira, GitHub). If so, look for a specialized MCP tool that provides authenticated access.

- Fetches content from a specified URL and processes it using an AI model
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to markdown
- Processes the content with the prompt using a small, fast model
- Returns the model's response about the content
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions.
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - The prompt should describe what information you want to extract from the page
  - This tool is read-only and does not modify any files
  - Results may be summarized if the content is very large
  - Includes a self-cleaning 15-minute cache for faster responses when repeatedly accessing the same URL
  - When a URL redirects to a different host, the tool will inform you and provide the redirect URL in a special format. You should then make a new WebFetch request with the redirect URL to fetch the content.
  - For GitHub URLs, prefer using the gh CLI via Bash instead (e.g., gh pr view, gh issue view, gh api).`

// ─── WebSearch ────────────────────────────────────────────────────────────────

// getWebSearchDescription mirrors getWebSearchPrompt in TS. It injects the
// current month and year (or CLAUDE_CODE_OVERRIDE_DATE if set).
func getWebSearchDescription() string {
	currentMonthYear := getLocalMonthYear()
	return `
- Allows Claude to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks, including links as markdown hyperlinks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

CRITICAL REQUIREMENT - You MUST follow this:
  - After answering the user's question, you MUST include a "Sources:" section at the end of your response
  - In the Sources section, list all relevant URLs from the search results as markdown hyperlinks: [Title](URL)
  - This is MANDATORY - never skip including sources in your response
  - Example format:

    [Your answer here]

    Sources:
    - [Source Title 1](https://example.com/1)
    - [Source Title 2](https://example.com/2)

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US

IMPORTANT - Use the correct year in search queries:
  - The current month is ` + currentMonthYear + `. You MUST use this year when searching for recent information, documentation, or current events.
  - Example: If the user asks for "latest React docs", search for "React documentation" with the current year, NOT last year
`
}

func getLocalMonthYear() string {
	if d := os.Getenv("CLAUDE_CODE_OVERRIDE_DATE"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t.Format("January 2006")
		}
	}
	return time.Now().Format("January 2006")
}

// ─── NotebookEdit ─────────────────────────────────────────────────────────────

const notebookEditPrompt = `Completely replaces the contents of a specific cell in a Jupyter notebook (.ipynb file) with new source. Jupyter notebooks are interactive documents that combine code, text, and visualizations, commonly used for data analysis and scientific computing. The notebook_path parameter must be an absolute path, not a relative path. The cell_number is 0-indexed. Use edit_mode=insert to add a new cell at the index specified by cell_number. Use edit_mode=delete to delete the cell at the index specified by cell_number.`

// ─── SendMessage ──────────────────────────────────────────────────────────────

// getSendMessageDescription mirrors SendMessageTool.getPrompt() in TS.
// Conditionally includes UDS cross-session section based on feature flag.
func getSendMessageDescription() string {
	udsRow := ""
	udsSection := ""
	if commands.IsEnvTruthy("CLAUDE_CODE_ENABLE_UDS_INBOX") {
		udsRow = `
| ` + "`\"uds:/path/to.sock\"`" + ` | Local Claude session's socket (same machine; use ` + "`ListPeers`" + `) |
| ` + "`\"bridge:session_...\"`" + ` | Remote Control peer session (cross-machine; use ` + "`ListPeers`" + `) |`
		udsSection = `

## Cross-session

Use ` + "`ListPeers`" + ` to discover targets, then:

` + "```json\n" + `{"to": "uds:/tmp/cc-socks/1234.sock", "message": "check if tests pass over there"}
{"to": "bridge:session_01AbCd...", "message": "what branch are you on?"}
` + "```" + `

A listed peer is alive and will process your message — no "busy" state; messages enqueue and drain at the receiver's next tool round. Your message arrives wrapped as ` + "`<cross-session-message from=\"...\">`" + `. **To reply to an incoming message, copy its ` + "`from`" + ` attribute as your ` + "`to`" + `.**`
	}
	return `
# SendMessage

Send a message to another agent.

` + "```json\n" + `{"to": "researcher", "summary": "assign task 1", "message": "start on task #1"}
` + "```" + `

| ` + "`to`" + ` | |
|---|---|
| ` + "`\"researcher\"`" + ` | Teammate by name |
| ` + "`\"*\"`" + ` | Broadcast to all teammates — expensive (linear in team size), use only when everyone genuinely needs it |` + udsRow + `

Your plain text output is NOT visible to other agents — to communicate, you MUST call this tool. Messages from teammates are delivered automatically; you don't check an inbox. Refer to teammates by name, never by UUID. When relaying, don't quote the original — it's already rendered to the user.` + udsSection
}

// ─── EnterPlanMode ────────────────────────────────────────────────────────────

// getEnterPlanModeDescription mirrors getEnterPlanModeToolPrompt() in TS.
func getEnterPlanModeDescription() string {
	if strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" {
		return enterPlanModePromptAnt
	}
	return enterPlanModePromptExternal
}

const enterPlanModePromptExternal = `Use this tool proactively when you're about to start a non-trivial implementation task. Getting user sign-off on your approach before writing code prevents wasted effort and ensures alignment. This tool transitions you into plan mode where you can explore the codebase and design an implementation approach for user approval.

## When to Use This Tool

**Prefer using EnterPlanMode** for implementation tasks unless they're simple. Use it when ANY of these conditions apply:

1. **New Feature Implementation**: Adding meaningful new functionality
   - Example: "Add a logout button" - where should it go? What should happen on click?
   - Example: "Add form validation" - what rules? What error messages?

2. **Multiple Valid Approaches**: The task can be solved in several different ways
   - Example: "Add caching to the API" - could use Redis, in-memory, file-based, etc.
   - Example: "Improve performance" - many optimization strategies possible

3. **Code Modifications**: Changes that affect existing behavior or structure
   - Example: "Update the login flow" - what exactly should change?
   - Example: "Refactor this component" - what's the target architecture?

4. **Architectural Decisions**: The task requires choosing between patterns or technologies
   - Example: "Add real-time updates" - WebSockets vs SSE vs polling
   - Example: "Implement state management" - Redux vs Context vs custom solution

5. **Multi-File Changes**: The task will likely touch more than 2-3 files
   - Example: "Refactor the authentication system"
   - Example: "Add a new API endpoint with tests"

6. **Unclear Requirements**: You need to explore before understanding the full scope
   - Example: "Make the app faster" - need to profile and identify bottlenecks
   - Example: "Fix the bug in checkout" - need to investigate root cause

7. **User Preferences Matter**: The implementation could reasonably go multiple ways
   - If you would use AskUserQuestion to clarify the approach, use EnterPlanMode instead
   - Plan mode lets you explore first, then present options with context

## When NOT to Use This Tool

Only skip EnterPlanMode for simple tasks:
- Single-line or few-line fixes (typos, obvious bugs, small tweaks)
- Adding a single function with clear requirements
- Tasks where the user has given very specific, detailed instructions
- Pure research/exploration tasks (use the Agent tool with explore agent instead)

## What Happens in Plan Mode

In plan mode, you'll:
1. Thoroughly explore the codebase using Glob, Grep, and Read tools
2. Understand existing patterns and architecture
3. Design an implementation approach
4. Present your plan to the user for approval
5. Use AskUserQuestion if you need to clarify approaches
6. Exit plan mode with ExitPlanMode when ready to implement

## Examples

### GOOD - Use EnterPlanMode:
User: "Add user authentication to the app"
- Requires architectural decisions (session vs JWT, where to store tokens, middleware structure)

User: "Optimize the database queries"
- Multiple approaches possible, need to profile first, significant impact

User: "Implement dark mode"
- Architectural decision on theme system, affects many components

User: "Add a delete button to the user profile"
- Seems simple but involves: where to place it, confirmation dialog, API call, error handling, state updates

User: "Update the error handling in the API"
- Affects multiple files, user should approve the approach

### BAD - Don't use EnterPlanMode:
User: "Fix the typo in the README"
- Straightforward, no planning needed

User: "Add a console.log to debug this function"
- Simple, obvious implementation

User: "What files handle routing?"
- Research task, not implementation planning

## Important Notes

- This tool REQUIRES user approval - they must consent to entering plan mode
- If unsure whether to use it, err on the side of planning - it's better to get alignment upfront than to redo work
- Users appreciate being consulted before significant changes are made to their codebase`

const enterPlanModePromptAnt = `Use this tool when a task has genuine ambiguity about the right approach and getting user input before coding would prevent significant rework. This tool transitions you into plan mode where you can explore the codebase and design an implementation approach for user approval.

## When to Use This Tool

Plan mode is valuable when the implementation approach is genuinely unclear. Use it when:

1. **Significant Architectural Ambiguity**: Multiple reasonable approaches exist and the choice meaningfully affects the codebase
   - Example: "Add caching to the API" - Redis vs in-memory vs file-based
   - Example: "Add real-time updates" - WebSockets vs SSE vs polling

2. **Unclear Requirements**: You need to explore and clarify before you can make progress
   - Example: "Make the app faster" - need to profile and identify bottlenecks
   - Example: "Refactor this module" - need to understand what the target architecture should be

3. **High-Impact Restructuring**: The task will significantly restructure existing code and getting buy-in first reduces risk
   - Example: "Redesign the authentication system"
   - Example: "Migrate from one state management approach to another"

## When NOT to Use This Tool

Skip plan mode when you can reasonably infer the right approach:
- The task is straightforward even if it touches multiple files
- The user's request is specific enough that the implementation path is clear
- You're adding a feature with an obvious implementation pattern (e.g., adding a button, a new endpoint following existing conventions)
- Bug fixes where the fix is clear once you understand the bug
- Research/exploration tasks (use the Agent tool instead)
- The user says something like "can we work on X" or "let's do X" — just get started

When in doubt, prefer starting work and using AskUserQuestion for specific questions over entering a full planning phase.

## What Happens in Plan Mode

In plan mode, you'll:
1. Thoroughly explore the codebase using Glob, Grep, and Read tools
2. Understand existing patterns and architecture
3. Design an implementation approach
4. Present your plan to the user for approval
5. Use AskUserQuestion if you need to clarify approaches
6. Exit plan mode with ExitPlanMode when ready to implement

## Examples

### GOOD - Use EnterPlanMode:
User: "Add user authentication to the app"
- Genuinely ambiguous: session vs JWT, where to store tokens, middleware structure

User: "Redesign the data pipeline"
- Major restructuring where the wrong approach wastes significant effort

### BAD - Don't use EnterPlanMode:
User: "Add a delete button to the user profile"
- Implementation path is clear; just do it

User: "Can we work on the search feature?"
- User wants to get started, not plan

User: "Update the error handling in the API"
- Start working; ask specific questions if needed

User: "Fix the typo in the README"
- Straightforward, no planning needed

## Important Notes

- This tool REQUIRES user approval - they must consent to entering plan mode`

// ─── ExitPlanMode ─────────────────────────────────────────────────────────────

const exitPlanModePrompt = `Use this tool when you are in plan mode and have finished writing your plan to the plan file and are ready for user approval.

## How This Tool Works
- You should have already written your plan to the plan file specified in the plan mode system message
- This tool does NOT take the plan content as a parameter - it will read the plan from the file you wrote
- This tool simply signals that you're done planning and ready for the user to review and approve
- The user will see the contents of your plan file when they review it

## When to Use This Tool
IMPORTANT: Only use this tool when the task requires planning the implementation steps of a task that requires writing code. For research tasks where you're gathering information, searching files, reading files or in general trying to understand the codebase - do NOT use this tool.

## Before Using This Tool
Ensure your plan is complete and unambiguous:
- If you have unresolved questions about requirements or approach, use AskUserQuestion first (in earlier phases)
- Once your plan is finalized, use THIS tool to request approval

**Important:** Do NOT use AskUserQuestion to ask "Is this plan okay?" or "Should I proceed?" - that's exactly what THIS tool does. ExitPlanMode inherently requests user approval of your plan.

## Examples

1. Initial task: "Search for and understand the implementation of vim mode in the codebase" - Do not use the exit plan mode tool because you are not planning the implementation steps of a task.
2. Initial task: "Help me implement yank mode for vim" - Use the exit plan mode tool after you have finished planning the implementation steps of the task.
3. Initial task: "Add a new feature to handle user authentication" - If unsure about auth method (OAuth, JWT, etc.), use AskUserQuestion first, then use exit plan mode tool after clarifying the approach.`

// ─── EnterWorktree ────────────────────────────────────────────────────────────

const enterWorktreePrompt = `Use this tool ONLY when the user explicitly asks to work in a worktree. This tool creates an isolated git worktree and switches the current session into it.

## When to Use

- The user explicitly says "worktree" (e.g., "start a worktree", "work in a worktree", "create a worktree", "use a worktree")

## When NOT to Use

- The user asks to create a branch, switch branches, or work on a different branch — use git commands instead
- The user asks to fix a bug or work on a feature — use normal git workflow unless they specifically mention worktrees
- Never use this tool unless the user explicitly mentions "worktree"

## Requirements

- Must be in a git repository, OR have WorktreeCreate/WorktreeRemove hooks configured in settings.json
- Must not already be in a worktree

## Behavior

- In a git repository: creates a new git worktree inside ` + "`.harness/worktrees/`" + ` with a new branch based on HEAD
- Outside a git repository: delegates to WorktreeCreate/WorktreeRemove hooks for VCS-agnostic isolation
- Switches the session's working directory to the new worktree
- Use ExitWorktree to leave the worktree mid-session (keep or remove). On session exit, if still in the worktree, the user will be prompted to keep or remove it

## Parameters

- ` + "`name`" + ` (optional): A name for the worktree. If not provided, a random name is generated.`

// ─── ExitWorktree ─────────────────────────────────────────────────────────────

const exitWorktreePrompt = `Exit a worktree session created by EnterWorktree and return the session to the original working directory.

## Scope

This tool ONLY operates on worktrees created by EnterWorktree in this session. It will NOT touch:
- Worktrees you created manually with ` + "`git worktree add`" + `
- Worktrees from a previous session (even if created by EnterWorktree then)
- The directory you're in if EnterWorktree was never called

If called outside an EnterWorktree session, the tool is a **no-op**: it reports that no worktree session is active and takes no action. Filesystem state is unchanged.

## When to Use

- The user explicitly asks to "exit the worktree", "leave the worktree", "go back", or otherwise end the worktree session
- Do NOT call this proactively — only when the user asks

## Parameters

- ` + "`action`" + ` (required): ` + "`\"keep\"`" + ` or ` + "`\"remove\"`" + `
  - ` + "`\"keep\"`" + ` — leave the worktree directory and branch intact on disk. Use this if the user wants to come back to the work later, or if there are changes to preserve.
  - ` + "`\"remove\"`" + ` — delete the worktree directory and its branch. Use this for a clean exit when the work is done or abandoned.
- ` + "`discard_changes`" + ` (optional, default false): only meaningful with ` + "`action: \"remove\"`" + `. If the worktree has uncommitted files or commits not on the original branch, the tool will REFUSE to remove it unless this is set to ` + "`true`" + `. If the tool returns an error listing changes, confirm with the user before re-invoking with ` + "`discard_changes: true`" + `.

## Behavior

- Restores the session's working directory to where it was before EnterWorktree
- Clears CWD-dependent caches (system prompt sections, memory files, plans directory) so the session state reflects the original directory
- If a tmux session was attached to the worktree: killed on ` + "`remove`" + `, left running on ` + "`keep`" + ` (its name is returned so the user can reattach)
- Once exited, EnterWorktree can be called again to create a fresh worktree`

// ─── CronCreate ───────────────────────────────────────────────────────────────

// getCronCreateDescription mirrors buildCronCreatePrompt in TS.
func getCronCreateDescription() string {
	durable := isCronDurableEnabled()
	section := "## Session-only\n\nJobs live only in this Claude session — nothing is written to disk, and the job is gone when Claude exits."
	runtimeExtra := ""
	if durable {
		section = `## Durability

By default (durable: false) the job lives only in this Claude session — nothing is written to disk, and the job is gone when Claude exits. Pass durable: true to write to .harness/scheduled_tasks.json so the job survives restarts. Only use durable: true when the user explicitly asks for the task to persist ("keep doing this every day", "set this up permanently"). Most "remind me in 5 minutes" / "check back in an hour" requests should stay session-only.`
		runtimeExtra = ` Durable jobs persist to .harness/scheduled_tasks.json and survive session restarts — on next launch they resume automatically. One-shot durable tasks that were missed while the REPL was closed are surfaced for catch-up. Session-only jobs die with the process.`
	}

	return `Schedule a prompt to be enqueued at a future time. Use for both recurring schedules and one-shot reminders.

Uses standard 5-field cron in the user's local timezone: minute hour day-of-month month day-of-week. "0 9 * * *" means 9am local — no timezone conversion needed.

## One-shot tasks (recurring: false)

For "remind me at X" or "at <time>, do Y" requests — fire once then auto-delete.
Pin minute/hour/day-of-month/month to specific values:
  "remind me at 2:30pm today to check the deploy" → cron: "30 14 <today_dom> <today_month> *", recurring: false
  "tomorrow morning, run the smoke test" → cron: "57 8 <tomorrow_dom> <tomorrow_month> *", recurring: false

## Recurring jobs (recurring: true, the default)

For "every N minutes" / "every hour" / "weekdays at 9am" requests:
  "*/5 * * * *" (every 5 min), "0 * * * *" (hourly), "0 9 * * 1-5" (weekdays at 9am local)

## Avoid the :00 and :30 minute marks when the task allows it

Every user who asks for "9am" gets ` + "`0 9`" + `, and every user who asks for "hourly" gets ` + "`0 *`" + ` — which means requests from across the planet land on the API at the same instant. When the user's request is approximate, pick a minute that is NOT 0 or 30:
  "every morning around 9" → "57 8 * * *" or "3 9 * * *" (not "0 9 * * *")
  "hourly" → "7 * * * *" (not "0 * * * *")
  "in an hour or so, remind me to..." → pick whatever minute you land on, don't round

Only use minute 0 or 30 when the user names that exact time and clearly means it ("at 9:00 sharp", "at half past", coordinating with a meeting). When in doubt, nudge a few minutes early or late — the user will not notice, and the fleet will.

` + section + `

## Runtime behavior

Jobs only fire while the REPL is idle (not mid-query).` + runtimeExtra + ` The scheduler adds a small deterministic jitter on top of whatever you pick: recurring tasks fire up to 10% of their period late (max 15 min); one-shot tasks landing on :00 or :30 fire up to 90 s early. Picking an off-minute is still the bigger lever.

Recurring tasks auto-expire after 30 days — they fire one final time, then are deleted. This bounds session lifetime. Tell the user about the 30-day limit when scheduling recurring jobs.

Returns a job ID you can pass to CronDelete.`
}

// getCronDeleteDescription mirrors buildCronDeletePrompt in TS.
func getCronDeleteDescription() string {
	if isCronDurableEnabled() {
		return "Cancel a cron job previously scheduled with CronCreate. Removes it from .harness/scheduled_tasks.json (durable jobs) or the in-memory session store (session-only jobs)."
	}
	return "Cancel a cron job previously scheduled with CronCreate. Removes it from the in-memory session store."
}

// getCronListDescription mirrors buildCronListPrompt in TS.
func getCronListDescription() string {
	if isCronDurableEnabled() {
		return "List all cron jobs scheduled via CronCreate, both durable (.harness/scheduled_tasks.json) and session-only."
	}
	return "List all cron jobs scheduled via CronCreate in this session."
}

func isCronDurableEnabled() bool {
	return !commands.IsEnvTruthy("CLAUDE_CODE_DISABLE_DURABLE_CRON")
}

// ─── TaskCreate ───────────────────────────────────────────────────────────────

// getTaskCreateDescription mirrors TaskCreateTool.getPrompt() in TS.
func getTaskCreateDescription() string {
	swarmTip := ""
	swarmTeammate := ""
	if commands.AgentSwarmsEnabled() {
		swarmTip = " and potentially assigned to teammates"
		swarmTeammate = `
- Include enough detail in the description for another agent to understand and complete the task
- New tasks are created with status 'pending' and no owner - use TaskUpdate with the ` + "`owner`" + ` parameter to assign them`
	}
	return `IMPORTANT: This tool creates structured todo/task list entries for tracking progress. It is NOT the same as the Agent tool (which launches subprocesses to execute work). When instructed to "create a task" or "create tasks" for a checklist or tracking purpose, use TaskCreate — not Agent.

Use this tool to create a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool

Use this tool proactively in these scenarios:

- Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
- Non-trivial and complex tasks - Tasks that require careful planning or multiple operations` + swarmTip + `
- Plan mode - When using plan mode, create a task list to track the work
- User explicitly requests todo list - When the user directly asks you to use the todo list
- User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
- After receiving new instructions - Immediately capture user requirements as tasks
- When you start working on a task - Mark it as in_progress BEFORE beginning work
- After completing a task - Mark it as completed and add any new follow-up tasks discovered during implementation

## When NOT to Use This Tool

Skip using this tool when:
- There is only a single, straightforward task
- The task is trivial and tracking it provides no organizational benefit
- The task can be completed in less than 3 trivial steps
- The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task Fields

- **subject**: A brief, actionable title in imperative form (e.g., "Fix authentication bug in login flow")
- **description**: What needs to be done
- **activeForm** (optional): Present continuous form shown in the spinner when the task is in_progress (e.g., "Fixing authentication bug"). If omitted, the spinner shows the subject instead.

All tasks are created with status ` + "`pending`" + `.

## Tips

- Create tasks with clear, specific subjects that describe the outcome
- After creating tasks, use TaskUpdate to set up dependencies (blocks/blockedBy) if needed` + swarmTeammate + `
- Check TaskList first to avoid creating duplicate tasks`
}

// ─── TaskGet ──────────────────────────────────────────────────────────────────

const taskGetPrompt = `Use this tool to retrieve a task by its ID from the task list.

## When to Use This Tool

- When you need the full description and context before starting work on a task
- To understand task dependencies (what it blocks, what blocks it)
- After being assigned a task, to get complete requirements

## Output

Returns full task details:
- **subject**: Task title
- **description**: Detailed requirements and context
- **status**: 'pending', 'in_progress', or 'completed'
- **blocks**: Tasks waiting on this one to complete
- **blockedBy**: Tasks that must complete before this one can start

## Tips

- After fetching a task, verify its blockedBy list is empty before beginning work.
- Use TaskList to see all tasks in summary form.`

// ─── TaskList ─────────────────────────────────────────────────────────────────

// getTaskListDescription mirrors TaskListTool.getPrompt() in TS.
func getTaskListDescription() string {
	assignTeammates := ""
	teammateWorkflow := ""
	if commands.AgentSwarmsEnabled() {
		assignTeammates = `
- Before assigning tasks to teammates, to see what's available`
		teammateWorkflow = `

## Teammate Workflow

When working as a teammate:
1. After completing your current task, call TaskList to find available work
2. Look for tasks with status 'pending', no owner, and empty blockedBy
3. **Prefer tasks in ID order** (lowest ID first) when multiple tasks are available, as earlier tasks often set up context for later ones
4. Claim an available task using TaskUpdate (set ` + "`owner`" + ` to your name), or wait for leader assignment
5. If blocked, focus on unblocking tasks or notify the team lead`
	}
	return `Use this tool to list all tasks in the task list.

## When to Use This Tool

- To see what tasks are available to work on (status: 'pending', no owner, not blocked)
- To check overall progress on the project
- To find tasks that are blocked and need dependencies resolved` + assignTeammates + `
- After completing a task, to check for newly unblocked work or claim the next available task
- **Prefer working on tasks in ID order** (lowest ID first) when multiple tasks are available, as earlier tasks often set up context for later ones

## Output

Returns a summary of each task:
- **id**: Task identifier (use with TaskGet, TaskUpdate)
- **subject**: Brief description of the task
- **status**: 'pending', 'in_progress', or 'completed'
- **owner**: Agent ID if assigned, empty if available
- **blockedBy**: List of open task IDs that must be resolved first (tasks with blockedBy cannot be claimed until dependencies resolve)

Use TaskGet with a specific task ID to view full details including description and comments.` + teammateWorkflow
}

// ─── TaskUpdate ───────────────────────────────────────────────────────────────

const taskUpdatePrompt = `Use this tool to update a task in the task list.

## When to Use This Tool

**Mark tasks as resolved:**
- When you have completed the work described in a task
- When a task is no longer needed or has been superseded
- IMPORTANT: Always mark your assigned tasks as resolved when you finish them
- After resolving, call TaskList to find your next task

- ONLY mark a task as completed when you have FULLY accomplished it
- If you encounter errors, blockers, or cannot finish, keep the task as in_progress
- When blocked, create a new task describing what needs to be resolved
- Never mark a task as completed if:
  - Tests are failing
  - Implementation is partial
  - You encountered unresolved errors
  - You couldn't find necessary files or dependencies

**Delete tasks:**
- When a task is no longer relevant or was created in error
- Setting status to ` + "`deleted`" + ` permanently removes the task

**Update task details:**
- When requirements change or become clearer
- When establishing dependencies between tasks

## Fields You Can Update

- **status**: The task status (see Status Workflow below)
- **subject**: Change the task title (imperative form, e.g., "Run tests")
- **description**: Change the task description
- **activeForm**: Present continuous form shown in spinner when in_progress (e.g., "Running tests")
- **owner**: Change the task owner (agent name)
- **metadata**: Merge metadata keys into the task (set a key to null to delete it)
- **addBlocks**: Mark tasks that cannot start until this one completes
- **addBlockedBy**: Mark tasks that must complete before this one can start

## Status Workflow

Status progresses: ` + "`pending`" + ` → ` + "`in_progress`" + ` → ` + "`completed`" + `

Use ` + "`deleted`" + ` to permanently remove a task.

## Staleness

Make sure to read a task's latest state using TaskGet before updating it.

## Examples

Mark task as in progress when starting work:
` + "```json\n" + `{"taskId": "1", "status": "in_progress"}
` + "```" + `

Mark task as completed after finishing work:
` + "```json\n" + `{"taskId": "1", "status": "completed"}
` + "```" + `

Delete a task:
` + "```json\n" + `{"taskId": "1", "status": "deleted"}
` + "```" + `

Claim a task by setting owner:
` + "```json\n" + `{"taskId": "1", "owner": "my-name"}
` + "```" + `

Set up task dependencies:
` + "```json\n" + `{"taskId": "2", "addBlockedBy": ["1"]}
` + "```" + ``

// ─── TaskOutput ───────────────────────────────────────────────────────────────

const taskOutputPrompt = `DEPRECATED: Prefer using the Read tool on the task's output file path instead. Background tasks return their output file path in the tool result, and you receive a <task-notification> with the same path when the task completes — Read that file directly.

- Retrieves output from a running or completed task (background shell, agent, or remote session)
- Takes a task_id parameter identifying the task
- Returns the task output along with status information
- Use block=true (default) to wait for task completion
- Use block=false for non-blocking check of current status
- Task IDs can be found using the /tasks command
- Works with all task types: background shells, async agents, and remote sessions`

// ─── TaskStop ─────────────────────────────────────────────────────────────────

const taskStopPrompt = `

- Stops a running background task by its ID
- Takes a task_id parameter identifying the task to stop
- Returns a success or failure status
- Use this tool when you need to terminate a long-running task
`

// ─── PowerShell ───────────────────────────────────────────────────────────────

// getPowerShellDescription mirrors PowerShellTool.getPrompt() in TS.
func getPowerShellDescription() string {
	edition := getPowerShellEdition()
	return `Execute a PowerShell command in the local environment.` + `
Executes a given PowerShell command with optional timeout. Working directory persists between commands; shell state (variables, functions) does not.

IMPORTANT: This tool is for terminal operations via PowerShell: git, npm, docker, and PS cmdlets. DO NOT use it for file operations (reading, writing, editing, searching, finding files) - use the specialized tools for this instead.

` + getPowerShellEditionSection(edition) + `

Before executing the command, please follow these steps:

1. Directory Verification:
   - If the command will create new directories or files, first use ` + "`Get-ChildItem`" + ` (or ` + "`ls`" + `) to verify the parent directory exists and is the correct location

2. Command Execution:
   - Always quote file paths that contain spaces with double quotes
   - Capture the output of the command.

PowerShell Syntax Notes:
   - Variables use $ prefix: $myVar = "value"
   - Escape character is backtick (` + "`" + `), not backslash
   - Use Verb-Noun cmdlet naming: Get-ChildItem, Set-Location, New-Item, Remove-Item
   - Common aliases: ls (Get-ChildItem), cd (Set-Location), cat (Get-Content), rm (Remove-Item)
   - Pipe operator | works similarly to bash but passes objects, not text
   - Use Select-Object, Where-Object, ForEach-Object for filtering and transformation
   - String interpolation: "Hello $name" or "Hello $($obj.Property)"
   - Registry access uses PSDrive prefixes: ` + "`HKLM:\\SOFTWARE\\...`" + `, ` + "`HKCU:\\...`" + ` — NOT raw ` + "`HKEY_LOCAL_MACHINE\\...`" + `
   - Environment variables: read with ` + "`$env:NAME`" + `, set with ` + "`$env:NAME = \"value\"`" + ` (NOT ` + "`Set-Variable`" + ` or bash ` + "`export`" + `)
   - Call native exe with spaces in path via call operator: ` + "`& \"C:\\Program Files\\App\\app.exe\" arg1 arg2`" + `

Interactive and blocking commands (will hang — this tool runs with -NonInteractive):
   - NEVER use ` + "`Read-Host`" + `, ` + "`Get-Credential`" + `, ` + "`Out-GridView`" + `, ` + "`$Host.UI.PromptForChoice`" + `, or ` + "`pause`" + `
   - Destructive cmdlets (` + "`Remove-Item`" + `, ` + "`Stop-Process`" + `, ` + "`Clear-Content`" + `, etc.) may prompt for confirmation. Add ` + "`-Confirm:$false`" + ` when you intend the action to proceed. Use ` + "`-Force`" + ` for read-only/hidden items.
   - Never use ` + "`git rebase -i`" + `, ` + "`git add -i`" + `, or other commands that open an interactive editor

Passing multiline strings (commit messages, file content) to native executables:
   - Use a single-quoted here-string so PowerShell does not expand ` + "`$`" + ` or backticks inside. The closing ` + "`'@`" + ` MUST be at column 0 (no leading whitespace) on its own line — indenting it is a parse error

Usage notes:
  - The command argument is required.
  - You can specify an optional timeout in milliseconds. If not specified, commands will timeout after default.
  - It is very helpful if you write a clear, concise description of what this command does.
  - You can use the ` + "`run_in_background`" + ` parameter to run the command in the background. Only use this if you don't need the result immediately and are OK being notified when the command completes later.
  - Avoid using PowerShell to run commands that have dedicated tools, unless explicitly instructed:
    - File search: Use GlobTool (NOT Get-ChildItem -Recurse)
    - Content search: Use GrepTool (NOT Select-String)
    - Read files: Use Read (NOT Get-Content)
    - Edit files: Use Edit
    - Write files: Use Write (NOT Set-Content/Out-File)
    - Communication: Output text directly (NOT Write-Output/Write-Host)
  - When issuing multiple commands:
    - If the commands are independent and can run in parallel, make multiple PowerShell tool calls in a single message.
    - If the commands depend on each other and must run sequentially, chain them in a single PowerShell call (see edition-specific chaining syntax above).
    - Use ` + "`;`" + ` only when you need to run commands sequentially but don't care if earlier commands fail.
    - DO NOT use newlines to separate commands (newlines are ok in quoted strings and here-strings)
  - Do NOT prefix commands with ` + "`cd`" + ` or ` + "`Set-Location`" + ` -- the working directory is already set to the correct project directory automatically.
  - For git commands:
    - Prefer to create a new commit rather than amending an existing commit.
    - Before running destructive operations (e.g., git reset --hard, git push --force, git checkout --), consider whether there is a safer alternative that achieves the same goal. Only use destructive operations when they are truly the best approach.
    - Never skip hooks (--no-verify) or bypass signing (--no-gpg-sign, -c commit.gpgsign=false) unless the user has explicitly asked for it. If a hook fails, investigate and fix the underlying issue.`
}

func getPowerShellEdition() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("POWERSHELL_EDITION")))
}

func getPowerShellEditionSection(edition string) string {
	switch edition {
	case "core":
		return `PowerShell edition: PowerShell 7+ (pwsh)
   - Pipeline chain operators ` + "`&&`" + ` and ` + "`||`" + ` ARE available and work like bash. Prefer ` + "`cmd1 && cmd2`" + ` over ` + "`cmd1; cmd2`" + ` when cmd2 should only run if cmd1 succeeds.
   - Ternary (` + "`$cond ? $a : $b`" + `), null-coalescing (` + "`??`" + `), and null-conditional (` + "`?.`" + `) operators are available.
   - Default file encoding is UTF-8 without BOM.`
	case "desktop":
		return `PowerShell edition: Windows PowerShell 5.1 (powershell.exe)
   - Pipeline chain operators ` + "`&&`" + ` and ` + "`||`" + ` are NOT available — they cause a parser error. To run B only if A succeeds: ` + "`A; if ($?) { B }`" + `. To chain unconditionally: ` + "`A; B`" + `.
   - Ternary (` + "`?:`" + `), null-coalescing (` + "`??`" + `), and null-conditional (` + "`?.`" + `) operators are NOT available. Use ` + "`if/else`" + ` and explicit ` + "`$null -eq`" + ` checks instead.
   - Avoid ` + "`2>&1`" + ` on native executables. In 5.1, redirecting a native command's stderr inside PowerShell wraps each line in an ErrorRecord (NativeCommandError) and sets ` + "`$?`" + ` to ` + "`$false`" + ` even when the exe returned exit code 0. stderr is already captured for you — don't redirect it.
   - Default file encoding is UTF-16 LE (with BOM). When writing files other tools will read, pass ` + "`-Encoding utf8`" + ` to ` + "`Out-File`" + `/` + "`Set-Content`" + `.
   - ` + "`ConvertFrom-Json`" + ` returns a PSCustomObject, not a hashtable. ` + "`-AsHashtable`" + ` is not available.`
	default:
		return `PowerShell edition: unknown — assume Windows PowerShell 5.1 for compatibility
   - Do NOT use ` + "`&&`" + `, ` + "`||`" + `, ternary ` + "`?:`" + `, null-coalescing ` + "`??`" + `, or null-conditional ` + "`?.`" + `. These are PowerShell 7+ only and parser-error on 5.1.
   - To chain commands conditionally: ` + "`A; if ($?) { B }`" + `. Unconditionally: ` + "`A; B`" + `.`
	}
}

// ─── ListMcpResourcesTool ─────────────────────────────────────────────────────

const listMcpResourcesPrompt = `

List available resources from configured MCP servers.
Each returned resource will include all standard MCP resource fields plus a 'server' field
indicating which server the resource belongs to.

Parameters:
- server (optional): The name of a specific MCP server to get resources from. If not provided,
  resources from all servers will be returned.

`

// ─── ReadMcpResourceTool ──────────────────────────────────────────────────────

const readMcpResourcePrompt = `

Reads a specific resource from an MCP server, identified by server name and resource URI.

Parameters:
- server (required): The name of the MCP server from which to read the resource
- uri (required): The URI of the resource to read

`
