package extractmemories

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File logging (optional) for the extract-memories pipeline.
//
// Set one of:
//   - GOC_EXTRACT_MEMORIES_LOG_FILE
//   - CLAUDE_CODE_EXTRACT_MEMORIES_LOG_FILE
//
// to a path; lines are appended with a UTC timestamp. Parent directories
// are created as needed. Concurrent writers are serialized.
//
// Optional: GOC_EXTRACT_MEMORIES_LOG or CLAUDE_CODE_EXTRACT_MEMORIES_LOG = 0|false|off|no
// forces logging off even if a log file path is set in the environment (e.g. inherited profile).

var fileLogMu sync.Mutex

func extractMemoriesLogFilePath() string {
	for _, k := range []string{
		"GOC_EXTRACT_MEMORIES_LOG_FILE",
		"CLAUDE_CODE_EXTRACT_MEMORIES_LOG_FILE",
	} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			return p
		}
	}
	return ""
}

func extractMemoriesFileLoggingExplicitlyOff() bool {
	for _, k := range []string{
		"GOC_EXTRACT_MEMORIES_LOG",
		"CLAUDE_CODE_EXTRACT_MEMORIES_LOG",
	} {
		v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
		if v == "0" || v == "false" || v == "off" || v == "no" {
			return true
		}
	}
	return false
}

func fileExtractMemoriesLogf(format string, args ...any) {
	if extractMemoriesFileLoggingExplicitlyOff() {
		return
	}
	path := extractMemoriesLogFilePath()
	if path == "" {
		return
	}
	line := fmt.Sprintf("%s [extract-memories] %s\n", time.Now().UTC().Format(time.RFC3339Nano), fmt.Sprintf(format, args...))
	fileLogMu.Lock()
	defer fileLogMu.Unlock()
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}
