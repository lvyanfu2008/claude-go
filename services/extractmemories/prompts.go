package extractmemories

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/claudemd"
	"goc/types"
)

// ExtractMemoriesRelaxThreshold is true when GOC_EXTRACT_MEMORIES_RELAX_THRESHOLD or
// CLAUDE_CODE_EXTRACT_MEMORIES_RELAX_THRESHOLD is 1|true|yes|on — use a less strict
// closing line so the sub-agent is more likely to Write when the user states preferences.
func ExtractMemoriesRelaxThreshold() bool {
	for _, k := range []string{
		"GOC_EXTRACT_MEMORIES_RELAX_THRESHOLD",
		"CLAUDE_CODE_EXTRACT_MEMORIES_RELAX_THRESHOLD",
	} {
		v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
		if v == "1" || v == "true" || v == "yes" || v == "on" {
			return true
		}
	}
	return false
}

// buildExtractionPrompt constructs the prompt for the extraction sub-agent.
// Mirrors src/services/extractMemories/prompts.ts buildExtractAutoOnlyPrompt.
func buildExtractionPrompt(p ExtractionParams, newMessages []types.Message, memoryDir string) string {
	memDisplay := memoryDirDisplayPath(memoryDir)
	existingMemories := scanExistingMemories(memoryDir)
	memoryIndex := readMemoryIndex(memoryDir)
	newCount := countModelVisibleMessages(newMessages)

	var b strings.Builder

	// Opener — mirrors TS opener() + tool list + turn-budget strategy
	b.WriteString(opener(newCount, existingMemories, memoryDir, memDisplay))

	// Existing MEMORY.md index (Go addition — not in TS extraction prompt)
	if memoryIndex != "" && !p.SkipIndex {
		b.WriteString("\n## Current MEMORY.md index\n\n")
		b.WriteString(memoryIndex)
		b.WriteString("\n\n")
	}

	// "If user explicitly asks to remember/forget" — mirrors TS
	b.WriteString("If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.\n\n")

	// Type taxonomy — mirrors TS TYPES_SECTION_INDIVIDUAL
	b.WriteString(typesSectionIndividual())

	// What NOT to save — mirrors TS WHAT_NOT_TO_SAVE_SECTION
	b.WriteString(whatNotToSaveSection())

	// How to save — mirrors TS howToSave with skipIndex variant
	b.WriteString(howToSaveSection(p.SkipIndex))

	// Threshold/footer
	if ExtractMemoriesRelaxThreshold() {
		fmt.Fprintf(&b, "\nPreference: if the user stated any preference, correction, or stable fact in these %d messages, save it in a new or updated .md under the memory directory (and the MEMORY.md index as needed). Only skip when there is no user-specific, repeatable information to persist.\n", newCount)
	} else {
		fmt.Fprintf(&b, "\nThreshold: if nothing new or notable was learned from these %d messages, do nothing. It's better to skip than to create trivial memories.\n", newCount)
	}

	return strings.TrimSpace(b.String())
}

// opener mirrors TS opener() in src/services/extractMemories/prompts.ts.
// It frames the task, lists available tools, and gives turn-budget strategy.
func opener(newMessageCount int, existingMemories, memoryDir, memDisplay string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("You are now acting as the memory extraction subagent. Analyze the most recent ~%d messages above and use them to update your persistent memory systems.\n\n", newMessageCount))

	b.WriteString("Available tools: Read, Grep, Glob, read-only Bash (ls/find/cat/stat/wc/head/tail and similar), and Edit/Write for paths inside the memory directory only. Bash rm is not permitted. All other tools — MCP, Agent, write-capable Bash, etc — will be denied.\n\n")

	b.WriteString("You have a limited turn budget. Edit requires a prior Read of the same file, so the efficient strategy is: turn 1 — issue all Read calls in parallel for every file you might update; turn 2 — issue all Write/Edit calls in parallel. Do not interleave reads and writes across multiple turns.\n")

	if memDisplay != "" {
		b.WriteString(fmt.Sprintf("\nMemory directory: `%s`\n", memDisplay))
	}
	b.WriteString("\n" + dirExistsGuidance + "\n")

	if existingMemories != "" {
		b.WriteString("\n## Existing memory files\n\n")
		b.WriteString(existingMemories)
		b.WriteString("\n\nCheck this list before writing — update an existing file rather than creating a duplicate.\n")
	}

	b.WriteString(fmt.Sprintf("\nYou MUST only use content from the last ~%d messages to update your persistent memories. Do not waste any turns attempting to investigate or verify that content further — no grepping source files, no reading code to confirm a pattern exists, no git commands.\n\n", newMessageCount))

	return b.String()
}

