package slashresolve

import (
	"fmt"

	"goc/types"
)

// Feature-gated bundled skills whose TS bodies are large or dynamic; embedded
// markdown is produced by claude-code/scripts/dump-bundled-prompts-for-go.ts.
// Args append as ## User Request (same pattern as keybindings-help listing).

func resolveBundledMarkdownUserRequest(mdName string, args string) (types.SlashResolveResult, error) {
	body, err := readBundledText(mdName)
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled %s: %w", mdName, err)
	}
	return types.SlashResolveResult{
		UserText: appendUserSection(body, args),
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
