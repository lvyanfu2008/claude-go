package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"goc/types"

	"gopkg.in/yaml.v3"
)

// Workflow *listing* from disk (Phase P6): implementation保留，但产品路径 **延后**（[DefaultLoadOptions] 不启用 [LoadOptions.WorkflowScripts]；见 docs/plans/goc-load-all-commands.md）。显式设置 WorkflowScripts=true 时仍会扫描 `.claude/workflows` 等。多步执行仍依赖 TS stub / 未实现。

// workflowFileMeta is a minimal root object for `.yaml`/`.yml`/`.json` workflow definitions
// (see docs/features/workflow-scripts.md — full execution engine is TS stub).
type workflowFileMeta struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

func loadWorkflowCommands(cwd string, opts LoadOptions) []types.Command {
	if !opts.WorkflowScripts {
		return nil
	}
	policyOff := opts.disablePolicySkillsEffective()
	projectSettingsEnabled := opts.isSettingSourceEnabled("projectSettings")
	seen := make(map[string]struct{})
	var out []types.Command

	appendUnique := func(cmds []types.Command) {
		for _, c := range cmds {
			if c.Name == "" {
				continue
			}
			if _, ok := seen[c.Name]; ok {
				continue
			}
			seen[c.Name] = struct{}{}
			out = append(out, c)
		}
	}

	if opts.effectiveBare() {
		if len(opts.AddSkillDirs) == 0 || !projectSettingsEnabled {
			return nil
		}
		for _, root := range opts.AddSkillDirs {
			abs, err := filepath.Abs(root)
			if err != nil {
				continue
			}
			dir := filepath.Join(abs, ".claude", "workflows")
			cmds, err := loadWorkflowsFromDir(dir, "projectSettings")
			if err != nil {
				continue
			}
			appendUnique(cmds)
		}
		return out
	}

	if opts.SkillsPluginOnlyLocked {
		if !policyOff {
			dir := filepath.Join(ManagedFilePath(), ".claude", "workflows")
			cmds, _ := loadWorkflowsFromDir(dir, "policySettings")
			appendUnique(cmds)
		}
		return out
	}

	if !policyOff {
		dir := filepath.Join(ManagedFilePath(), ".claude", "workflows")
		cmds, _ := loadWorkflowsFromDir(dir, "policySettings")
		appendUnique(cmds)
	}

	if cfgHome := ClaudeConfigHome(); cfgHome != "" && opts.isSettingSourceEnabled("userSettings") {
		dir := filepath.Join(cfgHome, "workflows")
		cmds, _ := loadWorkflowsFromDir(dir, "userSettings")
		appendUnique(cmds)
	}

	if projectSettingsEnabled {
		projDirs, err := projectWorkflowDirs(cwd, opts.sessionRootForBoundary(cwd))
		if err == nil {
			for _, d := range projDirs {
				cmds, err := loadWorkflowsFromDir(d, "projectSettings")
				if err != nil {
					continue
				}
				appendUnique(cmds)
			}
		}
		for _, root := range opts.AddSkillDirs {
			abs, err := filepath.Abs(root)
			if err != nil {
				continue
			}
			dir := filepath.Join(abs, ".claude", "workflows")
			cmds, err := loadWorkflowsFromDir(dir, "projectSettings")
			if err != nil {
				continue
			}
			appendUnique(cmds)
		}
	}

	return out
}

func loadWorkflowsFromDir(dir string, source string) ([]types.Command, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var out []types.Command
	for _, name := range names {
		path := filepath.Join(dir, name)
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		cmd, err := commandFromWorkflowFile(path, source)
		if err != nil {
			continue
		}
		out = append(out, cmd)
	}
	return out, nil
}

func commandFromWorkflowFile(path string, source string) (types.Command, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return types.Command{}, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var meta workflowFileMeta
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &meta); err != nil {
			return types.Command{}, fmt.Errorf("parse workflow json %s: %w", path, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(raw, &meta); err != nil {
			return types.Command{}, fmt.Errorf("parse workflow yaml %s: %w", path, err)
		}
	default:
		return types.Command{}, fmt.Errorf("unsupported workflow extension: %s", path)
	}
	base := strings.TrimSuffix(filepath.Base(path), ext)
	name := sanitizeWorkflowName(strings.TrimSpace(meta.Name))
	if name == "" {
		name = sanitizeWorkflowName(base)
	}
	if name == "" {
		return types.Command{}, fmt.Errorf("empty workflow name: %s", path)
	}
	desc := strings.TrimSpace(meta.Description)
	if desc == "" {
		desc = fmt.Sprintf("Workflow (%s)", filepath.Base(path))
	}
	lf := "workflow"
	pm := "running"
	src := source
	t := "prompt"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        name,
			Description:                 desc,
			HasUserSpecifiedDescription: boolPtr(strings.TrimSpace(meta.Description) != ""),
			UserInvocable:               boolPtr(true),
			IsHidden:                    boolPtr(false),
			LoadedFrom:                  &lf,
		},
		Type:            t,
		ProgressMessage: &pm,
		Source:          &src,
	}, nil
}

func sanitizeWorkflowName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) && r < unicode.MaxASCII:
			b.WriteRune(r)
			lastHyphen = false
		case unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() > 0 && !lastHyphen {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	return out
}
