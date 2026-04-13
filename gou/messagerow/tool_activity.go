// Tool activity lines mirror TS Tool.getActivityDescription + getToolUseSummary
// (e.g. FileReadTool, BashTool, GlobTool, GrepTool, WebFetchTool, WebSearchTool,
// FileWriteTool, FileEditTool, NotebookEditTool, AgentTool, PowerShellTool).
// Source: claude-code/src/tools/** and constants/toolLimits.ts TOOL_SUMMARY_MAX_LENGTH=50.

package messagerow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"goc/types"
)

const toolSummaryMaxLen = 50 // TS TOOL_SUMMARY_MAX_LENGTH

// VerboseToolOutputEnabled matches GOU_DEMO_VERBOSE_TOOL_OUTPUT used for tool_result preview;
// when true, tool_use rows show full formatNamedTool JSON instead of activity lines.
func VerboseToolOutputEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_VERBOSE_TOOL_OUTPUT")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func truncateToolSummary(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= toolSummaryMaxLen {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= toolSummaryMaxLen-1 {
			break
		}
		b.WriteRune(r)
		n++
	}
	b.WriteString("…")
	return b.String()
}

// DisplayPathForActivity approximates TS getDisplayPath: prefer path relative to cwd when safe.
func DisplayPathForActivity(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(p)
	}
	if rel, err := filepath.Rel(cwd, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return rel
	}
	return filepath.Clean(p)
}

func decodeToolInputMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return nil
	}
	return m
}

func strFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

// ActivityLineForToolUse returns a single-line user-facing activity string, or "" if unknown.
func ActivityLineForToolUse(toolName string, input json.RawMessage) string {
	name := strings.TrimSpace(toolName)
	m := decodeToolInputMap(input)
	switch name {
	case "Read":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Reading file"
		}
		return "Reading " + truncateToolSummary(DisplayPathForActivity(fp))
	case "Bash", "BashZog":
		cmd := strFromMap(m, "command")
		if cmd == "" {
			return "Running command"
		}
		desc := strFromMap(m, "description")
		if strings.TrimSpace(desc) == "" {
			desc = truncateToolSummary(cmd)
		} else {
			desc = truncateToolSummary(desc)
		}
		return "Running " + desc
	case "Glob":
		pat := strFromMap(m, "pattern")
		if pat == "" {
			return "Finding files"
		}
		return "Finding " + truncateToolSummary(pat)
	case "Grep":
		pat := strFromMap(m, "pattern")
		if pat == "" {
			return "Searching"
		}
		return "Searching for " + truncateToolSummary(pat)
	case "WebFetch":
		u := strFromMap(m, "url")
		if u == "" {
			return "Fetching web page"
		}
		return "Fetching " + truncateToolSummary(u)
	case "WebSearch":
		q := strFromMap(m, "query")
		if q == "" {
			return "Searching the web"
		}
		return "Searching for " + truncateToolSummary(q)
	case "Write":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Writing file"
		}
		return "Writing " + truncateToolSummary(DisplayPathForActivity(fp))
	case "Edit":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Editing file"
		}
		return "Editing " + truncateToolSummary(DisplayPathForActivity(fp))
	case "NotebookEdit":
		np := strFromMap(m, "notebook_path")
		if np == "" {
			return "Editing notebook"
		}
		return "Editing notebook " + truncateToolSummary(DisplayPathForActivity(np))
	case "Agent", "Task":
		d := strFromMap(m, "description")
		if strings.TrimSpace(d) == "" {
			return "Running task"
		}
		return d
	case "PowerShell":
		cmd := strFromMap(m, "command")
		if cmd == "" {
			return "Running command"
		}
		desc := strFromMap(m, "description")
		if strings.TrimSpace(desc) == "" {
			desc = truncateToolSummary(cmd)
		} else {
			desc = truncateToolSummary(desc)
		}
		return "Running " + desc
	default:
		return ""
	}
}

// ActivitySegmentForToolBlock builds SegToolUse / SegServerToolUse text like TS activity line.
func ActivitySegmentForToolBlock(b types.MessageContentBlock, kind SegmentKind) []Segment {
	if VerboseToolOutputEnabled() {
		k := "tool_use"
		if kind == SegServerToolUse {
			k = "server_tool_use"
		}
		return []Segment{{Kind: kind, Text: formatNamedTool(k, b.Name, b.ID, b.Input)}}
	}
	line := ActivityLineForToolUse(b.Name, b.Input)
	if line == "" {
		k := "tool_use"
		if kind == SegServerToolUse {
			k = "server_tool_use"
		}
		line = formatNamedTool(k, b.Name, b.ID, b.Input)
	}
	facing, paren, hint := ToolChromeParts(b.Name, b.Input)
	if facing == "" {
		return []Segment{{Kind: kind, Text: line, ToolUseID: b.ID}}
	}
	return []Segment{{Kind: kind, Text: line, ToolUseID: b.ID, ToolFacing: facing, ToolParen: paren, ToolHint: hint}}
}
