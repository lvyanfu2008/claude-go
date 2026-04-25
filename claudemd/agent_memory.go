package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// AgentMemoryScope mirrors TS agentMemory.ts AgentMemoryScope.
type AgentMemoryScope string

const (
	AgentMemoryUser    AgentMemoryScope = "user"
	AgentMemoryProject AgentMemoryScope = "project"
	AgentMemoryLocal   AgentMemoryScope = "local"
)

// sanitizeAgentTypeForPath mirrors TS sanitizeAgentTypeForPath:
// replaces colons (invalid on Windows, used in plugin-namespaced agent types)
// with dashes.
func sanitizeAgentTypeForPath(agentType string) string {
	return strings.ReplaceAll(agentType, ":", "-")
}

// getLocalAgentMemoryDir mirrors TS getLocalAgentMemoryDir.
// Returns the local agent memory directory, which is project-specific and not checked into VCS.
func getLocalAgentMemoryDir(dirName string) string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); v != "" {
		// Persists to the mount with project namespacing.
		projectRoot := findCanonicalGitRootOrCwd()
		return filepath.Join(v, "projects", SanitizePath(projectRoot), "agent-memory-local", dirName) + string(filepath.Separator)
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".claude", "agent-memory-local", dirName) + string(filepath.Separator)
}

// findCanonicalGitRootOrCwd returns the canonical git root if available, otherwise the current directory.
func findCanonicalGitRootOrCwd() string {
	cwd, _ := os.Getwd()
	if cr := ResolveCanonicalGitRoot(cwd); cr != "" {
		return cr
	}
	return cwd
}

// GetAgentMemoryDir mirrors TS getAgentMemoryDir.
// Returns the agent memory directory for a given agent type and scope.
// - 'user' scope: <memoryBase>/agent-memory/<agentType>/
// - 'project' scope: <cwd>/.claude/agent-memory/<agentType>/
// - 'local' scope: see getLocalAgentMemoryDir()
func GetAgentMemoryDir(agentType string, scope AgentMemoryScope) string {
	dirName := sanitizeAgentTypeForPath(agentType)
	switch scope {
	case AgentMemoryProject:
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".claude", "agent-memory", dirName) + string(filepath.Separator)
	case AgentMemoryLocal:
		return getLocalAgentMemoryDir(dirName)
	case AgentMemoryUser:
		return filepath.Join(MemoryBaseDir(), "agent-memory", dirName) + string(filepath.Separator)
	}
	return ""
}

// IsAgentMemoryPath mirrors TS isAgentMemoryPath.
// Checks if file path is within an agent memory directory (any scope).
func IsAgentMemoryPath(absolutePath string) bool {
	normalizedPath := filepath.Clean(absolutePath)
	memoryBase := MemoryBaseDir()

	// User scope: check memory base
	if strings.HasPrefix(normalizedPath, filepath.Join(memoryBase, "agent-memory")+string(filepath.Separator)) {
		return true
	}

	// Project scope: always cwd-based
	cwd, _ := os.Getwd()
	if strings.HasPrefix(normalizedPath, filepath.Join(cwd, ".claude", "agent-memory")+string(filepath.Separator)) {
		return true
	}

	// Local scope
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); v != "" {
		sep := string(filepath.Separator)
		if strings.Contains(normalizedPath, sep+"agent-memory-local"+sep) &&
			strings.HasPrefix(normalizedPath, filepath.Join(v, "projects")+sep) {
			return true
		}
	} else {
		sep := string(filepath.Separator)
		if strings.HasPrefix(normalizedPath, filepath.Join(cwd, ".claude", "agent-memory-local")+sep) {
			return true
		}
	}

	return false
}

// GetAgentMemoryEntrypoint mirrors TS getAgentMemoryEntrypoint.
// Returns the agent memory file path for a given agent type and scope.
func GetAgentMemoryEntrypoint(agentType string, scope AgentMemoryScope) string {
	return filepath.Join(GetAgentMemoryDir(agentType, scope), "MEMORY.md")
}

// GetMemoryScopeDisplay mirrors TS getMemoryScopeDisplay.
func GetMemoryScopeDisplay(memory AgentMemoryScope) string {
	switch memory {
	case AgentMemoryUser:
		return "User (" + filepath.Join(MemoryBaseDir(), "agent-memory") + "/)"
	case AgentMemoryProject:
		return "Project (.claude/agent-memory/)"
	case AgentMemoryLocal:
		return "Local (" + getLocalAgentMemoryDir("...") + ")"
	default:
		return "None"
	}
}

