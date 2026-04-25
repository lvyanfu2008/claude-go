package commands

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"goc/types"
	"goc/utils"

	"gopkg.in/yaml.v3"
)

// SkillLoadEntry pairs a loaded prompt command with the markdown file path used for deduplication (TS getFileIdentity / realpath).
type SkillLoadEntry struct {
	Cmd          types.Command
	MarkdownPath string
}

// skillFrontmatter is a subset of TS frontmatter for SKILL.md (src/skills/loadSkillsDir.ts).
type skillFrontmatter struct {
	Name                   string      `yaml:"name"`
	Description            *string     `yaml:"description"`
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
}

func SplitYAMLFrontmatter(raw []byte) (yamlBytes []byte, body []byte, ok bool) {
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

func parseAllowedTools(v interface{}) []string {
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
	default:
		return nil
	}
}

func extractDescriptionFromMarkdown(content string, fallback string) string {
	if fallback == "" {
		fallback = "Skill"
	}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			t = strings.TrimSpace(strings.TrimLeft(t, "#"))
		}
		if len(t) > 100 {
			return t[:97] + "..."
		}
		return t
	}
	return fallback
}

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

// commandFromSkillMD builds types.Command from a skill directory and SKILL.md bytes (data-only; no getPromptForCommand).
func commandFromSkillMD(skillDirName, skillRoot, markdownPath string, body []byte, source string) (types.Command, error) {
	return commandFromSkillMarkdown(skillDirName, skillRoot, markdownPath, body, source, "skills", "Skill")
}

// commandFromSkillMarkdown sets CommandBase.LoadedFrom to loadedFrom (e.g. skills, commands_DEPRECATED).
func commandFromSkillMarkdown(
	cmdName string,
	skillRoot string,
	markdownPath string,
	body []byte,
	source string,
	loadedFrom string,
	descriptionFallback string,
) (types.Command, error) {
	yamlBytes, mdBody, hasFM := SplitYAMLFrontmatter(body)
	if !hasFM {
		return types.Command{}, fmt.Errorf("markdown missing YAML frontmatter: %s", markdownPath)
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal(yamlBytes, &fm); err != nil {
		return types.Command{}, fmt.Errorf("parse frontmatter %s: %w", markdownPath, err)
	}
	content := string(mdBody)
	desc := ""
	if fm.Description != nil && strings.TrimSpace(*fm.Description) != "" {
		d := strings.TrimSpace(*fm.Description)
		desc = d
	} else {
		desc = extractDescriptionFromMarkdown(content, descriptionFallback)
	}
	hasUserDesc := fm.Description != nil && strings.TrimSpace(*fm.Description) != ""
	userInv := true
	if fm.UserInvocable != nil {
		userInv = *fm.UserInvocable
	}
	dmi := false
	if fm.DisableModelInvocation != nil {
		dmi = *fm.DisableModelInvocation
	}
	cl := len(content)
	lf := loadedFrom
	src := source
	pm := "running"
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name:                        cmdName,
			Description:                 desc,
			HasUserSpecifiedDescription: boolPtr(hasUserDesc),
			UserInvocable:               boolPtr(userInv),
			IsHidden:                    boolPtr(!userInv),
			LoadedFrom:                  &lf,
			DisableModelInvocation:      boolPtr(dmi),
		},
		Type:            "prompt",
		ProgressMessage: &pm,
		ContentLength:   &cl,
		AllowedTools:    parseAllowedTools(fm.AllowedTools),
		Model:           modelPtrFromFrontmatter(fm.Model),
		Source:          &src,
	}
	if skillRoot != "" {
		cmd.SkillRoot = &skillRoot
	}
	if fm.WhenToUse != "" {
		cmd.WhenToUse = strPtr(fm.WhenToUse)
	}
	if fm.Version != "" {
		cmd.Version = strPtr(fm.Version)
	}
	if fm.ArgumentHint != "" {
		cmd.ArgumentHint = strPtr(fm.ArgumentHint)
	}
	if fm.Context == "fork" {
		c := "fork"
		cmd.Context = &c
	}
	if fm.Agent != "" {
		cmd.Agent = strPtr(fm.Agent)
	}
	if fm.Effort != nil {
		if ev, ok := utils.ParseEffortValueYAML(fm.Effort); ok {
			evCopy := ev
			cmd.Effort = &evCopy
		}
	}
	// TS loadSkillsFromCommandsDir forces paths: undefined for legacy /commands (not path-conditional there).
	if loadedFrom != "commands_DEPRECATED" {
		if p := ParseSkillPaths(fm.Paths); len(p) > 0 {
			cmd.Paths = p
		}
	}
	return cmd, nil
}

func emptyStrPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func modelPtrFromFrontmatter(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "inherit") {
		return nil
	}
	return &s
}

// loadSkillsFromDir scans baseDir for subdirectories containing SKILL.md (src/skills/loadSkillsDir.ts shape).
//
// Entry order follows [os.ReadDir]: **lexicographically sorted by filename** (stable, cross-platform).
// Node [fs.readdir] does not guarantee order; TS loadSkillsFromSkillsDir uses that order as-is, so
// within-directory sequencing may differ from Go on some hosts — use sorted skill folder names if parity matters.
func loadSkillsFromDir(baseDir, source string) ([]SkillLoadEntry, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []SkillLoadEntry
	for _, ent := range entries {
		skillName := ent.Name()
		skillRoot := filepath.Join(baseDir, skillName)
		if !ent.IsDir() {
			// TS loadSkillsFromSkillsDir: directory OR symlink (resolved to directory).
			if ent.Type()&fs.ModeSymlink == 0 {
				continue
			}
			st, err := os.Stat(skillRoot)
			if err != nil || !st.IsDir() {
				continue
			}
		}
		skillPath := filepath.Join(skillRoot, "SKILL.md")
		raw, err := os.ReadFile(skillPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		cmd, err := commandFromSkillMD(skillName, skillRoot, skillPath, raw, source)
		if err != nil {
			return nil, err
		}
		absPath, err := filepath.Abs(skillPath)
		if err != nil {
			absPath = skillPath
		}
		out = append(out, SkillLoadEntry{Cmd: cmd, MarkdownPath: absPath})
	}
	return out, nil
}
