package slashresolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/debugpath"
	"goc/commands"
	"goc/types"
)

const debugTailLines = 20
const debugTailBytes = 64 * 1024

func debugLogPath(sessionID string) string {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		sid = "gou-demo"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if home == "" {
		return filepath.Join(os.TempDir(), "claude", "debug", sid+".txt")
	}
	return filepath.Join(home, ".claude", "debug", sid+".txt")
}

func resolveDebugBundled(args, sessionID string) types.SlashResolveResult {
	// TS side-effect parity: enable debug logging so future API calls are captured.
	enableDebugLogging()

	logPath := debugLogPath(sessionID)
	tail, err := tailTextFile(logPath, debugTailBytes, debugTailLines)
	logInfo := ""
	if err != nil {
		if os.IsNotExist(err) {
			logInfo = "No debug log exists yet — enable debug logging in your host or run with --debug."
		} else {
			logInfo = fmt.Sprintf("Failed to read last %d lines of debug log: %v", debugTailLines, err)
		}
	} else {
		logInfo = tail
	}

	cfgUser := filepath.Join(commands.ClaudeConfigHome(), "settings.json")
	cfgProj := ".claude/settings.json"
	cfgLocal := ".claude/settings.local.json"

	prompt := fmt.Sprintf(`# Debug Skill

Help the user debug an issue they're encountering in this current Claude Code session.

## Session Debug Log

The debug log for the current session is at: `+"`%s`"+`

%s

For additional context, grep for [ERROR] and [WARN] lines across the full file.

## Issue Description

%s

## Settings

Remember that settings are in:
* user - %s
* project - %s
* local - %s

## Instructions

1. Review the user's issue description
2. The last %d lines show the debug file format. Look for [ERROR] and [WARN] entries, stack traces, and failure patterns across the file
3. Consider launching a Claude Code guide subagent to understand relevant features
4. Explain what you found in plain language
5. Suggest concrete fixes or next steps
`, logPath, logInfo, issueDesc(args), cfgUser, cfgProj, cfgLocal, debugTailLines)

	return types.SlashResolveResult{UserText: prompt, Source: types.SlashResolveBundledEmbed}
}

func issueDesc(args string) string {
	if strings.TrimSpace(args) == "" {
		return "The user did not describe a specific issue. Read the debug log and summarize any errors, warnings, or notable issues."
	}
	return strings.TrimSpace(args)
}

func tailTextFile(path string, maxBytes, maxLines int) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	readSize := int(fi.Size())
	if readSize > maxBytes {
		readSize = maxBytes
	}
	if readSize == 0 {
		return "Log size: 0 B\n\n### Last lines\n\n```\n\n```", nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	start := fi.Size() - int64(readSize)
	if start < 0 {
		start = 0
	}
	buf := make([]byte, readSize)
	_, err = f.ReadAt(buf, start)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(buf), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	body := strings.Join(lines, "\n")
	return fmt.Sprintf("Log size: %d bytes\n\n### Last %d lines\n\n```\n%s\n```", fi.Size(), maxLines, body), nil
}

// enableDebugLogging mirrors src/utils/debug.ts enableDebugLogging: enables debug
// logging mid-session so subsequent API activity is captured. Returns true if
// logging was already active.
func enableDebugLogging() bool {
	wasActive := apilog.DebugModeEnabled()
	if !wasActive {
		os.Setenv("CLAUDE_CODE_DEBUG", "1")
		// Ensure the log file exists for the current session.
		if p := debugpath.ResolveLogPath(); p != "" {
			dir := filepath.Dir(p)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return false
			}
			f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err == nil {
				f.Close()
				debugpath.MaybeUpdateLatestSymlink(p)
			}
		}
	}
	return wasActive
}
