package memdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/claudebase"
)

// Constants from src/memdir/memdir.ts for MEMORY.md / team entrypoint truncation.
const (
	entrypointName     = "MEMORY.md"
	maxEntrypointLines = 200
	maxEntrypointBytes = 25000
	// maxEntrypointLinesStr is used in fmt.Sprintf for the team how-to-save section.
	maxEntrypointLinesStr = "200"
)

// grepToolName used in search context blocks.
const grepToolName = "Grep"

// AgentMemoryPromptParams holds parameters for building agent memory prompt.
type AgentMemoryPromptParams struct {
	DisplayName     string
	MemoryDir       string
	ExtraGuidelines []string
}

// AutoMemoryPromptOpts holds parameters for BuildAutoMemoryPrompt.
// Feature gates are resolved by the caller (commands side) and passed in.
type AutoMemoryPromptOpts struct {
	Cwd                    string
	MemorySkipIndex        bool
	KairosActive           bool
	TeamMemActive          bool
	EmbeddedSearchTools    bool
	ReplModeEnabled        bool
	MemorySearchPastContext bool
}

// TruncateEntrypointContent mirrors memdir.ts truncateEntrypointContent.
func TruncateEntrypointContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	contentLines := strings.Split(trimmed, "\n")
	lineCount := len(contentLines)
	byteCount := len(trimmed)

	wasLineTruncated := lineCount > maxEntrypointLines
	wasByteTruncated := byteCount > maxEntrypointBytes

	if !wasLineTruncated && !wasByteTruncated {
		return trimmed
	}

	truncated := trimmed
	if wasLineTruncated {
		truncated = strings.Join(contentLines[:maxEntrypointLines], "\n")
	}
	if len(truncated) > maxEntrypointBytes {
		cutAt := strings.LastIndex(truncated[:maxEntrypointBytes], "\n")
		if cutAt <= 0 {
			truncated = truncated[:maxEntrypointBytes]
		} else {
			truncated = truncated[:cutAt]
		}
	}

	reason := fmt.Sprintf("%d lines and %d bytes", lineCount, byteCount)
	if wasByteTruncated && !wasLineTruncated {
		reason = fmt.Sprintf("%d bytes (limit: %d) — index entries are too long", byteCount, maxEntrypointBytes)
	} else if wasLineTruncated && !wasByteTruncated {
		reason = fmt.Sprintf("%d lines (limit: %d)", lineCount, maxEntrypointLines)
	}

	return truncated + fmt.Sprintf(`

> WARNING: %s is %s. Only part of it was loaded. Keep index entries to one line under ~200 chars; move detail into topic files.`,
		entrypointName, reason)
}

// BuildAgentMemoryPrompt mirrors TS buildMemoryPrompt from memdir.ts.
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

	lines := BuildAgentMemoryLines(displayName, memoryDir, extraGuidelines, false, true)

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

// BuildAgentMemoryLines mirrors memdir.ts buildMemoryLines.
func BuildAgentMemoryLines(
	displayName string,
	memoryDir string,
	extraGuidelines []string,
	skipIndex bool,
	includeSearchingPastContext bool,
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
			fmt.Sprintf("- `MEMORY.md` is always loaded into your conversation context — lines after %d will be truncated, so keep the index concise", maxEntrypointLines),
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
	lines = append(lines, BuildTypesSectionIndividual()...)
	lines = append(lines, BuildWhatNotToSaveSection()...)
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

	if includeSearchingPastContext {
		lines = append(lines, buildSearchingPastContextLines(memoryDir)...)
	}

	return lines
}

// BuildKairosDailyLogPrompt mirrors memdir.ts buildAssistantDailyLogPrompt.
func BuildKairosDailyLogPrompt(memDirDisplay string, skipIndex bool) string {
	logPath := memDirDisplay + "logs/YYYY/MM/YYYY-MM-DD.md"
	lines := []string{
		"# auto memory",
		"",
		"You have a persistent, file-based memory system found at: `" + memDirDisplay + "`",
		"",
		"This session is long-lived. As you work, record anything worth remembering by **appending** to today's daily log file:",
		"",
		"`" + logPath + "`",
		"",
		"Substitute today's date (from `currentDate` in your context) for `YYYY-MM-DD`. When the date rolls over mid-session, start appending to the new day's file.",
		"",
		"Write each entry as a short timestamped bullet. Create the file (and parent directories) on first write if it does not exist. Do not rewrite or reorganize the log — it is append-only. A separate nightly process distills these logs into `MEMORY.md` and topic files.",
		"",
		"## What to log",
		"- User corrections and preferences (\"use bun, not npm\"; \"stop summarizing diffs\")",
		"- Facts about the user, their role, or their goals",
		"- Project context that is not derivable from the code (deadlines, incidents, decisions and their rationale)",
		"- Pointers to external systems (dashboards, Linear projects, Slack channels)",
		"- Anything the user explicitly asks you to remember",
		"",
	}
	lines = append(lines, BuildWhatNotToSaveSection()...)
	lines = append(lines, "")
	if !skipIndex {
		lines = append(lines,
			"## "+entrypointName,
			"",
			"`"+entrypointName+"` is the distilled index (maintained nightly from your logs) and is loaded into your context automatically. Read it for orientation, but do not edit it directly — record new information in today's log instead.",
			"",
		)
	}
	return strings.Join(lines, "\n")
}

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

