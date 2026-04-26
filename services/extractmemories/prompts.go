package extractmemories

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/claudemd"
	"goc/types"
)

// buildExtractionPrompt constructs the prompt for the extraction sub-agent.
// It frames the task as: review the recent conversation messages and extract
// new information into durable memory files.
//
// Mirrors src/services/extractMemories/prompts.ts buildExtractAutoOnlyPrompt.
func buildExtractionPrompt(p ExtractionParams, newMessages []types.Message, memoryDir string) string {
	memDisplay := memoryDirDisplayPath(memoryDir)

	existingMemories := scanExistingMemories(memoryDir)
	memoryIndex := readMemoryIndex(memoryDir)
	newCount := countModelVisibleMessages(newMessages)

	var b strings.Builder

	b.WriteString("# Extract Memories (auto)\n\n")
	b.WriteString(fmt.Sprintf("Review the most recent %d message(s) from the conversation below and extract new, durable information worth remembering.\n\n", newCount))
	b.WriteString(fmt.Sprintf("Memory directory: `%s`\n", memDisplay))
	b.WriteString(fmt.Sprintf("%s\n\n", dirExistsGuidance))

	if existingMemories != "" {
		b.WriteString("## Existing memory files\n\n")
		b.WriteString(existingMemories)
		b.WriteString("\n\n")
	}

	if memoryIndex != "" && !p.SkipIndex {
		b.WriteString("## Current MEMORY.md index\n\n")
		b.WriteString(memoryIndex)
		b.WriteString("\n\n")
	}

	b.WriteString(`## Instructions

1. **Read the recent messages** (they follow this prompt in the conversation).
2. **Identify new information** worth persisting — facts about the user, feedback about your approach, project context, or external references.
3. **Check existing memories** — if the information already exists, skip it. If an existing memory is contradicted, update it.
4. **Save new memories** using the Write tool (create individual .md files) and update MEMORY.md with index pointers.
5. **Keep it concise** — each memory should be a focused file. Merge related facts rather than creating many small files.

`)

	b.WriteString(memoryFormatSection())
	b.WriteString(whatNotToSaveSection())
	b.WriteString(howToSaveSection())
	b.WriteString(whenToAccessSection())

	b.WriteString(fmt.Sprintf("\nThreshold: if nothing new or notable was learned from these %d messages, do nothing. It's better to skip than to create trivial memories.\n", newCount))

	return strings.TrimSpace(b.String())
}

// memoryFormatSection returns the memory type system description.
func memoryFormatSection() string {
	return `## Types of memory

There are several discrete types of memory that you can store:

- **user** — Information about the user's role, goals, responsibilities, and knowledge.
- **feedback** — Guidance about how to approach work, both what to avoid and what to keep doing.
- **project** — Information about ongoing work, goals, initiatives, bugs, or non-obvious context.
- **reference** — Pointers to where information can be found in external systems.

`
}

// whatNotToSaveSection returns the "what not to save" instructions.
func whatNotToSaveSection() string {
	return `## What NOT to save

- Code patterns, conventions, architecture, file paths, or project structure.
- Git history, recent changes, or who-changed-what.
- Debugging solutions or fix recipes.
- Anything already documented in CLAUDE.md files.
- Ephemeral task details: in-progress work, temporary state, current conversation context.

`
}

// howToSaveSection returns the memory file format instructions.
func howToSaveSection() string {
	return `## How to save memories

Saving a memory is a two-step process:

**Step 1** — write the memory to its own file (e.g., user_role.md, feedback_testing.md) using this frontmatter format:

` + "```markdown" + `
---
name: {{memory name}}
description: {{one-line description — used to decide relevance in future conversations, so be specific}}
type: {{user, feedback, project, reference}}
---

{{memory content — for feedback/project types, structure as: rule/fact, then **Why:** and **How to apply:** lines}}
` + "```" + `

**Step 2** — add a pointer to that file in MEMORY.md. MEMORY.md is an index, not a memory — each entry should be one line, under ~150 characters: ` + "`- [Title](file.md) — one-line hook`" + `. It has no frontmatter. Never write memory content directly into MEMORY.md.

Key rules:
- Keep the name, description, and type fields in memory files up-to-date with the content.
- Organize memory semantically by topic, not chronologically.
- Update or remove memories that turn out to be wrong or outdated.
- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.

`
}

// whenToAccessSection returns instructions on when to use memories.
func whenToAccessSection() string {
	return `## When to access memories
- When memories seem relevant, or the user references prior-conversation work.
- You MUST access memory when the user explicitly asks you to check, recall, or remember.
- If the user says to ignore or not use memory: proceed as if MEMORY.md were empty. Do not apply remembered facts, cite, compare against, or mention memory content.
- Memory records can become stale over time. Use memory as context for what was true at a given point in time. Before answering or building assumptions based solely on information in memory records, verify that the memory is still correct and up-to-date by reading the current state of the files or resources. If a recalled memory conflicts with current information, trust what you observe now — and update or remove the stale memory rather than acting on it.

`
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

