package sessionmemory

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"goc/claudebase"
	"goc/compactservice"
)

const (
	// MaxSectionLength mirrors TS MAX_SECTION_LENGTH — per-section rough token budget.
	MaxSectionLength = 2000
	// MaxTotalSessionMemoryTokens mirrors TS MAX_TOTAL_SESSION_MEMORY_TOKENS.
	MaxTotalSessionMemoryTokens = 12000
)

// defaultSessionMemoryTemplate mirrors TS DEFAULT_SESSION_MEMORY_TEMPLATE.
const defaultSessionMemoryTemplate = `
# Session Title
_A short and distinctive 5-10 word descriptive title for the session. Super info dense, no filler_

# Current State
_What is actively being worked on right now? Pending tasks not yet completed. Immediate next steps._

# Task specification
_What did the user ask to build? Any design decisions or other explanatory context_

# Files and Functions
_What are the important files? In short, what do they contain and why are they relevant?_

# Workflow
_What bash commands are usually run and in what order? How to interpret their output if not obvious?_

# Errors & Corrections
_Errors encountered and how they were fixed. What did the user correct? What approaches failed and should not be tried again?_

# Codebase and System Documentation
_What are the important system components? How do they work/fit together?_

# Learnings
_What has worked well? What has not? What to avoid? Do not duplicate items from other sections_

# Key results
_If the user asked a specific output such as an answer to a question, a table, or other document, repeat the exact result here_

# Worklog
_Step by step, what was attempted, done? Very terse summary for each step_
`

// getDefaultUpdatePrompt mirrors TS getDefaultUpdatePrompt.
func getDefaultUpdatePrompt() string {
	return `IMPORTANT: This message and these instructions are NOT part of the actual user conversation. Do NOT include any references to "note-taking", "session notes extraction", or these update instructions in the notes content.

Based on the user conversation above (EXCLUDING this note-taking instruction message as well as system prompt, claude.md entries, or any past session summaries), update the session notes file.

The file {{notesPath}} has already been read for you. Here are its current contents:
<current_notes_content>
{{currentNotes}}
</current_notes_content>

Your ONLY task is to use the Edit tool to update the notes file, then stop. You can make multiple edits (update every section as needed) - make all Edit tool calls in parallel in a single message. Do not call any other tools.

CRITICAL RULES FOR EDITING:
- The file must maintain its exact structure with all sections, headers, and italic descriptions intact
-- NEVER modify, delete, or add section headers (the lines starting with '#' like # Task specification)
-- NEVER modify or delete the italic _section description_ lines (these are the lines in italics immediately following each header - they start and end with underscores)
-- The italic _section descriptions_ are TEMPLATE INSTRUCTIONS that must be preserved exactly as-is - they guide what content belongs in each section
-- ONLY update the actual content that appears BELOW the italic _section descriptions_ within each existing section
-- Do NOT add any new sections, summaries, or information outside the existing structure
- Do NOT reference this note-taking process or instructions anywhere in the notes
- It's OK to skip updating a section if there are no substantial new insights to add. Do not add filler content like "No info yet", just leave sections blank/unedited if appropriate.
- Write DETAILED, INFO-DENSE content for each section - include specifics like file paths, function names, error messages, exact commands, technical details, etc.
- For "Key results", include the complete, exact output the user requested (e.g., full table, full answer, etc.)
- Do not include information that's already in the CLAUDE.md files included in the context
- Keep each section under ~` + formatInt(MaxSectionLength) + ` tokens/words - if a section is approaching this limit, condense it by cycling out less important details while preserving the most critical information
- Focus on actionable, specific information that would help someone understand or recreate the work discussed in the conversation
- IMPORTANT: Always update "Current State" to reflect the most recent work - this is critical for continuity after compaction

Use the Edit tool with file_path: {{notesPath}}

STRUCTURE PRESERVATION REMINDER:
Each section has TWO parts that must be preserved exactly as they appear in the current file:
1. The section header (line starting with #)
2. The italic description line (the _italicized text_ immediately after the header - this is a template instruction)

You ONLY update the actual content that comes AFTER these two preserved lines. The italic description lines starting and ending with underscores are part of the template structure, NOT content to be edited or removed.

REMEMBER: Use the Edit tool in parallel and stop. Do not continue after the edits. Only include insights from the actual user conversation, never from these note-taking instructions. Do not delete or change section headers or italic _section descriptions_.`
}