// typesSectionIndividual mirrors TS TYPES_SECTION_INDIVIDUAL.
// Full XML-like type taxonomy with when_to_save, how_to_use, body_structure, and examples.
func typesSectionIndividual() string {
	return `## Types of memory

There are several discrete types of memory that you can store in your memory system:

<types>
<type>
    <name>user</name>
    <description>Contain information about the user's role, goals, responsibilities, and knowledge. Great user memories help you tailor your future behavior to the user's preferences and perspective. Your goal in reading and writing these memories is to build up an understanding of who the user is and how you can be most helpful to them specifically. For example, you should collaborate with a senior software engineer differently than a student who is coding for the very first time. Keep in mind, that the aim here is to be helpful to the user. Avoid writing memories about the user that could be viewed as a negative judgement or that are not relevant to the work you're trying to accomplish together.</description>
    <when_to_save>When you learn any details about the user's role, preferences, responsibilities, or knowledge</when_to_save>
    <how_to_use>When your work should be informed by the user's profile or perspective. For example, if the user is asking you to explain a part of the code, you should answer that question in a way that is tailored to the specific details that they will find most valuable or that helps them build their mental model in relation to domain knowledge they already have.</how_to_use>
    <examples>
    user: I'm a data scientist investigating what logging we have in place
    assistant: [saves user memory: user is a data scientist, currently focused on observability/logging]

    user: I've been writing Go for ten years but this is my first time touching the React side of this repo
    assistant: [saves user memory: deep Go expertise, new to React and this project's frontend — frame frontend explanations in terms of backend analogues]
    </examples>
</type>
<type>
    <name>feedback</name>
    <description>Guidance the user has given you about how to approach work — both what to avoid and what to keep doing. These are a very important type of memory to read and write as they allow you to remain coherent and responsive to the way you should approach work in the project. Record from failure AND success: if you only save corrections, you will avoid past mistakes but drift away from approaches the user has already validated, and may grow overly cautious.</description>
    <when_to_save>Any time the user corrects your approach ("no not that", "don't", "stop doing X") OR confirms a non-obvious approach worked ("yes exactly", "perfect, keep doing that", accepting an unusual choice without pushback). Corrections are easy to notice; confirmations are quieter — watch for them. In both cases, save what is applicable to future conversations, especially if surprising or not obvious from the code. Include *why* so you can judge edge cases later.</when_to_save>
    <how_to_use>Let these memories guide your behavior so that the user does not need to offer the same guidance twice.</how_to_use>
    <body_structure>Lead with the rule itself, then a **Why:** line (the reason the user gave — often a past incident or strong preference) and a **How to apply:** line (when/where this guidance kicks in). Knowing *why* lets you judge edge cases instead of blindly following the rule.</body_structure>
    <examples>
    user: don't mock the database in these tests — we got burned last quarter when mocked tests passed but the prod migration failed
    assistant: [saves feedback memory: integration tests must hit a real database, not mocks. Reason: prior incident where mock/prod divergence masked a broken migration]

    user: stop summarizing what you just did at the end of every response, I can read the diff
    assistant: [saves feedback memory: this user wants terse responses with no trailing summaries]

    user: yeah the single bundled PR was the right call here, splitting this one would've just been churn
    assistant: [saves feedback memory: for refactors in this area, user prefers one bundled PR over many small ones. Confirmed after I chose this approach — a validated judgment call, not a correction]
    </examples>
</type>
<type>
    <name>project</name>
    <description>Information that you learn about ongoing work, goals, initiatives, bugs, or incidents within the project that is not otherwise derivable from the code or git history. Project memories help you understand the broader context and motivation behind the work the user is doing within this working directory.</description>
    <when_to_save>When you learn who is doing what, why, or by when. These states change relatively quickly so try to keep your understanding of this up to date. Always convert relative dates in user messages to absolute dates when saving (e.g., "Thursday" → "2026-03-05"), so the memory remains interpretable after time passes.</when_to_save>
    <how_to_use>Use these memories to more fully understand the details and nuance behind the user's request and make better informed suggestions.</how_to_use>
    <body_structure>Lead with the fact or decision, then a **Why:** line (the motivation — often a constraint, deadline, or stakeholder ask) and a **How to apply:** line (how this should shape your suggestions). Project memories decay fast, so the why helps future-you judge whether the memory is still load-bearing.</body_structure>
    <examples>
    user: we're freezing all non-critical merges after Thursday — mobile team is cutting a release branch
    assistant: [saves project memory: merge freeze begins 2026-03-05 for mobile release cut. Flag any non-critical PR work scheduled after that date]

    user: the reason we're ripping out the old auth middleware is that legal flagged it for storing session tokens in a way that doesn't meet the new compliance requirements
    assistant: [saves project memory: auth middleware rewrite is driven by legal/compliance requirements around session token storage, not tech-debt cleanup — scope decisions should favor compliance over ergonomics]
    </examples>
</type>
<type>
    <name>reference</name>
    <description>Stores pointers to where information can be found in external systems. These memories allow you to remember where to look to find up-to-date information outside of the project directory.</description>
    <when_to_save>When you learn about resources in external systems and their purpose. For example, that bugs are tracked in a specific project in Linear or that feedback can be found in a specific Slack channel.</when_to_save>
    <how_to_use>When the user references an external system or information that may be in an external system.</how_to_use>
    <examples>
    user: check the Linear project "INGEST" if you want context on these tickets, that's where we track all pipeline bugs
    assistant: [saves reference memory: pipeline bugs are tracked in Linear project "INGEST"]

    user: the Grafana board at grafana.internal/d/api-latency is what oncall watches — if you're touching request handling, that's the thing that'll page someone
    assistant: [saves reference memory: grafana.internal/d/api-latency is the oncall latency dashboard — check it when editing request-path code]
    </examples>
</type>
</types>

`
}

