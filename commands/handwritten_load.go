package commands

import (
	"strings"

	"goc/commands/handwritten"
	"goc/types"
)

func loadBuiltinCommands() []types.Command {
	out := handwritten.AssembleBuiltinCommands()
	seen := make(map[string]struct{}, len(out))
	for _, c := range out {
		if n := strings.TrimSpace(c.Name); n != "" {
			seen[n] = struct{}{}
		}
	}
	out = append(out, loadBuiltinCommandsDiskOverlay(seen)...)
	return out
}

func loadBundledSkills() []types.Command {
	return handwritten.AssembleBundledSkills()
}

func loadBuiltinPluginSkills() []types.Command {
	return handwritten.AssembleBuiltinPluginSkills()
}
