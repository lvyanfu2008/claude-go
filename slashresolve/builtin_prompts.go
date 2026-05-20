package slashresolve

import (
	"fmt"

	"goc/types"
)

// IsBuiltinPrompt returns true for prompt commands whose Source is "builtin".
func IsBuiltinPrompt(cmd types.Command) bool {
	return cmd.Type == "prompt" && cmd.Source != nil && *cmd.Source == "builtin"
}

// ResolveBuiltinPrompt resolves a built-in prompt command to its prompt text.
func ResolveBuiltinPrompt(cmd types.Command, args string) (types.SlashResolveResult, error) {
	switch cmd.Name {
	default:
		return types.SlashResolveResult{}, fmt.Errorf("builtin prompt %q not yet implemented in Go", cmd.Name)
	}
}
