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

func init() {
	RegisterBundledSkill(BundledSkillDefinition{
		Name:     "hunter",
		Resolver: func(args string, opt *BundledResolveOptions) (types.SlashResolveResult, error) { return resolveBundledMarkdownUserRequest("hunter.md", args) },
		FeatureGate: "REVIEW_ARTIFACT",
	})
	RegisterBundledSkill(BundledSkillDefinition{
		Name:     "run-skill-generator",
		Resolver: func(args string, opt *BundledResolveOptions) (types.SlashResolveResult, error) { return resolveBundledMarkdownUserRequest("run-skill-generator.md", args) },
		FeatureGate: "RUN_SKILL_GENERATOR",
	})
}
