package commands

import (
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
	"goc/types"
)

// Mirrors src/tools/SkillTool/prompt.ts constants and formatting.

const (
	SkillBudgetContextPercent = 0.01
	CharsPerToken             = 4
	DefaultCharBudget         = 8000
	MaxListingDescChars       = 250
	minDescLength             = 20
)

// GetCharBudget mirrors getCharBudget(contextWindowTokens?) in prompt.ts.
func GetCharBudget(contextWindowTokens *int) int {
	if v := os.Getenv("SLASH_COMMAND_TOOL_CHAR_BUDGET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if contextWindowTokens != nil && *contextWindowTokens > 0 {
		return int(float64(*contextWindowTokens) * float64(CharsPerToken) * SkillBudgetContextPercent)
	}
	return DefaultCharBudget
}

func skillStringWidth(s string) int {
	w := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		w += runewidth.StringWidth(gr.Str())
	}
	return w
}

func getCommandDescription(cmd types.Command) string {
	var desc string
	if cmd.WhenToUse != nil && strings.TrimSpace(*cmd.WhenToUse) != "" {
		desc = cmd.Description + " - " + *cmd.WhenToUse
	} else {
		desc = cmd.Description
	}
	rc := utf8RuneCount(desc)
	if rc > MaxListingDescChars {
		return string([]rune(desc)[:MaxListingDescChars-1]) + "\u2026"
	}
	return desc
}

func utf8RuneCount(s string) int {
	return len([]rune(s))
}

func formatCommandDescription(cmd types.Command) string {
	displayName := types.GetCommandName(cmd)
	return "- " + displayName + ": " + getCommandDescription(cmd)
}

// FormatCommandsWithinBudget mirrors formatCommandsWithinBudget in prompt.ts.
// Bundled partition uses cmd.Source == "bundled" (same as TS), not LoadedFrom.
func FormatCommandsWithinBudget(commands []types.Command, contextWindowTokens *int) string {
	if len(commands) == 0 {
		return ""
	}
	budget := GetCharBudget(contextWindowTokens)

	fullEntries := make([]struct {
		cmd  types.Command
		full string
	}, len(commands))
	for i, cmd := range commands {
		fullEntries[i].cmd = cmd
		fullEntries[i].full = formatCommandDescription(cmd)
	}
	fullTotal := 0
	for i, e := range fullEntries {
		fullTotal += skillStringWidth(e.full)
		if i > 0 {
			fullTotal += 1 // newline between entries (join semantics)
		}
	}

	if fullTotal <= budget {
		lines := make([]string, len(fullEntries))
		for i, e := range fullEntries {
			lines[i] = e.full
		}
		return strings.Join(lines, "\n")
	}

	bundledIndices := make(map[int]struct{})
	var restCommands []types.Command
	for i, cmd := range commands {
		if cmd.Type == "prompt" && cmd.Source != nil && *cmd.Source == "bundled" {
			bundledIndices[i] = struct{}{}
		} else {
			restCommands = append(restCommands, cmd)
		}
	}

	bundledChars := 0
	for i, e := range fullEntries {
		if _, ok := bundledIndices[i]; ok {
			bundledChars += skillStringWidth(e.full) + 1
		}
	}
	remainingBudget := budget - bundledChars

	if len(restCommands) == 0 {
		lines := make([]string, len(fullEntries))
		for i, e := range fullEntries {
			lines[i] = e.full
		}
		return strings.Join(lines, "\n")
	}

	restNameOverhead := 0
	for i, cmd := range restCommands {
		restNameOverhead += skillStringWidth(cmd.Name) + 4 // "- " + ": " prefix/suffix overhead per TS
		if i > 0 {
			restNameOverhead += 1
		}
	}
	availableForDescs := remainingBudget - restNameOverhead
	maxDescLen := availableForDescs / len(restCommands)

	if maxDescLen < minDescLength {
		var b strings.Builder
		for i, cmd := range commands {
			if i > 0 {
				b.WriteByte('\n')
			}
			if _, ok := bundledIndices[i]; ok {
				b.WriteString(fullEntries[i].full)
			} else {
				b.WriteString("- ")
				b.WriteString(cmd.Name)
			}
		}
		return b.String()
	}

	var b strings.Builder
	for i, cmd := range commands {
		if i > 0 {
			b.WriteByte('\n')
		}
		if _, ok := bundledIndices[i]; ok {
			b.WriteString(fullEntries[i].full)
			continue
		}
		description := getCommandDescription(cmd)
		b.WriteString("- ")
		b.WriteString(cmd.Name)
		b.WriteString(": ")
		b.WriteString(truncateDisplayWidth(description, maxDescLen))
	}
	return b.String()
}

func truncateDisplayWidth(str string, maxWidth int) string {
	if skillStringWidth(str) <= maxWidth {
		return str
	}
	if maxWidth <= 1 {
		return "\u2026"
	}
	w := 0
	var out strings.Builder
	gr := uniseg.NewGraphemes(str)
	for gr.Next() {
		seg := gr.Str()
		sw := runewidth.StringWidth(seg)
		if w+sw > maxWidth-1 {
			break
		}
		out.WriteString(seg)
		w += sw
	}
	return out.String() + "\u2026"
}
