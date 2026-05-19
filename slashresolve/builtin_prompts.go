package slashresolve

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"goc/types"
)

// IsBuiltinPrompt returns true for prompt commands whose Source is "builtin".
func IsBuiltinPrompt(cmd types.Command) bool {
	return cmd.Type == "prompt" && cmd.Source != nil && *cmd.Source == "builtin"
}

// ResolveBuiltinPrompt resolves a built-in prompt command to its prompt text.
func ResolveBuiltinPrompt(cmd types.Command, args string) (types.SlashResolveResult, error) {
	switch cmd.Name {
	case "init":
		return resolveInit(args)
	default:
		return types.SlashResolveResult{}, fmt.Errorf("builtin prompt %q not yet implemented in Go", cmd.Name)
	}
}

func resolveInit(args string) (types.SlashResolveResult, error) {
	body, err := readBundledText("init.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("builtin prompt init: %w", err)
	}
	if os.Getenv("CLAUDE_CODE_SKIP_INIT_PHASE0") == "1" {
		re := regexp.MustCompile(`(?s)## Phase 0:.*?\n## Phase 1:`)
		body = re.ReplaceAllString(body, "## Phase 1:")
	}
	text := body
	if a := strings.TrimSpace(args); a != "" {
		text = appendUserSection(text, a)
	}
	return types.SlashResolveResult{
		UserText: text,
		Source:   types.SlashResolveBuiltinPrompt,
	}, nil
}
