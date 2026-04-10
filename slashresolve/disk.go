package slashresolve

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/types"

	"gopkg.in/yaml.v3"
)

// skillFileYAML is frontmatter for SKILL.md (arguments extend skill_md.go).
type skillFileYAML struct {
	Arguments interface{} `yaml:"arguments"`
}

// ResolveDiskSkill builds SlashResolveResult from a disk skill Command (prompt + SkillRoot).
// sessionID is substituted for ${CLAUDE_SESSION_ID}; use empty to replace with empty string.
// Does not run executeShellCommandsInPrompt (TS-only); see docs/plans/go-slash-resolve.md.
func ResolveDiskSkill(cmd types.Command, args string, sessionID string) (types.SlashResolveResult, error) {
	if cmd.Type != "prompt" {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: command type %q not prompt", cmd.Type)
	}
	if cmd.SkillRoot == nil || strings.TrimSpace(*cmd.SkillRoot) == "" {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: missing SkillRoot")
	}
	root := filepath.Clean(*cmd.SkillRoot)
	mdPath := filepath.Join(root, "SKILL.md")
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: read %s: %w", mdPath, err)
	}

	yamlBytes, body, ok := splitYAMLFrontmatter(raw)
	if !ok {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: missing YAML frontmatter in %s", mdPath)
	}
	var fm skillFileYAML
	if err := yaml.Unmarshal(yamlBytes, &fm); err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve: yaml %s: %w", mdPath, err)
	}
	argNames := ParseArgumentNames(fm.Arguments)

	markdown := string(body)
	final := markdown
	if root != "" {
		final = fmt.Sprintf("Base directory for this skill: %s\n\n%s", root, markdown)
	}
	final = SubstituteArguments(final, args, true, argNames)

	if root != "" && strings.Contains(final, "${CLAUDE_SKILL_DIR}") {
		sd := strings.ReplaceAll(root, `\`, `/`)
		final = strings.ReplaceAll(final, "${CLAUDE_SKILL_DIR}", sd)
	}
	final = strings.ReplaceAll(final, "${CLAUDE_SESSION_ID}", sessionID)

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
	return res, nil
}

func splitYAMLFrontmatter(raw []byte) (yamlBytes []byte, body []byte, ok bool) {
	s := bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	if !bytes.HasPrefix(s, []byte("---")) {
		return nil, s, false
	}
	rest := s[3:]
	if len(rest) > 0 && rest[0] == '\r' {
		rest = rest[1:]
	}
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	sep := []byte("\n---\n")
	idx := bytes.Index(rest, sep)
	if idx < 0 {
		sep = []byte("\n---\r\n")
		idx = bytes.Index(rest, sep)
	}
	if idx < 0 {
		return nil, s, false
	}
	return rest[:idx], rest[idx+len(sep):], true
}
