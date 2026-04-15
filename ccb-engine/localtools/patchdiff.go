package localtools

import (
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Mirrors src/utils/diff.ts CONTEXT_LINES and DIFF_TIMEOUT_MS.
const patchDiffTimeout = 5 * time.Second

const ampersandToken = "<<:AMPERSAND_TOKEN:>>"
const dollarToken = "<<:DOLLAR_TOKEN:>>"

// ConvertLeadingTabsToSpaces mirrors src/utils/file.ts convertLeadingTabsToSpaces.
func ConvertLeadingTabsToSpaces(content string) string {
	if !strings.Contains(content, "\t") {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		n := 0
		for n < len(line) && line[n] == '\t' {
			n++
		}
		if n > 0 {
			lines[i] = strings.Repeat("  ", n) + line[n:]
		}
	}
	return strings.Join(lines, "\n")
}

func escapeForDiff(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "&", ampersandToken), "$", dollarToken)
}

func unescapeFromDiff(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, ampersandToken, "&"), dollarToken, "$")
}

// StructuredPatchHunk matches src/tools/FileEditTool/types.ts hunkSchema (JSON field names).
type StructuredPatchHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

func splitDiffTextLines(text string) []string {
	if text == "" {
		return nil
	}
	// Line-mode diffs from DiffCharsToLines use trailing \n per line.
	if !strings.Contains(text, "\n") {
		return []string{text}
	}
	s := strings.TrimSuffix(text, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// GetPatchFromContents mirrors src/utils/diff.ts getPatchFromContents.
// Produces a single hunk covering the full diff (valid unified-style counts); sufficient for tool JSON parity.
func GetPatchFromContents(filePath, oldContent, newContent string) []StructuredPatchHunk {
	_ = filePath
	oldE := escapeForDiff(ConvertLeadingTabsToSpaces(oldContent))
	newE := escapeForDiff(ConvertLeadingTabsToSpaces(newContent))
	if oldE == newE {
		return nil
	}
	dmp := diffmatchpatch.New()
	dmp.DiffTimeout = patchDiffTimeout
	ch1, ch2, lineArr := dmp.DiffLinesToChars(oldE, newE)
	diffs := dmp.DiffMain(ch1, ch2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArr)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var hlines []string
	oldLines, newLines := 0, 0
	for _, d := range diffs {
		for _, ln := range splitDiffTextLines(d.Text) {
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				hlines = append(hlines, " "+ln)
				oldLines++
				newLines++
			case diffmatchpatch.DiffDelete:
				hlines = append(hlines, "-"+ln)
				oldLines++
			case diffmatchpatch.DiffInsert:
				hlines = append(hlines, "+"+ln)
				newLines++
			}
		}
	}
	if len(hlines) == 0 {
		return nil
	}
	for i := range hlines {
		hlines[i] = unescapeFromDiff(hlines[i])
	}
	return []StructuredPatchHunk{{
		OldStart: 1,
		OldLines: oldLines,
		NewStart: 1,
		NewLines: newLines,
		Lines:    hlines,
	}}
}
