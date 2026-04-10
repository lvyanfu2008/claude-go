package slashresolve

import (
	"fmt"
	"strings"

	"goc/types"
)

// updateConfigHooksOnlyPrefix matches src/skills/bundled/updateConfig.ts getPromptForCommand.
const updateConfigHooksOnlyPrefix = "[hooks-only]"

// resolveUpdateConfig mirrors TS registerUpdateConfigSkill → getPromptForCommand.
// Body text is embedded under bundleddata/ (from scripts/dump-bundled-prompts-for-go.ts).
func resolveUpdateConfig(args string) (types.SlashResolveResult, error) {
	if strings.HasPrefix(args, updateConfigHooksOnlyPrefix) {
		body, err := readBundledText("update-config-hooks.md")
		if err != nil {
			return types.SlashResolveResult{}, fmt.Errorf("bundled update-config hooks-only: %w", err)
		}
		rest := strings.TrimSpace(strings.TrimPrefix(args, updateConfigHooksOnlyPrefix))
		return types.SlashResolveResult{
			UserText: appendUserSection(body, rest),
			Source:   types.SlashResolveBundledEmbed,
		}, nil
	}

	body, err := readBundledText("update-config.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled update-config: %w", err)
	}
	return types.SlashResolveResult{
		UserText: appendUserSection(body, args),
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