// whatNotToSaveSection mirrors TS WHAT_NOT_TO_SAVE_SECTION.
// Includes the explicit-save gate: exclusions apply even when user explicitly asks to save.
func whatNotToSaveSection() string {
	return `## What NOT to save in memory

- Code patterns, conventions, architecture, file paths, or project structure — these can be derived by reading the current project state.
- Git history, recent changes, or who-changed-what — ` + "`" + `git log` + "`" + ` / ` + "`" + `git blame` + "`" + ` are authoritative.
- Debugging solutions or fix recipes — the fix is in the code; the commit message has the context.
- Anything already documented in CLAUDE.md files.
- Ephemeral task details: in-progress work, temporary state, current conversation context.

These exclusions apply even when the user explicitly asks you to save. If they ask you to save a PR list or activity summary, ask what was *surprising* or *non-obvious* about it — that is the part worth keeping.

`
}

// howToSaveSection mirrors TS howToSave in buildExtractAutoOnlyPrompt.
// Supports skipIndex variant: when true, only Step 1 (write the file) is described,
// without the MEMORY.md index update step.
func howToSaveSection(skipIndex bool) string {
	var b strings.Builder

	b.WriteString("## How to save memories\n\n")

	if skipIndex {
		b.WriteString("Write each memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:\n\n")
		b.WriteString(memoryFrontmatterExample())
		b.WriteString("\n- Organize memory semantically by topic, not chronologically\n")
		b.WriteString("- Update or remove memories that turn out to be wrong or outdated\n")
		b.WriteString("- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.\n\n")
	} else {
		b.WriteString("Saving a memory is a two-step process:\n\n")
		b.WriteString("**Step 1** — write the memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:\n\n")
		b.WriteString(memoryFrontmatterExample())
		b.WriteString("\n**Step 2** — add a pointer to that file in `MEMORY.md`. `MEMORY.md` is an index, not a memory — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. It has no frontmatter. Never write memory content directly into `MEMORY.md`.\n\n")
		b.WriteString("- `MEMORY.md` is always loaded into your system prompt — lines after 200 will be truncated, so keep the index concise\n")
		b.WriteString("- Organize memory semantically by topic, not chronologically\n")
		b.WriteString("- Update or remove memories that turn out to be wrong or outdated\n")
		b.WriteString("- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.\n\n")
	}

	return b.String()
}

