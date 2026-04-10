package claudemd

import (
	"os"
	"strings"
)

// MemoryInstructionPrompt matches claudemd.ts MEMORY_INSTRUCTION_PROMPT.
const MemoryInstructionPrompt = `Codebase and user instructions are shown below. Be sure to adhere to these instructions. IMPORTANT: These instructions OVERRIDE any default behavior and you MUST follow them exactly as written.`

// MaxMemoryCharacterCount matches claudemd.ts MAX_MEMORY_CHARACTER_COUNT.
const MaxMemoryCharacterCount = 40000

// FormatGetClaudeMds mirrors getClaudeMds (no filter callback).
func FormatGetClaudeMds(memoryFiles []MemoryFileInfo) string {
	skipProjectLevel := truthy(os.Getenv("CLAUDE_CODE_TENGU_PAPER_HALYARD"))
	var memories []string
	for _, file := range memoryFiles {
		if skipProjectLevel && (file.Type == MemoryProject || file.Type == MemoryLocal) {
			continue
		}
		if strings.TrimSpace(file.Content) == "" {
			continue
		}
		desc := descriptionForType(file.Type)
		content := strings.TrimSpace(file.Content)
		if featureTeamMem() && file.Type == MemoryTeamMem {
			memories = append(memories, "Contents of "+file.Path+desc+":\n\n<team-memory-content source=\"shared\">\n"+content+"\n</team-memory-content>")
		} else {
			memories = append(memories, "Contents of "+file.Path+desc+":\n\n"+content)
		}
	}
	if len(memories) == 0 {
		return ""
	}
	s := MemoryInstructionPrompt + "\n\n" + strings.Join(memories, "\n\n")
	if len(s) > MaxMemoryCharacterCount {
		s = s[:MaxMemoryCharacterCount] + "\n... (truncated)"
	}
	return s
}

func descriptionForType(t MemoryType) string {
	switch t {
	case MemoryProject:
		return " (project instructions, checked into the codebase)"
	case MemoryLocal:
		return " (user's private project instructions, not checked in)"
	case MemoryAutoMem:
		return " (user's auto-memory, persists across conversations)"
	case MemoryTeamMem:
		return " (shared team memory, synced across the organization)"
	default:
		// Managed, User, TeamMem (non-XML branch uses same as TS fallback)
		return " (user's private global instructions for all projects)"
	}
}

func featureTeamMem() bool {
	return truthy(os.Getenv("FEATURE_TEAMMEM"))
}

// FilterInjectedMemoryFiles mirrors filterInjectedMemoryFiles when GrowthBook flag is enabled via env.
func FilterInjectedMemoryFiles(files []MemoryFileInfo) []MemoryFileInfo {
	if !truthy(os.Getenv("CLAUDE_CODE_TENGU_MOTH_COPSE")) {
		return files
	}
	var out []MemoryFileInfo
	for _, f := range files {
		if f.Type == MemoryAutoMem || f.Type == MemoryTeamMem {
			continue
		}
		out = append(out, f)
	}
	return out
}
