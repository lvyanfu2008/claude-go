package extractmemories

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/claudemd"
	"goc/memdir"
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
	return strings.Join(memdir.BuildTypesSectionIndividual(), "\n")
}

func whatNotToSaveSection() string {
	return strings.Join(memdir.BuildWhatNotToSaveSection(), "\n")
}

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
