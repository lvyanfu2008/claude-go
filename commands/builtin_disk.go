package commands

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"goc/types"
)

// loadBuiltinCommandsDiskOverlay appends commands from goc/commands/builtin*/ (this package directory):
//   - Any *.json file: JSON array of types.Command, or a single object
//   - Any *.md except README.md: SKILL-style YAML frontmatter + body (same as skill dirs)
//
// Only immediate subdirectories of the commands/ package whose name has prefix "builtin" are scanned
// (e.g. commands/builtin_overlay/). Names already present in seen are skipped (embedded COMMANDS wins on conflict).
func loadBuiltinCommandsDiskOverlay(seen map[string]struct{}) []types.Command {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	root := filepath.Dir(file)
	return loadBuiltinPrefixDirs(root, seen)
}

func loadBuiltinPrefixDirs(commandsPackageDir string, seen map[string]struct{}) []types.Command {
	entries, err := os.ReadDir(commandsPackageDir)
	if err != nil {
		return nil
	}
	var out []types.Command
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasPrefix(name, "builtin") {
			continue
		}
		dir := filepath.Join(commandsPackageDir, name)
		out = append(out, scanBuiltinOverlayDir(dir, seen)...)
	}
	return out
}

func scanBuiltinOverlayDir(dir string, seen map[string]struct{}) []types.Command {
	var out []types.Command
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := d.Name()
		if strings.EqualFold(base, "README.md") {
			return nil
		}
		low := strings.ToLower(path)
		switch {
		case strings.HasSuffix(low, ".json"):
			raw, errR := os.ReadFile(path)
			if errR != nil {
				return nil
			}
			appendCommandsFromJSON(raw, &out, seen)
		case strings.HasSuffix(low, ".md"):
			raw, errR := os.ReadFile(path)
			if errR != nil {
				return nil
			}
			cmdName := skillNameFromMarkdownPath(path)
			cmd, errC := commandFromSkillMarkdown(cmdName, filepath.Dir(path), path, raw, "builtin", "skills", "Skill")
			if errC != nil {
				return nil
			}
			appendIfNew(&out, seen, cmd)
		}
		return nil
	})
	return out
}

func skillNameFromMarkdownPath(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	if strings.EqualFold(base, "SKILL.md") {
		return filepath.Base(dir)
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func appendCommandsFromJSON(raw []byte, out *[]types.Command, seen map[string]struct{}) {
	var batch []types.Command
	if err := json.Unmarshal(raw, &batch); err == nil && len(batch) > 0 {
		for _, c := range batch {
			appendIfNew(out, seen, c)
		}
		return
	}
	var one types.Command
	if err := json.Unmarshal(raw, &one); err == nil && strings.TrimSpace(one.Name) != "" {
		appendIfNew(out, seen, one)
	}
}

func appendIfNew(out *[]types.Command, seen map[string]struct{}, c types.Command) {
	n := strings.TrimSpace(c.Name)
	if n == "" {
		return
	}
	if _, ok := seen[n]; ok {
		return
	}
	seen[n] = struct{}{}
	*out = append(*out, c)
}
