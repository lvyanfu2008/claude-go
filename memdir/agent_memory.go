package memdir

import (
	"os"
	"path/filepath"
	"strings"

	"goc/claudebase"
)

// AgentMemoryScope mirrors TS agentMemory.ts AgentMemoryScope.
type AgentMemoryScope string

const (
	AgentMemoryUser    AgentMemoryScope = "user"
	AgentMemoryProject AgentMemoryScope = "project"
	AgentMemoryLocal   AgentMemoryScope = "local"
)

// sanitizeAgentTypeForPath mirrors TS sanitizeAgentTypeForPath:
// replaces colons (invalid on Windows, used in plugin-namespaced agent types)
// with dashes.
func sanitizeAgentTypeForPath(agentType string) string {
	return strings.ReplaceAll(agentType, ":", "-")
}

// getLocalAgentMemoryDir mirrors TS getLocalAgentMemoryDir.
// Returns the local agent memory directory, which is project-specific and not checked into VCS.
func getLocalAgentMemoryDir(dirName string) string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); v != "" {
		// Persists to the mount with project namespacing.
		projectRoot := findCanonicalGitRootOrCwd()
		return filepath.Join(v, "projects", claudebase.SanitizePath(projectRoot), "agent-memory-local", dirName) + string(filepath.Separator)
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".harness", "agent-memory-local", dirName) + string(filepath.Separator)
}

// findCanonicalGitRootOrCwd returns the canonical git root if available, otherwise the current directory.
func findCanonicalGitRootOrCwd() string {
	cwd, _ := os.Getwd()
	if cr := claudebase.ResolveCanonicalGitRoot(cwd); cr != "" {
		return cr
	}
	return cwd
}

// GetAgentMemoryDir mirrors TS getAgentMemoryDir.
func GetAgentMemoryDir(agentType string, scope AgentMemoryScope) string {
	dirName := sanitizeAgentTypeForPath(agentType)
	switch scope {
	case AgentMemoryProject:
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".harness", "agent-memory", dirName) + string(filepath.Separator)
	case AgentMemoryLocal:
		return getLocalAgentMemoryDir(dirName)
	case AgentMemoryUser:
		return filepath.Join(MemoryBaseDir(), "agent-memory", dirName) + string(filepath.Separator)
	}
	return ""
}

// IsAgentMemoryPath mirrors TS isAgentMemoryPath.
func IsAgentMemoryPath(absolutePath string) bool {
	normalizedPath := filepath.Clean(absolutePath)
	memoryBase := MemoryBaseDir()

	// User scope: check memory base
	if strings.HasPrefix(normalizedPath, filepath.Join(memoryBase, "agent-memory")+string(filepath.Separator)) {
		return true
	}

	// Project scope: always cwd-based
	cwd, _ := os.Getwd()
	if strings.HasPrefix(normalizedPath, filepath.Join(cwd, ".harness", "agent-memory")+string(filepath.Separator)) {
		return true
	}

	// Local scope
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REMOTE_MEMORY_DIR")); v != "" {
		sep := string(filepath.Separator)
		if strings.Contains(normalizedPath, sep+"agent-memory-local"+sep) &&
			strings.HasPrefix(normalizedPath, filepath.Join(v, "projects")+sep) {
			return true
		}
	} else {
		sep := string(filepath.Separator)
		if strings.HasPrefix(normalizedPath, filepath.Join(cwd, ".harness", "agent-memory-local")+sep) {
			return true
		}
	}

	return false
}

// GetAgentMemoryEntrypoint mirrors TS getAgentMemoryEntrypoint.
func GetAgentMemoryEntrypoint(agentType string, scope AgentMemoryScope) string {
	return filepath.Join(GetAgentMemoryDir(agentType, scope), "MEMORY.md")
}

// GetMemoryScopeDisplay mirrors TS getMemoryScopeDisplay.
func GetMemoryScopeDisplay(memory AgentMemoryScope) string {
	switch memory {
	case AgentMemoryUser:
		return "User (" + filepath.Join(MemoryBaseDir(), "agent-memory") + "/)"
	case AgentMemoryProject:
		return "Project (.harness/agent-memory/)"
	case AgentMemoryLocal:
		return "Local (" + getLocalAgentMemoryDir("...") + ")"
	default:
		return "None"
	}
}

// LoadAgentMemoryPrompt mirrors TS loadAgentMemoryPrompt.
func LoadAgentMemoryPrompt(agentType string, scope AgentMemoryScope) string {
	var scopeNote string
	switch scope {
	case AgentMemoryUser:
		scopeNote = "- Since this memory is user-scope, keep learnings general since they apply across all projects"
	case AgentMemoryProject:
		scopeNote = "- Since this memory is project-scope and shared with your team via version control, tailor your memories to this project"
	case AgentMemoryLocal:
		scopeNote = "- Since this memory is local-scope (not checked into version control), tailor your memories to this project and machine"
	}

	memoryDir := GetAgentMemoryDir(agentType, scope)

	// Fire-and-forget: ensure the directory exists so the model can write without checking.
	_ = EnsureMemoryDirExists(memoryDir)

	coworkExtraGuidelines := strings.TrimSpace(os.Getenv("CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES"))
	var extraGuidelines []string
	if coworkExtraGuidelines != "" {
		extraGuidelines = append(extraGuidelines, scopeNote, coworkExtraGuidelines)
	} else {
		extraGuidelines = []string{scopeNote}
	}

	return BuildAgentMemoryPrompt(AgentMemoryPromptParams{
		DisplayName:     "Persistent Agent Memory",
		MemoryDir:       memoryDir,
		ExtraGuidelines: extraGuidelines,
	})
}