func formatInt(n int) string {
	return strconv.Itoa(n)
}

// LoadSessionMemoryTemplate loads the custom session memory template from
// ~/.claude/session-memory/config/template.md, falling back to the default.
// Mirrors TS loadSessionMemoryTemplate.
func LoadSessionMemoryTemplate() string {
	configHome, err := claudebase.ClaudeConfigHomeDir()
	if err != nil {
		return defaultSessionMemoryTemplate
	}
	templatePath := filepath.Join(configHome,
		"session-memory", "config", "template.md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultSessionMemoryTemplate
		}
		return defaultSessionMemoryTemplate
	}
	return string(data)
}

// LoadSessionMemoryPrompt loads the custom session memory prompt from
// ~/.claude/session-memory/config/prompt.md, falling back to the default.
// Mirrors TS loadSessionMemoryPrompt.
func LoadSessionMemoryPrompt() string {
	configHome, err := claudebase.ClaudeConfigHomeDir()
	if err != nil {
		return getDefaultUpdatePrompt()
	}
	promptPath := filepath.Join(configHome,
		"session-memory", "config", "prompt.md")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return getDefaultUpdatePrompt()
		}
		return getDefaultUpdatePrompt()
	}
	return string(data)
}

// analyzeSectionSizes parses the session memory file and returns a map of
// section header -> rough token count for content beneath each header.
// Mirrors TS analyzeSectionSizes.
func analyzeSectionSizes(content string) map[string]int {
	sections := make(map[string]int)
	lines := strings.Split(content, "\n")
	var currentSection string
	var currentContent []string

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			if currentSection != "" && len(currentContent) > 0 {
				sectionContent := strings.TrimSpace(strings.Join(currentContent, "\n"))
				sections[currentSection] = compactservice.RoughTokenCountEstimation(sectionContent)
			}
			currentSection = line
			currentContent = nil
		} else {
			currentContent = append(currentContent, line)
		}
	}

	if currentSection != "" && len(currentContent) > 0 {
		sectionContent := strings.TrimSpace(strings.Join(currentContent, "\n"))
		sections[currentSection] = compactservice.RoughTokenCountEstimation(sectionContent)
	}

	return sections
}

// generateSectionReminders produces warnings about oversized sections and
// total token budget. Mirrors TS generateSectionReminders.
func generateSectionReminders(sectionSizes map[string]int, totalTokens int) string {
	overBudget := totalTokens > MaxTotalSessionMemoryTokens

	type oversized struct {
		section string
		tokens  int
	}
	var oversizedSections []oversized
	for section, tokens := range sectionSizes {
		if tokens > MaxSectionLength {
			oversizedSections = append(oversizedSections, oversized{section, tokens})
		}
	}
	// Sort descending by token count.
	for i := 0; i < len(oversizedSections); i++ {
		for j := i + 1; j < len(oversizedSections); j++ {
			if oversizedSections[j].tokens > oversizedSections[i].tokens {
				oversizedSections[i], oversizedSections[j] = oversizedSections[j], oversizedSections[i]
			}
		}
	}

	if len(oversizedSections) == 0 && !overBudget {
		return ""
	}

	var parts []string

	if overBudget {
		parts = append(parts,
			"\n\nCRITICAL: The session memory file is currently ~"+formatInt(totalTokens)+
				" tokens, which exceeds the maximum of "+formatInt(MaxTotalSessionMemoryTokens)+
				" tokens. You MUST condense the file to fit within this budget. Aggressively shorten oversized sections by removing less important details, merging related items, and summarizing older entries. Prioritize keeping \"Current State\" and \"Errors & Corrections\" accurate and detailed.")
	}

	if len(oversizedSections) > 0 {
		header := "\n\nIMPORTANT: The following sections exceed the per-section limit and MUST be condensed"
		if overBudget {
			header = "\n\nOversized sections to condense"
		}
		var lines []string
		for _, os := range oversizedSections {
			lines = append(lines, `- "`+os.section+`" is ~`+formatInt(os.tokens)+
				` tokens (limit: `+formatInt(MaxSectionLength)+`)`)
		}
		parts = append(parts, header+":\n"+strings.Join(lines, "\n"))
	}

	return strings.Join(parts, "")
}

