package screen

import (
	"fmt"
	"strings"
)

// TranscriptFooterLines returns the transcript footer line(s) matching TS REPL.tsx.
func TranscriptFooterLines(narrow, showAll, dumpMode bool) []string {
	toggle := "ctrl+o"
	showAllHint := "off"
	if showAll {
		showAllHint = "on"
	}
	if dumpMode {
		line := fmt.Sprintf("Transcript · %s toggle · [ dump · v $EDITOR · Esc/q/ctrl+c", toggle)
		if narrow {
			line = fmt.Sprintf("Transcript · %s · [ · v · Esc", toggle)
		}
		return []string{line}
	}
	line := fmt.Sprintf("Transcript · %s toggle · ctrl+l redraw · ctrl+e %s · jk gG ctrl+udbf · / search · [ v · Esc/q/ctrl+c", toggle, showAllHint)
	if narrow {
		line = fmt.Sprintf("Transcript · %s · ctrl+l · ctrl+e %s · jk · / · [ v · Esc", toggle, showAllHint)
	}
	return []string{line}
}

// JoinFooterLines joins lines with newlines, truncating each to cols.
func JoinFooterLines(lines []string, cols int) string {
	var b strings.Builder
	for i, ln := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if cols > 0 && len(ln) > cols {
			ln = ln[:max(0, cols-1)] + "…"
		}
		b.WriteString(ln)
	}
	return b.String()
}
