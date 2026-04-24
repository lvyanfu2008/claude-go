package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/commands/handwritten"
	"goc/types"
)

// HelpResult is the JSON payload returned by /help.
type HelpResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleHelpCommand returns a text listing of available commands for /help.
// Mirrors TS src/commands/help/ (local-jsx -> HelpV2 component).
func HandleHelpCommand() ([]byte, error) {
	// Gather all bundled + builtin commands for display
	allCmds := handwritten.AssembleBundledSkills()

	var lines []string
	lines = append(lines, "Claude Code (gou-demo) — Available Commands")
	lines = append(lines, strings.Repeat("═", 50))
	lines = append(lines, "")

	// Local commands
	lines = append(lines, "Local Commands (execute without LLM):")
	lines = append(lines, strings.Repeat("─", 40))

	localCmds := []struct {
		name, desc string
	}{
		{"clear", "Clear conversation history"},
		{"compact", "Compact conversation (not yet implemented in gou-demo)"},
		{"context", "Show context usage summary"},
		{"cost", "Show session token usage"},
		{"doctor", "Diagnose and verify your Claude Code installation"},
		{"effort", "Set effort level for model usage [low|medium|high|max|auto]"},
		{"files", "List files tracked in this session"},
		{"help", "Show this help message"},
		{"init", "Initialize a new CLAUDE.md file with codebase documentation"},
		{"keybindings", "Show keyboard shortcuts"},
		{"model", "Set the AI model for Claude Code"},
		{"plugins", "Manage Claude Code plugins (stub in gou-demo)"},
		{"release-notes", "Show release notes"},
		{"session", "Show remote session URL (remote mode only)"},
		{"status", "Show Claude Code status information"},
		{"stickers", "Order Claude Code stickers"},
		{"version", "Show gou-demo version"},
		{"vim", "Toggle vim mode [on|off]"},
	}

	for _, c := range localCmds {
		lines = append(lines, fmt.Sprintf("  /%-16s %s", c.name, c.desc))
	}
	lines = append(lines, "")

	// Bundled prompt skills
	lines = append(lines, "Bundled Skills (resolve to LLM prompts):")
	lines = append(lines, strings.Repeat("─", 40))
	for _, c := range allCmds {
		if c.Type == "prompt" {
			label := c.Name
			if c.ArgumentHint != nil && *c.ArgumentHint != "" {
				label = label + " " + *c.ArgumentHint
			}
			desc := c.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			lines = append(lines, fmt.Sprintf("  /%-24s %s", label, desc))
		}
	}
	lines = append(lines, "")
	lines = append(lines, "For more details, use the TS CLI: claude /help")

	msg := HelpResult{
		Type:  "text",
		Value: strings.Join(lines, "\n"),
	}
	return json.Marshal(msg)
}

// used only for the []types.Command import — keep the compiler happy
var _ = []types.Command{}
