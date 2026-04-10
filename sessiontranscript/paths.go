// Package sessiontranscript mirrors TS session transcript persistence (recordTranscript / insertMessageChain subset).
package sessiontranscript

import (
	"os"
	"path/filepath"
	"strings"

	"goc/claudemd"
)

// ConfigHomeDir matches getClaudeConfigHomeDir: CLAUDE_CONFIG_DIR or ~/.claude (NFC via filepath.Clean).
func ConfigHomeDir() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return filepath.Clean(d)
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(".claude")
	}
	return filepath.Join(h, ".claude")
}

// ProjectsDir is ~/.claude/projects (or under ConfigHomeDir).
func ProjectsDir(configHome string) string {
	return filepath.Join(configHome, "projects")
}

// ProjectDirForOriginalCwd matches getProjectDir(projectPath) in sessionStoragePortable.ts.
func ProjectDirForOriginalCwd(projectPath, configHome string) string {
	return filepath.Join(ProjectsDir(configHome), claudemd.SanitizePath(projectPath))
}

// TranscriptPath matches getTranscriptPath: join(sessionProjectDir ?? projectDir(originalCwd), sessionId+".jsonl").
func TranscriptPath(sessionID, originalCwd, sessionProjectDirOverride, configHome string) string {
	base := strings.TrimSpace(sessionProjectDirOverride)
	if base == "" {
		base = ProjectDirForOriginalCwd(originalCwd, configHome)
	}
	return filepath.Join(base, sessionID+".jsonl")
}

// AgentTranscriptPath matches getAgentTranscriptPath when subdir is empty.
func AgentTranscriptPath(sessionID, originalCwd, sessionProjectDirOverride, configHome, agentID, subdir string) string {
	projectDir := strings.TrimSpace(sessionProjectDirOverride)
	if projectDir == "" {
		projectDir = ProjectDirForOriginalCwd(originalCwd, configHome)
	}
	base := filepath.Join(projectDir, sessionID, "subagents")
	if strings.TrimSpace(subdir) != "" {
		base = filepath.Join(base, strings.TrimSpace(subdir))
	}
	return filepath.Join(base, "agent-"+agentID+".jsonl")
}