// LoadAgentMemoryPrompt mirrors TS loadAgentMemoryPrompt.
// Loads persistent agent memory prompt for an agent with memory enabled.
// Creates the memory directory if needed and returns a prompt with memory contents.
func LoadAgentMemoryPrompt(agentType string, scope AgentMemoryScope) string {
	var scopeNote string
	switch scope {
	case AgentMemoryUser:
		scopeNote = "- Since this memory is user-scope, keep learnings general since they apply across all projects"
	case AgentMemoryProject:
		scopeNote = "- Since this memory is project-scope and shared with your team via version control, tailor your memories to this project"
	case AgentMemoryLocal:
		scopeNote = "- Since this memory is local-scope (not checked into version control), tailor your memories to this project and machine"
	}

	memoryDir := GetAgentMemoryDir(agentType, scope)

	// Fire-and-forget: ensure the directory exists so the model can write without checking.
	_ = EnsureMemoryDirExists(memoryDir)

	coworkExtraGuidelines := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES"))
	var extraGuidelines []string
	if coworkExtraGuidelines != "" {
		extraGuidelines = append(extraGuidelines, scopeNote, coworkExtraGuidelines)
	} else {
		extraGuidelines = []string{scopeNote}
	}

	return BuildAgentMemoryPrompt(AgentMemoryPromptParams{
		DisplayName:     "Persistent Agent Memory",
		MemoryDir:       memoryDir,
		ExtraGuidelines: extraGuidelines,
	})
}

// AgentMemoryPromptParams holds parameters for building agent memory prompt.
type AgentMemoryPromptParams struct {
	DisplayName     string
	MemoryDir       string
	ExtraGuidelines []string
}

// BuildAgentMemoryPrompt mirrors TS buildMemoryPrompt from memdir.ts.
// Builds the typed-memory prompt with MEMORY.md content included.
// Used by agent memory (which has no getClaudeMds() equivalent).
func BuildAgentMemoryPrompt(params AgentMemoryPromptParams) string {
	displayName := params.DisplayName
	memoryDir := params.MemoryDir
	extraGuidelines := params.ExtraGuidelines

	// Read existing memory entrypoint
	entrypoint := filepath.Join(memoryDir, "MEMORY.md")
	entrypointContent := ""
	if data, err := os.ReadFile(entrypoint); err == nil {
		entrypointContent = string(data)
	}

	lines := BuildAgentMemoryLines(displayName, memoryDir, extraGuidelines, false)

	if strings.TrimSpace(entrypointContent) != "" {
		truncated := TruncateEntrypointContent(entrypointContent)
		lines = append(lines, "## MEMORY.md", "", truncated)
	} else {
		lines = append(lines,
			"## MEMORY.md",
			"",
			"Your MEMORY.md is currently empty. When you save new memories, they will appear here.",
		)
	}

	return strings.Join(lines, "\n")
}

// buildMemoryFrontmatterExample returns the frontmatter example section.
func buildMemoryFrontmatterExample() []string {
	return []string{
		"```markdown",
		"---",
		"name: {{memory name}}",
		"description: {{one-line description}}",
		"type: {{user, feedback, project, reference}}",
		"---",
		"",
		"{{memory content}}",
		"```",
	}
}

// buildTypesSectionIndividual returns types section content.
func buildTypesSectionIndividual() []string {
	return []string{
		"## Types of memory",
		"",
		"There are several discrete types of memory that you can store in your memory system:",
		"",
		"<types>",
		"<type>",
		"    <name>user</name>",
		"    <description>Contain information about the user's role, goals, responsibilities, and knowledge.</description>",
		"</type>",
		"<type>",
		"    <name>feedback</name>",
		"    <description>Guidance the user has given you about how to approach work.</description>",
		"</type>",
		"<type>",
		"    <name>project</name>",
		"    <description>Information about ongoing work, goals, initiatives, bugs, or incidents.</description>",
		"</type>",
		"<type>",
		"    <name>reference</name>",
		"    <description>Pointers to where information can be found in external systems.</description>",
		"</type>",
		"</types>",
	}
}

// buildWhatNotToSaveSection returns the "what NOT to save" section.
func buildWhatNotToSaveSection() []string {
	return []string{
		"## What NOT to save in memory",
		"",
		"- Code patterns, conventions, architecture, file paths, or project structure — these can be derived by reading the current project state.",
		"- Git history, recent changes, or who-changed-what — git log / git blame are authoritative.",
		"- Debugging solutions or fix recipes — the fix is in the code; the commit message has the context.",
		"- Anything already documented in CLAUDE.md files.",
		"- Ephemeral task details: in-progress work, temporary state, current conversation context.",
		"",
		"These exclusions apply even when the user explicitly asks you to save. If they ask you to save a PR list or activity summary, ask what was *surprising* or *non-obvious* about it — that is the part worth keeping.",
	}
}

// buildWhenToAccessSection returns the "when to access" section.
func buildWhenToAccessSection() []string {
	return []string{
		"## When to access memories",
		"- When memories seem relevant, or the user references prior-conversation work.",
		"- You MUST access memory when the user explicitly asks you to check, recall, or remember.",
		"- If the user says to *ignore* or *not use* memory: proceed as if MEMORY.md were empty.",
		"- Memory records can become stale over time. Before answering the user or building assumptions based solely on information in memory records, verify that the memory is still correct and up-to-date.",
		"- Before recommending from memory, verify the referenced functions/files still exist.",
	}
}

