package autodream

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"goc/claudebase"
)

const (
	lockFileName     = ".consolidate-lock"
	holderStaleMs    = 60 * 60 * 1000 // 1 hour PID reuse guard
	sessionScanLimit = 1000
)

// lockFilePath returns the path to the consolidation lock file within memoryDir.
func lockFilePath(memoryDir string) string {
	return filepath.Join(memoryDir, lockFileName)
}

// ReadLastConsolidatedAt mirrors readLastConsolidatedAt() in
// src/services/autoDream/consolidationLock.ts.
// Returns the mtime of the lock file (epoch ms), or 0 if absent.
func ReadLastConsolidatedAt(memoryDir string) (int64, error) {
	s, err := os.Stat(lockFilePath(memoryDir))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return s.ModTime().UnixMilli(), nil
}

// TryAcquireConsolidationLock mirrors tryAcquireConsolidationLock() in
// src/services/autoDream/consolidationLock.ts.
// Writes PID to lock file and verifies ownership.
// Returns the pre-acquire mtime (for rollback), or -1 if blocked.
func TryAcquireConsolidationLock(memoryDir string) (priorMtime int64, err error) {
	path := lockFilePath(memoryDir)

	// Check existing lock.
	var existingMtime int64
	var holderPid int
	if data, err := os.ReadFile(path); err == nil {
		if s, err := os.Stat(path); err == nil {
			existingMtime = s.ModTime().UnixMilli()
		}
		pidStr := strings.TrimSpace(string(data))
		if p, err := strconv.Atoi(pidStr); err == nil {
			holderPid = p
		}
	}

	// If lock is recent and held by a live process, bail.
	if existingMtime > 0 && time.Since(time.UnixMilli(existingMtime)).Milliseconds() < holderStaleMs {
		if holderPid > 0 && isProcessRunning(holderPid) {
			return -1, fmt.Errorf("lock held by live PID %d", holderPid)
		}
	}

	// Ensure memory dir exists.
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return -1, err
	}

	// Write our PID.
	pid := os.Getpid()
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return -1, err
	}

	// Verify we won the write race.
	verify, err := os.ReadFile(path)
	if err != nil {
		return -1, fmt.Errorf("lock verify read failed: %w", err)
	}
	verifyPid, err := strconv.Atoi(strings.TrimSpace(string(verify)))
	if err != nil || verifyPid != pid {
		return -1, fmt.Errorf("lock race lost: wrote %d, read %d", pid, verifyPid)
	}

	if existingMtime > 0 {
		return existingMtime, nil
	}
	return 0, nil
}

// RollbackConsolidationLock mirrors rollbackConsolidationLock() in
// src/services/autoDream/consolidationLock.ts.
// Rewinds mtime to pre-acquire state. priorMtime 0 means unlink (restore absent).
func RollbackConsolidationLock(memoryDir string, priorMtime int64) {
	path := lockFilePath(memoryDir)
	if priorMtime == 0 {
		_ = os.Remove(path)
		return
	}
	// Write empty body and rewind mtime.
	_ = os.WriteFile(path, []byte{}, 0644)
	t := time.UnixMilli(priorMtime)
	_ = os.Chtimes(path, t, t)
}

// ListSessionsTouchedSince mirrors listSessionsTouchedSince() in
// src/services/autoDream/consolidationLock.ts.
// Lists session IDs (.jsonl files) whose mtime > sinceMs in the project
// transcript directory derived from originalCwd.
func ListSessionsTouchedSince(sinceMs int64, originalCwd, configHome string) ([]string, error) {
	projectDir := ProjectDirForOriginalCwd(originalCwd, configHome)
	return listSessionIDsSince(projectDir, sinceMs)
}

// listSessionIDsSince scans a directory for .jsonl files and returns session
// IDs (filename without .jsonl extension) whose mtime > sinceMs.
func listSessionIDsSince(dir string, sinceMs int64) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")
		// Validate UUID-like format (basic check matching TS validateUuid).
		if !looksLikeUUID(sessionID) {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixMilli() > sinceMs {
			ids = append(ids, sessionID)
		}
		if len(ids) > sessionScanLimit {
			break
		}
	}
	return ids, nil
}

// RecordConsolidation mirrors recordConsolidation() in
// src/services/autoDream/consolidationLock.ts.
// Stamps the lock file with current PID (manual /dream).
func RecordConsolidation(memoryDir string) error {
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(lockFilePath(memoryDir), []byte(strconv.Itoa(os.Getpid())), 0644)
}

// isProcessRunning checks if a process with the given PID exists on this
// system. Mirrors isProcessRunning in TS utils/genericProcessUtils.ts.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Signal 0 tests existence.
	return p.Signal(os.Signal(nil)) == nil
}

// ProjectDirForOriginalCwd mirrors ProjectDirForOriginalCwd in sessiontranscript/paths.go.
// Included here to avoid import cycle (sessiontranscript imports claudemd).
func ProjectDirForOriginalCwd(projectPath, configHome string) string {
	return filepath.Join(ProjectsDir(configHome), sanitizePath(projectPath))
}

// ProjectsDir is ~/.harness/projects.
func ProjectsDir(configHome string) string {
	return filepath.Join(configHome, "projects")
}

// sanitizePath delegates to claudebase.SanitizePath.
func sanitizePath(p string) string {
	return claudebase.SanitizePath(p)
}

// looksLikeUUID checks whether s looks like a UUID (hex with hyphens).
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if c == '-' {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
		_ = i
	}
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}