// memoryFrontmatterExample mirrors TS MEMORY_FRONTMATTER_EXAMPLE.
func memoryFrontmatterExample() string {
	return "```markdown\n" +
		"---\n" +
		"name: {{memory name}}\n" +
		"description: {{one-line description — used to decide relevance in future conversations, so be specific}}\n" +
		"type: {{user, feedback, project, reference}}\n" +
		"---\n\n" +
		"{{memory content — for feedback/project types, structure as: rule/fact, then **Why:** and **How to apply:** lines}}\n" +
		"```\n"
}

// memoryFileInfo holds parsed metadata from a memory file's frontmatter.
type memoryFileInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

// scanExistingMemories lists existing .md files (other than MEMORY.md) in memoryDir,
// with frontmatter metadata (name, description, type) parsed from each file.
func scanExistingMemories(memoryDir string) string {
	if memoryDir == "" {
		return ""
	}
	entries, err := os.ReadDir(memoryDir)
	if err != nil {
		return ""
	}
	var lines []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.EqualFold(e.Name(), entrypointName) {
			continue
		}
		line := e.Name()
		// Parse frontmatter for richer context.
		if info := readMemoryFileFrontmatter(filepath.Join(memoryDir, e.Name())); info != nil {
			parts := []string{e.Name()}
			if info.Name != "" {
				parts = append(parts, "name="+info.Name)
			}
			if info.Description != "" {
				d := info.Description
				if len(d) > 80 {
					d = d[:80] + "..."
				}
				parts = append(parts, d)
			}
			line = strings.Join(parts, " — ")
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// readMemoryFileFrontmatter reads a .md file and parses its YAML frontmatter
// into a memoryFileInfo. Returns nil if the file has no valid frontmatter.
func readMemoryFileFrontmatter(path string) *memoryFileInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	fm, _ := claudemd.ParseFrontmatter(string(data))
	if len(fm) == 0 {
		return nil
	}
	var info memoryFileInfo
	if v, ok := fm["name"]; ok {
		info.Name, _ = v.(string)
	}
	if v, ok := fm["description"]; ok {
		info.Description, _ = v.(string)
	}
	if v, ok := fm["type"]; ok {
		info.Type, _ = v.(string)
	}
	if info.Name == "" && info.Description == "" {
		return nil
	}
	return &info
}

// readMemoryIndex reads MEMORY.md from memoryDir (first 200 lines, matching TS truncation).
func readMemoryIndex(memoryDir string) string {
	if memoryDir == "" {
		return ""
	}
	idxPath := filepath.Join(memoryDir, entrypointName)
	data, err := os.ReadFile(idxPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	const maxLines = 200
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// memoryDirDisplayPath formats a memory directory path for display.
func memoryDirDisplayPath(memDir string) string {
	p := strings.TrimSpace(memDir)
	if p == "" {
		return ""
	}
	p = filepath.Clean(strings.TrimSuffix(p, string(filepath.Separator)))
	p = filepath.ToSlash(p)
	return p + "/"
}

// dirExistsGuidance matches TS DIR_EXISTS_GUIDANCE.
const dirExistsGuidance = "This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence)."