// MemorySearchingPastContextBlock mirrors buildSearchingPastContextSection from memdir.ts.
// Generates instructions for searching past memory and transcript files.
func MemorySearchingPastContextBlock(memoryDir, projectDir string, embeddedOrRepl bool) string {
	memSearch := fmt.Sprintf(`%s with pattern="<search term>" path="%s" glob="*.md"`, grepToolName, memoryDir)
	transcriptSearch := fmt.Sprintf(`%s with pattern="<search term>" path="%s/" glob="*.jsonl"`, grepToolName, strings.TrimSuffix(filepath.ToSlash(projectDir), "/"))
	if embeddedOrRepl {
		memSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.md"`, memoryDir)
		transcriptSearch = fmt.Sprintf(`grep -rn "<search term>" %s/ --include="*.jsonl"`, strings.TrimSuffix(filepath.ToSlash(projectDir), "/"))
	}
	return fmt.Sprintf("## Searching past context\n\nWhen looking for past context:\n1. Search topic files in your memory directory:\n```\n%s\n```\n2. Session transcript logs (last resort — large files, slow):\n```\n%s\n```\nUse narrow search terms (error messages, file paths, function names) rather than broad keywords.\n", memSearch, transcriptSearch)
}

// AppendMemorySearchPastContext appends search-past-context guidance to an auto-memory prompt.
func AppendMemorySearchPastContext(s string, memDir, cwd string, embeddedSearchOrRepl bool) string {
	pdir := claudeProjectSessionDir(cwd)
	if pdir == "" {
		return s
	}
	md := strings.TrimSuffix(filepath.ToSlash(memDir), "/")
	return s + "\n\n" + MemorySearchingPastContextBlock(md, pdir, embeddedSearchOrRepl)
}

func memoryDirDisplayPath(memDir string) string {
	p := strings.TrimSpace(memDir)
	if p == "" {
		return ""
	}
	p = filepath.Clean(strings.TrimSuffix(p, string(filepath.Separator)))
	p = filepath.ToSlash(p)
	return p + "/"
}

func claudeProjectSessionDir(originalCwd string) string {
	abs, err := filepath.Abs(strings.TrimSpace(originalCwd))
	if err != nil || abs == "" {
		abs = "."
	}
	key := claudebase.SanitizePath(abs)
	if cr := claudebase.ResolveCanonicalGitRoot(abs); cr != "" {
		key = claudebase.SanitizePath(cr)
	}
	base := MemoryBaseDir()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "projects", key)
}

// BuildAutoMemoryPrompt mirrors loadMemoryPrompt() (src/memdir/memdir.ts): KAIROS daily log, TEAMMEM combined,
// auto-only buildMemoryLines; ensureMemoryDirExists on team + auto-only branches; cowork extra only on team + auto.
func BuildAutoMemoryPrompt(o AutoMemoryPromptOpts) string {
	if !IsAutoMemoryEnabled() {
		return ""
	}
	cwd := strings.TrimSpace(o.Cwd)
	if cwd == "" {
		cwd = "."
	}
	memDir := GetAutoMemPath(cwd)
	skipIndex := o.MemorySkipIndex

	if o.KairosActive {
		s := BuildKairosDailyLogPrompt(memoryDirDisplayPath(memDir), skipIndex)
		if o.MemorySearchPastContext {
			s = AppendMemorySearchPastContext(s, memDir, cwd, o.EmbeddedSearchTools || o.ReplModeEnabled)
		}
		return strings.TrimSpace(s)
	}

	if o.TeamMemActive && IsTeamMemoryPromptActive() {
		teamDir := GetTeamMemPath(cwd)
		_ = EnsureMemoryDirExists(teamDir)
		var extra []string
		if x := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES")); x != "" {
			extra = []string{x}
		}
		s := BuildTeamCombinedMemoryPrompt(
			memoryDirDisplayPath(memDir),
			memoryDirDisplayPath(teamDir),
			skipIndex,
			extra,
		)
		if o.MemorySearchPastContext {
			s = AppendMemorySearchPastContext(s, memDir, cwd, o.EmbeddedSearchTools || o.ReplModeEnabled)
		}
		return strings.TrimSpace(s)
	}

	_ = EnsureMemoryDirExists(memDir)
	var extra []string
	if x := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES")); x != "" {
		extra = []string{x}
	}
	lines := BuildAgentMemoryLines("auto memory", memoryDirDisplayPath(memDir), extra, skipIndex, false)
	s := strings.Join(lines, "\n")
	if o.MemorySearchPastContext {
		s = AppendMemorySearchPastContext(s, memDir, cwd, o.EmbeddedSearchTools || o.ReplModeEnabled)
	}
	return strings.TrimSpace(s)
}
