package commands

import (
	"path/filepath"

	"goc/types"
)

// fileIdentity mirrors TS getFileIdentity (realpath) for SKILL.md / markdown paths; fallback to absolute path.
func fileIdentity(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(abs)
}

// dedupeSkillEntries drops later entries that resolve to the same file path as an earlier one (order-preserving first wins).
func dedupeSkillEntries(entries []SkillLoadEntry) []types.Command {
	seen := make(map[string]struct{})
	out := make([]types.Command, 0, len(entries))
	for _, e := range entries {
		id := fileIdentity(e.MarkdownPath)
		if id == "" {
			out = append(out, e.Cmd)
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, e.Cmd)
	}
	return out
}
