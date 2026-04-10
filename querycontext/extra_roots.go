package querycontext

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goc/types"
)

// ParseAdditionalWorkingDirsJSON extracts directory paths from ToolPermissionContext.additionalWorkingDirectories.
// TS uses a ReadonlyMap<string, AdditionalWorkingDirectory>; JSON may be an object keyed by path, an array of paths, or empty.
func ParseAdditionalWorkingDirsJSON(raw json.RawMessage) []string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return normalizeRootPaths(arr)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			if s := strings.TrimSpace(k); s != "" {
				keys = append(keys, s)
			}
		}
		sort.Strings(keys)
		return normalizeRootPaths(keys)
	}
	return nil
}

func normalizeRootPaths(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func parseCommaSeparatedEnv(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, string(os.PathListSeparator))
	if len(parts) == 1 && !strings.Contains(v, string(os.PathListSeparator)) {
		parts = strings.Split(v, ",")
	}
	return normalizeRootPaths(parts)
}

// ExtraClaudeMdRootsForFetch returns paths to pass as [FetchOpts.ExtraClaudeMdRoots]:
// optional [types.ProcessUserInputContextData.ToolPermissionContext], then
// GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS and CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma- or PATH-style list).
func ExtraClaudeMdRootsForFetch(rc *types.ProcessUserInputContextData) []string {
	var merged []string
	if rc != nil && rc.ToolPermissionContext != nil {
		merged = append(merged, ParseAdditionalWorkingDirsJSON(rc.ToolPermissionContext.AdditionalWorkingDirectories)...)
	}
	merged = append(merged, parseCommaSeparatedEnv("GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS")...)
	merged = append(merged, parseCommaSeparatedEnv("CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS")...)
	return dedupeAbsPaths(merged)
}

func dedupeAbsPaths(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		ad, err := filepath.Abs(strings.TrimSpace(s))
		if err != nil || ad == "" {
			continue
		}
		if _, ok := seen[ad]; ok {
			continue
		}
		seen[ad] = struct{}{}
		out = append(out, ad)
	}
	return out
}
