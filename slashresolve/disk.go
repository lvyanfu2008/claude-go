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

// skillFileYAML is frontmatter for SKILL.md (mirrors commands.skillFrontmatter for richer parsing).
type skillFileYAML struct {
	Name                   string      `yaml:"name"`
	Description            string      `yaml:"description"`
	AllowedTools           interface{} `yaml:"allowed-tools"`
	UserInvocable          *bool       `yaml:"user-invocable"`
	WhenToUse              string      `yaml:"when_to_use"`
	DisableModelInvocation *bool       `yaml:"disable-model-invocation"`
	Model                  string      `yaml:"model"`
	Version                string      `yaml:"version"`
	ArgumentHint           string      `yaml:"argument-hint"`
	Context                string      `yaml:"context"`
	Agent                  string      `yaml:"agent"`
	Effort                 interface{} `yaml:"effort"`
	Paths                  interface{} `yaml:"paths"`
	Hooks                  interface{} `yaml:"hooks"`
	Shell                  string      `yaml:"shell"`
	Arguments              interface{} `yaml:"arguments"`
	Engine                 string      `yaml:"engine"`
}

// ResolveDiskSkill builds SlashResolveResult from a disk skill Command (prompt + SkillRoot).
// sessionID is substituted for ${CLAUDE_SESSION_ID}; use empty to replace with empty string.
// Shell command substitutions (${...} and backtick patterns) are executed via sh -c.
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

		// Starlark engine path: execute the script body directly.
		if fm.Engine == "starlark" {
			sctx := &StarlarkContext{
				SessionID: sessionID,
				SkillRoot: root,
			}
			result, err := ExecuteStarlarkSkill(string(body), args, sctx, mdPath)
			text := result
			if err != nil {
				text = fmt.Sprintf("## Starlark Error\n\n%v\n\n---\n\n%s", err, string(body))
			}
			if root != "" {
				text = fmt.Sprintf("Base directory for this skill: %s\n\n%s", root, text)
			}
			text = strings.ReplaceAll(text, "${CLAUDE_SKILL_DIR}", strings.ReplaceAll(root, "\\", "/"))
			text = strings.ReplaceAll(text, "${CLAUDE_SESSION_ID}", sessionID)
			allowedTools := parseAllowedToolsFromFM(fm.AllowedTools)
			res := types.SlashResolveResult{
				UserText:     text,
				AllowedTools: allowedTools,
				Source:       types.SlashResolveDisk,
			}
			return res, nil
		}

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

	// Execute shell command substitutions (${...} and backtick patterns) in the resolved prompt.
	if shellResult, shellErr := ExecuteShellCommandsInPrompt(final); shellErr != nil {
		// Non-fatal: substitute what we can, log errors in user text.
		final = shellResult
		_ = shellErr // errors are embedded in-place by shell_exec.go
	} else {
		final = shellResult
	}

	// Merge frontmatter overrides from SKILL.md into the result.
	allowedTools := append([]string(nil), cmd.AllowedTools...)
	if len(allowedTools) == 0 && fm.AllowedTools != nil {
		allowedTools = parseAllowedToolsFromFM(fm.AllowedTools)
	}
	res := types.SlashResolveResult{
		UserText:     final,
		AllowedTools: allowedTools,
		Source:       types.SlashResolveDisk,
	}
	// Model: prefer Command (already parsed by skill_md.go), fallback to frontmatter.
	if cmd.Model != nil {
		m := *cmd.Model
		res.Model = &m
	} else if fm.Model != "" && !strings.EqualFold(fm.Model, "inherit") {
		m := fm.Model
		res.Model = &m
	}
	if cmd.Effort != nil {
		ev := *cmd.Effort
		res.Effort = &ev
	}
	// Context / Agent for fork execution.
	if cmd.Context != nil {
		c := *cmd.Context
		res.Context = &c
	} else if fm.Context == "fork" {
		c := "fork"
		res.Context = &c
	}
	if cmd.Agent != nil {
		a := *cmd.Agent
		res.Agent = &a
	} else if fm.Agent != "" {
		a := fm.Agent
		res.Agent = &a
	}
	return res, nil
}

// parseAllowedToolsFromFM is a local copy to avoid import cycle with commands package.
func parseAllowedToolsFromFM(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		if strings.Contains(s, ",") {
			var out []string
			for _, p := range strings.Split(s, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return strings.Fields(s)
	case []interface{}:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	}
	return nil
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
