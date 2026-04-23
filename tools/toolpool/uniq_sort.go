package toolpool

import (
	"sort"

	"goc/types"
)

// ByName mirrors (a, b) => a.name.localeCompare(b.name) in src/tools.ts / toolPool.ts
// for ASCII tool identifiers (same ordering as Node default locale for [A-Za-z0-9_] names).
func ByNameLess(a, b types.ToolSpec) bool {
	return a.Name < b.Name
}

// UniqByName mirrors lodash uniqBy(tools, 'name'): first occurrence wins; later duplicates dropped.
// Insertion order is preserved among kept entries (src/tools.ts assembleToolPool concat + uniqBy).
func UniqByName(tools []types.ToolSpec) []types.ToolSpec {
	if len(tools) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tools))
	out := make([]types.ToolSpec, 0, len(tools))
	for _, t := range tools {
		if _, ok := seen[t.Name]; ok {
			continue
		}
		seen[t.Name] = struct{}{}
		out = append(out, t)
	}
	return out
}

func sortToolsByNameInPlace(tools []types.ToolSpec) {
	sort.Slice(tools, func(i, j int) bool {
		return ByNameLess(tools[i], tools[j])
	})
}
