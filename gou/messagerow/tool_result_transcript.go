// Transcript-aligned resolved-tool ⎿ lines (TS GrepTool SearchResultSummary + FileReadTool renderToolResultMessage).
package messagerow

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// CollectToolResultContentByToolUseID maps tool_use_id → tool_result content JSON (last wins if duplicated).
func CollectToolResultContentByToolUseID(msgs []types.Message) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage)
	for _, msg := range msgs {
		if msg.Type != types.MessageTypeUser || len(msg.Content) == 0 {
			continue
		}
		var blocks []types.MessageContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type != "tool_result" {
				continue
			}
			id := strings.TrimSpace(b.ToolUseID)
			if id == "" || len(b.Content) == 0 {
				continue
			}
			out[id] = b.Content
		}
	}
	return out
}

// TranscriptResolvedHintExtra returns the ⎿ summary line and optional indented preview line (TS verbose SearchResultSummary).
// toolFacing is SegToolUse.ToolFacing ("Read", "Search", "Bash", …).
func TranscriptResolvedHintExtra(toolFacing string, resultJSON json.RawMessage) (hint string, extra string) {
	if len(resultJSON) == 0 {
		return "", ""
	}
	switch toolFacing {
	case "Read":
		return readResultTranscriptLines(resultJSON)
	case "Search":
		return searchResultTranscriptLines(resultJSON)
	default:
		return "", ""
	}
}

func readResultTranscriptLines(raw json.RawMessage) (hint, extra string) {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "", ""
	}
	switch probe.Type {
	case "text":
		var o struct {
			File struct {
				NumLines int `json:"numLines"`
			} `json:"file"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		n := o.File.NumLines
		if n < 1 {
			return "", ""
		}
		if n == 1 {
			return "Read 1 line", ""
		}
		return fmt.Sprintf("Read %d lines", n), ""
	case "notebook":
		var o struct {
			File struct {
				Cells []json.RawMessage `json:"cells"`
			} `json:"file"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		n := len(o.File.Cells)
		if n < 1 {
			return "", ""
		}
		if n == 1 {
			return "Read 1 cell", ""
		}
		return fmt.Sprintf("Read %d cells", n), ""
	case "image":
		return "Read image", ""
	case "pdf":
		return "Read PDF", ""
	case "parts":
		var o struct {
			File struct {
				Count int `json:"count"`
			} `json:"file"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		c := o.File.Count
		if c == 1 {
			return "Read 1 page", ""
		}
		if c > 1 {
			return fmt.Sprintf("Read %d pages", c), ""
		}
		return "", ""
	case "file_unchanged":
		return "Unchanged since last read", ""
	default:
		return "", ""
	}
}

// grepStructuredOutput mirrors ccb-engine/localtools grep JSON shape.
type grepStructuredOutput struct {
	Mode       string   `json:"mode,omitempty"`
	NumFiles   int      `json:"numFiles"`
	Filenames  []string `json:"filenames"`
	Content    string   `json:"content,omitempty"`
	NumLines   *int     `json:"numLines,omitempty"`
	NumMatches *int     `json:"numMatches,omitempty"`
}

func searchResultTranscriptLines(raw json.RawMessage) (hint, extra string) {
	// Glob (and some paths) return a plain string listing files or "No files found".
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && strings.TrimSpace(asString) != "" {
		return globPlainStringTranscriptLines(asString)
	}
	var o grepStructuredOutput
	if err := json.Unmarshal(raw, &o); err != nil {
		return "", ""
	}
	mode := strings.TrimSpace(o.Mode)
	if mode == "" {
		mode = "files_with_matches"
	}
	switch mode {
	case "content":
		n := 0
		if o.NumLines != nil {
			n = *o.NumLines
		}
		if n == 0 && strings.TrimSpace(o.Content) != "" {
			n = strings.Count(o.Content, "\n") + 1
		}
		if n == 0 {
			return "", ""
		}
		label := "lines"
		if n == 1 {
			label = "line"
		}
		hint = fmt.Sprintf("Found %d %s", n, label)
		extra = firstNonEmptyLine(o.Content)
		return hint, extra
	case "count":
		n := 0
		if o.NumMatches != nil {
			n = *o.NumMatches
		}
		if n == 0 {
			return "", ""
		}
		label := "matches"
		if n == 1 {
			label = "match"
		}
		hint = fmt.Sprintf("Found %d %s", n, label)
		if o.NumFiles > 0 {
			fl := "files"
			if o.NumFiles == 1 {
				fl = "file"
			}
			hint += fmt.Sprintf(" across %d %s", o.NumFiles, fl)
		}
		return hint, ""
	default: // files_with_matches
		n := o.NumFiles
		if n == 0 {
			return "", ""
		}
		label := "files"
		if n == 1 {
			label = "file"
		}
		hint = fmt.Sprintf("Found %d %s", n, label)
		if len(o.Filenames) > 0 {
			extra = o.Filenames[0]
		}
		return hint, extra
	}
}

func globPlainStringTranscriptLines(s string) (hint, extra string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if strings.Contains(strings.ToLower(s), "no files found") {
		return "No files found", ""
	}
	lines := strings.Split(s, "\n")
	nonEmpty := lines[:0]
	for _, ln := range lines {
		if t := strings.TrimSpace(ln); t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}
	n := len(nonEmpty)
	if n == 0 {
		return "", ""
	}
	label := "files"
	if n == 1 {
		label = "file"
	}
	hint = fmt.Sprintf("Found %d %s", n, label)
	extra = nonEmpty[0]
	return hint, extra
}

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}
