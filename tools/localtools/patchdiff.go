package localtools

import (
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Mirrors src/utils/diff.ts CONTEXT_LINES and DIFF_TIMEOUT_MS.
const patchDiffTimeout = 5 * time.Second
const patchContextLines = 3

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
		// Inserted/removed blank line(s) — a lone "\n" means one empty content line.
		// Preserve it so blank-line edits produce a "+"/"-" row instead of vanishing.
		if text == "\n" {
			return []string{""}
		}
		return nil
	}
	return strings.Split(s, "\n")
}

// GetPatchFromContents mirrors src/utils/diff.ts getPatchFromContents.
// Uses diffmatchpatch to compute the diff, then trims context to patchContextLines (3) around changes,
// splitting into multiple hunks when changes are far apart. Matches TS structuredPatch behavior.
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

	// Build flat line list with markers
	type dline struct {
		text  string
		kind  byte // ' ', '-', '+'
		oldNo int  // 1-based, 0 if not in old
		newNo int  // 1-based, 0 if not in new
	}
	var lines []dline
	oi, ni := 1, 1
	for _, d := range diffs {
		for _, ln := range splitDiffTextLines(d.Text) {
			dl := dline{text: unescapeFromDiff(ln)}
			switch d.Type {
			case diffmatchpatch.DiffEqual:
				dl.kind = ' '
				dl.oldNo = oi
				dl.newNo = ni
				oi++
				ni++
			case diffmatchpatch.DiffDelete:
				dl.kind = '-'
				dl.oldNo = oi
				oi++
			case diffmatchpatch.DiffInsert:
				dl.kind = '+'
				dl.newNo = ni
				ni++
			}
			lines = append(lines, dl)
		}
	}
	if len(lines) == 0 {
		return nil
	}

	// Find changed line indices
	changed := make([]bool, len(lines))
	for i, l := range lines {
		if l.kind != ' ' {
			changed[i] = true
		}
	}

	// Build context windows: expand each changed line by patchContextLines
	inWindow := make([]bool, len(lines))
	for i := range lines {
		if !changed[i] {
			continue
		}
		start := i - patchContextLines
		if start < 0 {
			start = 0
		}
		end := i + patchContextLines
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			inWindow[j] = true
		}
	}

	// Split into hunks at gaps in the window
	var hunks []StructuredPatchHunk
	start := 0
	for start < len(lines) {
		if !inWindow[start] {
			start++
			continue
		}
		end := start
		for end+1 < len(lines) && inWindow[end+1] {
			end++
		}
		// Extract hunk lines
		hunkLines := lines[start : end+1]
		var hlines []string
		oldCount, newCount := 0, 0
		oldStart := 0
		newStart := 0
		for _, l := range hunkLines {
			prefix := string(l.kind)
			hlines = append(hlines, prefix+l.text)
			if l.kind == ' ' || l.kind == '-' {
				if oldStart == 0 {
					oldStart = l.oldNo
				}
				oldCount++
			}
			if l.kind == ' ' || l.kind == '+' {
				if newStart == 0 {
					newStart = l.newNo
				}
				newCount++
			}
		}
		hunks = append(hunks, StructuredPatchHunk{
			OldStart: oldStart,
			OldLines: oldCount,
			NewStart: newStart,
			NewLines: newCount,
			Lines:    hlines,
		})
		start = end + 1
	}
	if len(hunks) == 0 {
		return nil
	}
	return hunks
}
