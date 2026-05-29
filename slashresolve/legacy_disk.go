package slashresolve

import (
	"fmt"
	"os"
	"strings"

	"goc/types"
)

// ResolveLegacyMarkdownCommand reads a standalone .md file (from .claude/commands/
// or .harness/commands/) and returns its body content. Unlike ResolveDiskSkill,
// the file may not have YAML frontmatter — the content is returned as-is after
// argument and shell substitution.
func ResolveLegacyMarkdownCommand(cmd types.Command, args string, sessionID string) (types.SlashResolveResult, error) {
	if cmd.LegacyMarkdownPath == nil || strings.TrimSpace(*cmd.LegacyMarkdownPath) == "" {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: missing LegacyMarkdownPath")
	}
	raw, err := os.ReadFile(*cmd.LegacyMarkdownPath)
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: read %s: %w", *cmd.LegacyMarkdownPath, err)
	}

	// Strip YAML frontmatter if present; otherwise use full content as body.
	_, body, _ := splitYAMLFrontmatter(raw)
	markdown := string(body)

	final := SubstituteArguments(markdown, args, true, cmd.ArgNames)
	final = strings.ReplaceAll(final, "${CLAUDE_SESSION_ID}", sessionID)

	if shellResult, shellErr := ExecuteShellCommandsInPrompt(final); shellErr != nil {
		final = shellResult
	} else {
		final = shellResult
	}

	res := types.SlashResolveResult{
		UserText:     final,
		AllowedTools: append([]string(nil), cmd.AllowedTools...),
		Source:       types.SlashResolveDisk,
	}
	if cmd.Model != nil {
		m := *cmd.Model
		res.Model = &m
	}
	if cmd.Effort != nil {
		ev := *cmd.Effort
		res.Effort = &ev
	}
	if cmd.Context != nil {
		c := *cmd.Context
		res.Context = &c
	}
	if cmd.Agent != nil {
		a := *cmd.Agent
		res.Agent = &a
	}
	return res, nil
}