// substituteVariables replaces {{variable}} placeholders in the template.
// Mirrors TS substituteVariables.
func substituteVariables(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// IsSessionMemoryEmpty returns true if the content is essentially the default template.
// Mirrors TS isSessionMemoryEmpty.
func IsSessionMemoryEmpty(content string) bool {
	return strings.TrimSpace(content) == strings.TrimSpace(defaultSessionMemoryTemplate)
}

// BuildSessionMemoryUpdatePrompt builds the extraction prompt with section
// reminders. Mirrors TS buildSessionMemoryUpdatePrompt.
func BuildSessionMemoryUpdatePrompt(currentNotes, notesPath string) string {
	promptTemplate := LoadSessionMemoryPrompt()

	sectionSizes := analyzeSectionSizes(currentNotes)
	totalTokens := compactservice.RoughTokenCountEstimation(currentNotes)
	sectionReminders := generateSectionReminders(sectionSizes, totalTokens)

	vars := map[string]string{
		"currentNotes": currentNotes,
		"notesPath":    notesPath,
	}

	return substituteVariables(promptTemplate, vars) + sectionReminders
}

// TruncateSessionMemoryForCompact truncates session memory sections that exceed
// the per-section token limit. Returns truncated content and whether any truncation occurred.
// Mirrors TS truncateSessionMemoryForCompact.
func TruncateSessionMemoryForCompact(content string) (truncatedContent string, wasTruncated bool) {
	lines := strings.Split(content, "\n")
	maxCharsPerSection := MaxSectionLength * 4 // roughTokenCountEstimation uses length/4

	var outputLines []string
	var currentSectionLines []string
	var currentSectionHeader string

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			res := flushSessionSection(currentSectionHeader, currentSectionLines, maxCharsPerSection)
			outputLines = append(outputLines, res.lines...)
			wasTruncated = wasTruncated || res.wasTruncated
			currentSectionHeader = line
			currentSectionLines = nil
		} else {
			currentSectionLines = append(currentSectionLines, line)
		}
	}

	// Flush the last section.
	res := flushSessionSection(currentSectionHeader, currentSectionLines, maxCharsPerSection)
	outputLines = append(outputLines, res.lines...)
	wasTruncated = wasTruncated || res.wasTruncated

	return strings.Join(outputLines, "\n"), wasTruncated
}

type sectionFlushResult struct {
	lines        []string
	wasTruncated bool
}

// flushSessionSection mirrors TS flushSessionSection.
func flushSessionSection(sectionHeader string, sectionLines []string, maxCharsPerSection int) sectionFlushResult {
	if sectionHeader == "" {
		return sectionFlushResult{lines: sectionLines, wasTruncated: false}
	}

	sectionContent := strings.Join(sectionLines, "\n")
	if len(sectionContent) <= maxCharsPerSection {
		return sectionFlushResult{
			lines:        append([]string{sectionHeader}, sectionLines...),
			wasTruncated: false,
		}
	}

	// Truncate at a line boundary near the limit.
	charCount := 0
	keptLines := []string{sectionHeader}
	for _, line := range sectionLines {
		if charCount+len(line)+1 > maxCharsPerSection {
			break
		}
		keptLines = append(keptLines, line)
		charCount += len(line) + 1
	}
	keptLines = append(keptLines, "\n[... section truncated for length ...]")
	return sectionFlushResult{lines: keptLines, wasTruncated: true}
}