// buildTrustingRecallSection returns the "trusting recall" section.
func buildTrustingRecallSection() []string {
	return []string{
		"## Before recommending from memory",
		"",
		"A memory that names a specific function, file, or flag is a claim that it existed *when the memory was written*. It may have been renamed, removed, or never merged. Before recommending it:",
		"",
		"- If the memory names a file path: check the file exists.",
		"- If the memory names a function or flag: grep for it.",
		"- If the user is about to act on your recommendation (not just asking about history), verify first.",
		"",
		"\"The memory says X exists\" is not the same as \"X exists now.\"",
		"",
		"A memory that summarizes repo state (activity logs, architecture snapshots) is frozen in time. If the user asks about *recent* or *current* state, prefer git log or reading the code over recalling the snapshot.",
	}
}

// buildSearchingPastContextLines builds the searching past context guidance.
func buildSearchingPastContextLines(memoryDir string) []string {
	return []string{
		"## Searching past context",
		"",
		"When looking for past context:",
		"1. Search topic files in your memory directory:",
		"```",
		"grep -rn \"<search term>\" " + memoryDir + " --include=\"*.md\"",
		"```",
		"2. Session transcript logs (last resort — large files, slow):",
		"```",
		"grep -rn \"<search term>\" /path/to/project/ --include=\"*.jsonl\"",
		"```",
		"Use narrow search terms (error messages, file paths, function names) rather than broad keywords.",
		"",
	}
}

// BuildAgentMemoryLines mirrors TS buildMemoryLines from memdir.ts.
// Builds the typed-memory behavioral instructions (without MEMORY.md content).
func BuildAgentMemoryLines(
	displayName string,
	memoryDir string,
	extraGuidelines []string,
	skipIndex bool,
) []string {
	var howToSave []string
	if skipIndex {
		howToSave = []string{
			"## How to save memories",
			"",
			"Write each memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
			"",
		}
		howToSave = append(howToSave, buildMemoryFrontmatterExample()...)
		howToSave = append(howToSave,
			"",
			"- Keep the name, description, and type fields in memory files up-to-date with the content",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		)
	} else {
		howToSave = []string{
			"## How to save memories",
			"",
			"Saving a memory is a two-step process:",
			"",
			"**Step 1** — write the memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
			"",
		}
		howToSave = append(howToSave, buildMemoryFrontmatterExample()...)
		howToSave = append(howToSave,
			"",
			"**Step 2** — add a pointer to that file in `MEMORY.md`. `MEMORY.md` is an index, not a memory — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. It has no frontmatter. Never write memory content directly into `MEMORY.md`.",
			"",
			"- `MEMORY.md` is always loaded into your conversation context — lines after 200 will be truncated, so keep the index concise",
			"- Keep the name, description, and type fields in memory files up-to-date with the content",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		)
	}

	dirExistsGuidance := "This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence)."

	lines := []string{
		"# " + displayName,
		"",
		"You have a persistent, file-based memory system at `" + memoryDir + "`. " + dirExistsGuidance,
		"",
		"You should build up this memory system over time so that future conversations can have a complete picture of who the user is, how they'd like to collaborate with you, what behaviors to avoid or repeat, and the context behind the work the user gives you.",
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
	}
	lines = append(lines, buildTypesSectionIndividual()...)
	lines = append(lines, "")
	lines = append(lines, buildWhatNotToSaveSection()...)
	lines = append(lines, "")
	lines = append(lines, howToSave...)
	lines = append(lines, "")
	lines = append(lines, buildWhenToAccessSection()...)
	lines = append(lines, "")
	lines = append(lines, buildTrustingRecallSection()...)
	lines = append(lines, "",
		"## Memory and other forms of persistence",
		"Memory is one of several persistence mechanisms available to you as you assist the user in a given conversation. The distinction is often that memory can be recalled in future conversations and should not be used for persisting information that is only useful within the scope of the current conversation.",
		"- When to use or update a plan instead of memory: If you are about to start a non-trivial implementation task and would like to reach alignment with the user on your approach you should use a Plan rather than saving this information to memory. Similarly, if you already have a plan within the conversation and you have changed your approach persist that change by updating the plan rather than saving a memory.",
		"- When to use or update tasks instead of memory: When you need to break your work in current conversation into discrete steps or keep track of your progress use tasks instead of saving to memory. Tasks are great for persisting information about the work that needs to be done in the current conversation, but memory should be reserved for information that will be useful in future conversations.",
		"",
	)
	if len(extraGuidelines) > 0 {
		for _, g := range extraGuidelines {
			lines = append(lines, g)
		}
		lines = append(lines, "")
	}

	lines = append(lines, buildSearchingPastContextLines(memoryDir)...)

	return lines
}
