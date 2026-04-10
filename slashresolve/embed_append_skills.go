package slashresolve

import (
	"fmt"
	"strings"

	"goc/types"
)

// --- remember (src/skills/bundled/remember.ts) ---

func resolveRemember(args string) (types.SlashResolveResult, error) {
	body, err := readBundledText("remember.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled remember: %w", err)
	}
	if s := strings.TrimSpace(args); s != "" {
		body += "\n## Additional context from user\n\n" + s
	}
	return types.SlashResolveResult{UserText: body, Source: types.SlashResolveBundledEmbed}, nil
}

// --- simplify (src/skills/bundled/simplify.ts) ---

func resolveSimplify(args string) (types.SlashResolveResult, error) {
	body, err := readBundledText("simplify.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled simplify: %w", err)
	}
	if s := strings.TrimSpace(args); s != "" {
		body += "\n\n## Additional Focus\n\n" + s
	}
	return types.SlashResolveResult{UserText: body, Source: types.SlashResolveBundledEmbed}, nil
}

// --- stuck (src/skills/bundled/stuck.ts) ---

func resolveStuck(args string) (types.SlashResolveResult, error) {
	body, err := readBundledText("stuck.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled stuck: %w", err)
	}
	if s := strings.TrimSpace(args); s != "" {
		body += "\n## User-provided context\n\n" + s + "\n"
	}
	return types.SlashResolveResult{UserText: body, Source: types.SlashResolveBundledEmbed}, nil
}

// --- keybindings-help (src/skills/bundled/keybindings.ts) — args use ## User Request ---

func resolveKeybindingsHelp(args string) (types.SlashResolveResult, error) {
	body, err := readBundledText("keybindings-help.md")
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled keybindings-help: %w", err)
	}
	return types.SlashResolveResult{
		UserText: appendUserSection(body, args),
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
