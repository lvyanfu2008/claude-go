// Heuristics aligned with claude-code/src/utils/memoryFileDetection.ts (subset for collapseReadSearch).
package messagerow

import (
	"path/filepath"
	"strings"
)

func getFilePathFromToolInput(m map[string]any) string {
	if m == nil {
		return ""
	}
	if fp, ok := m["file_path"].(string); ok && strings.TrimSpace(fp) != "" {
		return fp
	}
	if p, ok := m["path"].(string); ok && strings.TrimSpace(p) != "" {
		return p
	}
	return ""
}

func toComparablePath(p string) string {
	p = filepath.ToSlash(filepath.Clean(strings.TrimSpace(p)))
	if p == "" {
		return ""
	}
	return strings.ToLower(p)
}

// isAutoManagedMemoryFileGo approximates TS isAutoManagedMemoryFile for transcript collapse (no memdir config).
func isAutoManagedMemoryFileGo(filePath string) bool {
	cp := toComparablePath(filePath)
	if cp == "" {
		return false
	}
	if strings.Contains(cp, "/.claude/memory/") || strings.Contains(cp, "/.claude\\memory\\") {
		return true
	}
	if strings.Contains(cp, "session-memory") && strings.HasSuffix(cp, ".md") {
		return true
	}
	if strings.Contains(cp, "/agent-memory/") || strings.Contains(cp, "/agent-memory-local/") {
		return true
	}
	if strings.Contains(cp, "/projects/") && strings.HasSuffix(cp, ".jsonl") {
		return true
	}
	return false
}

func isMemoryDirectoryGo(dirPath string) bool {
	cp := toComparablePath(dirPath)
	if cp == "" {
		return false
	}
	return strings.Contains(cp, "/.claude/memory") ||
		strings.Contains(cp, "session-memory") ||
		strings.Contains(cp, "/agent-memory/")
}

func isAutoManagedMemoryPatternGo(glob string) bool {
	g := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(glob), `\`, `/`))
	return strings.Contains(g, "session-memory") ||
		(strings.Contains(g, "projects") && strings.Contains(g, ".jsonl"))
}

func isShellCommandTargetingMemoryGo(command string) bool {
	c := strings.ToLower(command)
	return strings.Contains(c, ".claude/memory") ||
		strings.Contains(c, "session-memory") ||
		strings.Contains(c, "agent-memory")
}

// isMemorySearchGo mirrors TS isMemorySearch (path / glob / bash command heuristics).
func isMemorySearchGo(toolInput map[string]any) bool {
	if toolInput == nil {
		return false
	}
	if p, ok := toolInput["path"].(string); ok && strings.TrimSpace(p) != "" {
		if isAutoManagedMemoryFileGo(p) || isMemoryDirectoryGo(p) {
			return true
		}
	}
	if g, ok := toolInput["glob"].(string); ok && strings.TrimSpace(g) != "" {
		if isAutoManagedMemoryPatternGo(g) {
			return true
		}
	}
	if cmd, ok := toolInput["command"].(string); ok && strings.TrimSpace(cmd) != "" {
		if isShellCommandTargetingMemoryGo(cmd) {
			return true
		}
	}
	return false
}

func isMemoryWriteOrEditGo(toolName string, toolInput map[string]any) bool {
	if toolName != "Write" && toolName != "Edit" {
		return false
	}
	fp := getFilePathFromToolInput(toolInput)
	return fp != "" && isAutoManagedMemoryFileGo(fp)
}
