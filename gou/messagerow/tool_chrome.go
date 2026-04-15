// Tool chrome mirrors TS AssistantToolUseMessage row1 (userFacingName + renderToolUseMessage)
// and CollapsedReadSearch-style ⎿ hints (claude-code/src/utils/collapseReadSearch.ts latestDisplayHint).

package messagerow

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"goc/types"
)

const maxCommandHintChars = 300

// ToolChromeParts returns facing name, parenthetical detail (TS renderToolUseMessage plain text),
// and hint body for the ⎿ row while the tool_use is unresolved (TS active collapsed group).
func ToolChromeParts(toolName string, input json.RawMessage) (facing, paren, hint string) {
	m := decodeToolInputMap(input)
	switch toolName {
	case "Read":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Read", "", ""
		}
		paren, hint := readParenAndHintForReadInput(m)
		return "Read", paren, hint
	case "Grep":
		pat := strFromMap(m, "pattern")
		if pat == "" {
			return "Search", "", ""
		}
		path := strFromMap(m, "path")
		if path == "" {
			paren = fmt.Sprintf(`pattern: "%s"`, pat)
		} else {
			paren = fmt.Sprintf(`pattern: "%s", path: "%s"`, pat, DisplayPathForActivity(path))
		}
		hint = `"` + pat + `"`
		return "Search", paren, hint
	case "Glob":
		pat := strFromMap(m, "pattern")
		if pat == "" {
			return "Search", "", ""
		}
		path := strFromMap(m, "path")
		if path == "" {
			paren = fmt.Sprintf(`pattern: "%s"`, pat)
		} else {
			paren = fmt.Sprintf(`pattern: "%s", path: "%s"`, pat, DisplayPathForActivity(path))
		}
		hint = `"` + pat + `"`
		return "Search", paren, hint
	case "Bash", "BashZog":
		cmd := strFromMap(m, "command")
		if cmd == "" {
			return "Bash", "", ""
		}
		return "Bash", truncateToolSummary(cmd), commandAsHintBody(cmd)
	case "Write":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Write", "", ""
		}
		d := DisplayPathForActivity(fp)
		return "Write", d, d
	case "Edit":
		fp := strFromMap(m, "file_path")
		if fp == "" {
			return "Edit", "", ""
		}
		d := DisplayPathForActivity(fp)
		return "Edit", d, d
	case "NotebookEdit":
		np := strFromMap(m, "notebook_path")
		if np == "" {
			return "NotebookEdit", "", ""
		}
		d := DisplayPathForActivity(np)
		return "NotebookEdit", d, d
	case "WebFetch":
		u := strFromMap(m, "url")
		if u == "" {
			return "Fetch", "", ""
		}
		t := truncateToolSummary(u)
		return "Fetch", t, t
	case "WebSearch":
		q := strFromMap(m, "query")
		if q == "" {
			return "WebSearch", "", ""
		}
		t := truncateToolSummary(q)
		return "WebSearch", t, t
	case "PowerShell":
		cmd := strFromMap(m, "command")
		if cmd == "" {
			return "PowerShell", "", ""
		}
		return "PowerShell", truncateToolSummary(cmd), commandAsHintBody(cmd)
	case "Task", "Agent":
		d := strFromMap(m, "description")
		if strings.TrimSpace(d) == "" {
			return toolName, "", ""
		}
		t := truncateToolSummary(d)
		return toolName, t, t
	default:
		return "", "", ""
	}
}

// commandAsHintBody mirrors TS commandAsHint without the "$ " prefix (gou-demo uses ⎿ only).
func commandAsHintBody(command string) string {
	lines := strings.Split(command, "\n")
	var b strings.Builder
	first := true
	for _, ln := range lines {
		s := strings.TrimSpace(strings.Join(strings.Fields(ln), " "))
		if s == "" {
			continue
		}
		if !first {
			b.WriteByte('\n')
		}
		first = false
		b.WriteString(s)
	}
	out := b.String()
	if utf8.RuneCountInString(out) > maxCommandHintChars {
		out = string([]rune(out)[:maxCommandHintChars-1]) + "…"
	}
	return out
}

// readParenAndHintForReadInput mirrors TS FileReadTool/UI.tsx renderToolUseMessage (display path + optional · lines / pages).
func readParenAndHintForReadInput(m map[string]any) (paren, hint string) {
	fp := strFromMap(m, "file_path")
	if fp == "" {
		return "", ""
	}
	d := DisplayPathForActivity(fp)
	hint = d
	pages := strFromMap(m, "pages")
	if pages != "" {
		return d + " · pages " + pages, hint
	}
	off, hasOff := intFromMapAny(m, "offset")
	lim, hasLim := intFromMapAny(m, "limit")
	if !hasOff && !hasLim {
		return d, hint
	}
	startLine := 1
	if hasOff && off > 0 {
		startLine = off
	}
	if hasLim && lim > 0 {
		endLine := startLine + lim - 1
		return fmt.Sprintf("%s · lines %d-%d", d, startLine, endLine), hint
	}
	return fmt.Sprintf("%s · from line %d", d, startLine), hint
}

func intFromMapAny(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

// CollectResolvedToolUseIDs returns tool_use_id values that already have a user tool_result (TS lookups.resolvedToolUseIDs).
func CollectResolvedToolUseIDs(msgs []types.Message) map[string]struct{} {
	out := make(map[string]struct{})
	for _, msg := range msgs {
		if msg.Type != types.MessageTypeUser {
			continue
		}
		msg = NormalizeMessageJSON(msg)
		if len(msg.Content) == 0 {
			continue
		}
		var blocks []types.MessageContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_result" && strings.TrimSpace(b.ToolUseID) != "" {
				out[b.ToolUseID] = struct{}{}
			}
		}
	}
	return out
}
