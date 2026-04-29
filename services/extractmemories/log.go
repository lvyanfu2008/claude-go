package extractmemories

import (
	"fmt"
	"os"
	"strings"

	"goc/ccb-engine/diaglog"
)

// fileExtractMemoriesLogf writes an extract-memories diagnostic line to the
// standard Claude debug log (same destination as diaglog.LineOrStderr).
// Gated by the same env vars as before: set GOC_EXTRACT_MEMORIES_LOG=0 or
// CLAUDE_CODE_EXTRACT_MEMORIES_LOG=0 to disable.

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
	diaglog.LineOrStderr("[extract-memories] %s", fmt.Sprintf(format, args...))
}
