package messagerow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Limits mirror TS FileWriteTool / FileEditTool transcript diff caps (avoid flooding the terminal).
const (
	maxStructuredPatchTotalLines = 120
	maxCreateFilePreviewLines    = 48
)

type structuredPatchJSONHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

type writeEditToolResultWire struct {
	Type            string                    `json:"type"`
	FilePath        string                    `json:"filePath"`
	Content         string                    `json:"content"`
	StructuredPatch []structuredPatchJSONHunk `json:"structuredPatch"`
}

// FormatWriteEditToolResultBodyIfApplicable returns non-empty text when tool_result JSON is a FileWrite
// (create/update) or FileEdit payload with a structured patch, or a Write "create" with file body.
// When ok is false, callers should fall back to generic JSON preview (toolResultContentPreview).
func FormatWriteEditToolResultBodyIfApplicable(raw json.RawMessage) (text string, ok bool) {
	n := NormalizeToolResultContentJSON(raw)
	if len(n) == 0 || n[0] != '{' {
		return "", false
	}
	var w writeEditToolResultWire
	if err := json.Unmarshal(n, &w); err != nil {
		return "", false
	}
	if len(w.StructuredPatch) > 0 {
		return formatStructuredPatchDisplay(w.FilePath, w.StructuredPatch), true
	}
	if strings.TrimSpace(w.Type) == "create" && strings.TrimSpace(w.FilePath) != "" {
		s := formatCreatedFileAsPlusLines(w.FilePath, w.Content)
		if s != "" {
			return s, true
		}
	}
	return "", false
}

// IndentedWriteEditDiffLinesFromToolResultJSON returns 2-space-indented lines for transcript/TUI when
// content is JSON from FileWrite or FileEdit with structuredPatch (or Write create body).
func IndentedWriteEditDiffLinesFromToolResultJSON(contentJSON string) ([]string, bool) {
	txt, ok := FormatWriteEditToolResultBodyIfApplicable(json.RawMessage(contentJSON))
	if !ok || strings.TrimSpace(txt) == "" {
		return nil, false
	}
	lines := strings.Split(txt, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		out = append(out, "  "+ln)
	}
	return out, true
}

func formatStructuredPatchDisplay(filePath string, hunks []structuredPatchJSONHunk) string {
	var b strings.Builder
	fp := strings.TrimSpace(filePath)
	if fp != "" {
		b.WriteString("--- ")
		b.WriteString(fp)
		b.WriteString("\n+++ ")
		b.WriteString(fp)
		b.WriteByte('\n')
	}
	emitted := 0
	truncated := false
hunkLoop:
	for hi := range hunks {
		h := hunks[hi]
		if emitted >= maxStructuredPatchTotalLines {
			truncated = true
			break
		}
		_, _ = fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
		emitted++
		collapsed := CollapseUnifiedDiffContextLines(h.Lines, DefaultUnifiedDiffContextLines)
		for _, ln := range collapsed {
			if emitted >= maxStructuredPatchTotalLines {
				truncated = true
				break hunkLoop
			}
			b.WriteString(ln)
			b.WriteByte('\n')
			emitted++
		}
	}
	if truncated {
		b.WriteString("… (diff truncated)\n")
	}
	return strings.TrimSpace(b.String())
}

func formatCreatedFileAsPlusLines(filePath, content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	var b strings.Builder
	b.WriteString("+++ ")
	b.WriteString(strings.TrimSpace(filePath))
	b.WriteString(" (new file)\n")
	n := 0
	for _, ln := range lines {
		if n >= maxCreateFilePreviewLines {
			b.WriteString("… (truncated)\n")
			break
		}
		b.WriteByte('+')
		b.WriteString(ln)
		b.WriteByte('\n')
		n++
	}
	return strings.TrimSpace(b.String())
}
