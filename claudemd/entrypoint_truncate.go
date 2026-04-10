package claudemd

import (
	"fmt"
	"strings"
)

// Constants from src/memdir/memdir.ts for MEMORY.md / team entrypoint truncation.
const (
	entrypointName       = "MEMORY.md"
	maxEntrypointLines   = 200
	maxEntrypointBytes   = 25000
)

// TruncateEntrypointContent mirrors memdir.ts truncateEntrypointContent.
func TruncateEntrypointContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	contentLines := strings.Split(trimmed, "\n")
	lineCount := len(contentLines)
	byteCount := len(trimmed)

	wasLineTruncated := lineCount > maxEntrypointLines
	wasByteTruncated := byteCount > maxEntrypointBytes

	if !wasLineTruncated && !wasByteTruncated {
		return trimmed
	}

	truncated := trimmed
	if wasLineTruncated {
		truncated = strings.Join(contentLines[:maxEntrypointLines], "\n")
	}
	if len(truncated) > maxEntrypointBytes {
		cutAt := strings.LastIndex(truncated[:maxEntrypointBytes], "\n")
		if cutAt <= 0 {
			truncated = truncated[:maxEntrypointBytes]
		} else {
			truncated = truncated[:cutAt]
		}
	}

	reason := fmt.Sprintf("%d lines and %d bytes", lineCount, byteCount)
	if wasByteTruncated && !wasLineTruncated {
		reason = fmt.Sprintf("%d bytes (limit: %d) — index entries are too long", byteCount, maxEntrypointBytes)
	} else if wasLineTruncated && !wasByteTruncated {
		reason = fmt.Sprintf("%d lines (limit: %d)", lineCount, maxEntrypointLines)
	}

	return truncated + fmt.Sprintf(`

> WARNING: %s is %s. Only part of it was loaded. Keep index entries to one line under ~200 chars; move detail into topic files.`,
		entrypointName, reason)
}
