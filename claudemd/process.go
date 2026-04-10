package claudemd

import (
	"os"
	"path/filepath"
	"strings"
)

// ProcessMemoryFile mirrors claudemd.ts processMemoryFile (sync).
func ProcessMemoryFile(filePath string, typ MemoryType, processed map[string]struct{}, includeExternal bool, originalCwd string, depth int, parent string, exclude *ExcludeChecker) []MemoryFileInfo {
	norm := NormalizePathForComparison(filePath)
	if _, ok := processed[norm]; ok || depth >= maxIncludeDepth {
		return nil
	}
	if exclude != nil && exclude.IsExcluded(filePath, typ) {
		return nil
	}
	resolvedPath, isSym := SafeResolvePath(filePath)
	processed[norm] = struct{}{}
	if isSym {
		processed[NormalizePathForComparison(resolvedPath)] = struct{}{}
	}
	info, incPaths := ReadMemoryFileFromDisk(filePath, typ, resolvedPath)
	if info == nil || strings.TrimSpace(info.Content) == "" {
		return nil
	}
	if parent != "" {
		info.Parent = parent
	}
	var result []MemoryFileInfo
	result = append(result, *info)
	for _, inc := range incPaths {
		if !PathInWorkingPath(inc, originalCwd) && !includeExternal {
			continue
		}
		sub := ProcessMemoryFile(inc, typ, processed, includeExternal, originalCwd, depth+1, filePath, exclude)
		result = append(result, sub...)
	}
	return result
}

// ProcessMdRules mirrors processMdRules (conditionalRule: false for main tree).
func ProcessMdRules(rulesDir string, typ MemoryType, processed map[string]struct{}, includeExternal bool, originalCwd string, conditionalRule bool, visited map[string]struct{}, exclude *ExcludeChecker) []MemoryFileInfo {
	if visited == nil {
		visited = map[string]struct{}{}
	}
	if _, ok := visited[rulesDir]; ok {
		return nil
	}
	resolvedRulesDir, isSym := SafeResolvePath(rulesDir)
	visited[rulesDir] = struct{}{}
	if isSym {
		visited[resolvedRulesDir] = struct{}{}
	}
	entries, err := os.ReadDir(resolvedRulesDir)
	if err != nil {
		return nil
	}
	var result []MemoryFileInfo
	for _, entry := range entries {
		full := filepath.Join(resolvedRulesDir, entry.Name())
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		if st.IsDir() {
			result = append(result, ProcessMdRules(full, typ, processed, includeExternal, originalCwd, conditionalRule, visited, exclude)...)
			continue
		}
		if !st.Mode().IsRegular() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		files := ProcessMemoryFile(full, typ, processed, includeExternal, originalCwd, 0, "", exclude)
		for _, f := range files {
			if conditionalRule {
				if len(f.Globs) > 0 {
					result = append(result, f)
				}
			} else if len(f.Globs) == 0 {
				result = append(result, f)
			}
		}
	}
	return result
}

func directoryChainUp(abs string) []string {
	cur := filepath.Clean(abs)
	var chain []string
	for {
		chain = append(chain, cur)
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return chain
}
