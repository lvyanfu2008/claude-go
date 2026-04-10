package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type legacyFile struct {
	path    string
	baseDir string
	source  string
}

type legacyRoot struct {
	dir    string
	source string
}

func isSkillMarkdownBase(name string) bool {
	return strings.EqualFold(name, "SKILL.md")
}

func walkMarkdownFiles(root string) ([]string, error) {
	return findMarkdownFilesNative(root)
}

// buildNamespace mirrors src/skills/loadSkillsDir.ts buildNamespace (path segments joined by ':').
func buildNamespace(targetDir, baseDir string) string {
	b := filepath.Clean(baseDir)
	t := filepath.Clean(targetDir)
	rel, err := filepath.Rel(b, t)
	if err != nil || rel == "." {
		return ""
	}
	return strings.ReplaceAll(rel, string(filepath.Separator), ":")
}

func legacyCommandName(f legacyFile) string {
	if isSkillMarkdownBase(filepath.Base(f.path)) {
		skillDir := filepath.Dir(f.path)
		parentOfSkill := filepath.Dir(skillDir)
		cmdBase := filepath.Base(skillDir)
		ns := buildNamespace(parentOfSkill, f.baseDir)
		if ns != "" {
			return ns + ":" + cmdBase
		}
		return cmdBase
	}
	dir := filepath.Dir(f.path)
	base := filepath.Base(f.path)
	ext := filepath.Ext(base)
	if strings.EqualFold(ext, ".md") {
		base = strings.TrimSuffix(base, ext)
	}
	ns := buildNamespace(dir, f.baseDir)
	if ns != "" {
		return ns + ":" + base
	}
	return base
}

func transformLegacySkillFiles(files []legacyFile) []legacyFile {
	byDir := make(map[string][]legacyFile)
	for _, f := range files {
		d := filepath.Dir(f.path)
		byDir[d] = append(byDir[d], f)
	}
	dirs := make([]string, 0, len(byDir))
	for d := range byDir {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	out := make([]legacyFile, 0, len(files))
	for _, d := range dirs {
		group := byDir[d]
		var skills []legacyFile
		var nonSkill []legacyFile
		for _, f := range group {
			if isSkillMarkdownBase(filepath.Base(f.path)) {
				skills = append(skills, f)
			} else {
				nonSkill = append(nonSkill, f)
			}
		}
		if len(skills) > 0 {
			sort.Slice(skills, func(i, j int) bool { return skills[i].path < skills[j].path })
			out = append(out, skills[0])
		} else {
			sort.Slice(nonSkill, func(i, j int) bool { return nonSkill[i].path < nonSkill[j].path })
			out = append(out, nonSkill...)
		}
	}
	return out
}

// loadLegacyCommandEntries mirrors loadSkillsFromCommandsDir in src/skills/loadSkillsDir.ts (subset: filesystem walk, no ripgrep).
func loadLegacyCommandEntries(cwd string, opts LoadOptions) ([]SkillLoadEntry, error) {
	if opts.SkillsPluginOnlyLocked {
		return nil, nil
	}
	var roots []legacyRoot
	roots = append(roots, legacyRoot{
		dir:    filepath.Join(ManagedFilePath(), ".claude", "commands"),
		source: "policySettings",
	})
	if h := ClaudeConfigHome(); h != "" && opts.isSettingSourceEnabled("userSettings") {
		roots = append(roots, legacyRoot{
			dir:    filepath.Join(h, "commands"),
			source: "userSettings",
		})
	}
	if opts.isSettingSourceEnabled("projectSettings") {
		proj, err := projectClaudeSubdirs(cwd, "commands", opts.sessionRootForBoundary(cwd))
		if err != nil {
			return nil, err
		}
		for _, d := range proj {
			roots = append(roots, legacyRoot{dir: d, source: "projectSettings"})
		}
	}

	var files []legacyFile
	for _, r := range roots {
		paths, err := walkMarkdownFiles(r.dir)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			files = append(files, legacyFile{path: p, baseDir: r.dir, source: r.source})
		}
	}
	files = transformLegacySkillFiles(files)

	var entries []SkillLoadEntry
	for _, f := range files {
		raw, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}
		name := legacyCommandName(f)
		skillRoot := ""
		if isSkillMarkdownBase(filepath.Base(f.path)) {
			skillRoot = filepath.Dir(f.path)
		}
		cmd, err := commandFromSkillMarkdown(name, skillRoot, f.path, raw, f.source, "commands_DEPRECATED", "Custom command")
		if err != nil {
			continue
		}
		absPath, err := filepath.Abs(f.path)
		if err != nil {
			absPath = f.path
		}
		entries = append(entries, SkillLoadEntry{Cmd: cmd, MarkdownPath: absPath})
	}
	return entries, nil
}
