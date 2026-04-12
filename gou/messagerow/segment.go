// Package messagerow maps types.Message to terminal-friendly segments
// (MessageRow.tsx / Message.tsx: content blocks + grouped_tool_use + collapsed_read_search).
package messagerow

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// SegmentKind classifies a slice of message body for styling and markdown pass-through.
type SegmentKind int

const (
	SegTextMarkdown SegmentKind = iota
	SegToolUse
	SegToolResult
	SegThinking
	SegServerToolUse
	SegAdvisorToolResult
	SegGroupedToolUse
	SegCollapsedReadSearch
	SegUnknown
)

// Segment is one rendered unit (plain text inside; caller applies lipgloss / markdown).
type Segment struct {
	Kind SegmentKind
	Text string
}

const maxDisplayNest = 8

// SegmentsFromMessage handles message.type + content[] blocks (TS RenderableMessage / MessageRow displayMsg).
func SegmentsFromMessage(msg types.Message) []Segment {
	return segmentsFromMessageDepth(msg, 0)
}

func segmentsFromMessageDepth(msg types.Message, depth int) []Segment {
	if depth > maxDisplayNest {
		return []Segment{{Kind: SegUnknown, Text: "…"}}
	}
	msg = NormalizeMessageJSON(msg)
	switch msg.Type {
	case types.MessageTypeGroupedToolUse:
		return segmentsGroupedToolUse(msg, depth)
	case types.MessageTypeCollapsedReadSearch:
		return segmentsCollapsedReadSearch(msg, depth)
	default:
		return segmentsFromContentArray(msg)
	}
}

func segmentsGroupedToolUse(msg types.Message, depth int) []Segment {
	var sb strings.Builder
	sb.WriteString("grouped_tool_use")
	if msg.ToolName != "" {
		sb.WriteString(" · ")
		sb.WriteString(msg.ToolName)
	}
	sb.WriteString(fmt.Sprintf(" · %d assistant · %d results", len(msg.Messages), len(msg.Results)))
	out := []Segment{{Kind: SegGroupedToolUse, Text: strings.TrimSpace(sb.String())}}
	if msg.DisplayMessage != nil {
		out = append(out, segmentsFromMessageDepth(*msg.DisplayMessage, depth+1)...)
	}
	return out
}

func segmentsCollapsedReadSearch(msg types.Message, depth int) []Segment {
	var sb strings.Builder
	sb.WriteString("collapsed_read_search")
	sb.WriteString(fmt.Sprintf(" · read=%d search=%d list=%d", msg.ReadCount, msg.SearchCount, msg.ListCount))
	if len(msg.ReadFilePaths) > 0 {
		sb.WriteString("\npaths: ")
		sb.WriteString(compactJoin(msg.ReadFilePaths, 5, 120))
	}
	if len(msg.SearchArgs) > 0 {
		sb.WriteString("\nsearch: ")
		sb.WriteString(compactJoin(msg.SearchArgs, 3, 120))
	}
	if msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != "" {
		sb.WriteString("\nhint: ")
		sb.WriteString(compactJSON(*msg.LatestDisplayHint, 200))
	}
	out := []Segment{{Kind: SegCollapsedReadSearch, Text: strings.TrimSpace(sb.String())}}
	if msg.DisplayMessage != nil {
		out = append(out, segmentsFromMessageDepth(*msg.DisplayMessage, depth+1)...)
	}
	return out
}

func compactJoin(ss []string, maxN, maxRunes int) string {
	if len(ss) == 0 {
		return ""
	}
	end := len(ss)
	if end > maxN {
		end = maxN
	}
	s := strings.Join(ss[:end], ", ")
	if len(ss) > maxN {
		s += ", …"
	}
	return compactJSON(s, maxRunes)
}

func segmentsFromContentArray(msg types.Message) []Segment {
	if len(msg.Content) == 0 {
		if msg.Type == types.MessageTypeUser || msg.Type == types.MessageTypeAssistant {
			return []Segment{{Kind: SegTextMarkdown, Text: "[" + string(msg.Type) + " · empty content]"}}
		}
		return []Segment{{Kind: SegTextMarkdown, Text: "[" + string(msg.Type) + "]"}}
	}
	var blocks []types.MessageContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return []Segment{{Kind: SegTextMarkdown, Text: string(msg.Content)}}
	}
	if len(blocks) == 0 {
		return []Segment{{Kind: SegTextMarkdown, Text: "[" + string(msg.Type) + "]"}}
	}
	var out []Segment
	for _, b := range blocks {
		out = append(out, segmentFromBlock(b)...)
	}
	if len(out) == 0 && (msg.Type == types.MessageTypeAssistant || msg.Type == types.MessageTypeUser) {
		// e.g. content: [{type:"text",text:""}] or only unrecognized blocks — avoid a blank body under the role header.
		return []Segment{{Kind: SegTextMarkdown, Text: "[" + string(msg.Type) + " · no visible text in content blocks]"}}
	}
	return out
}

func segmentFromBlock(b types.MessageContentBlock) []Segment {
	switch b.Type {
	case "text":
		if strings.TrimSpace(b.Text) == "" {
			return nil
		}
		return []Segment{{Kind: SegTextMarkdown, Text: b.Text}}
	case "tool_use":
		return []Segment{{Kind: SegToolUse, Text: formatNamedTool("tool_use", b.Name, b.ID, b.Input)}}
	case "server_tool_use":
		return []Segment{{Kind: SegServerToolUse, Text: formatNamedTool("server_tool_use", b.Name, b.ID, b.Input)}}
	case "advisor_tool_result":
		var sb strings.Builder
		sb.WriteString("advisor_tool_result")
		if b.ID != "" {
			sb.WriteString(" id=")
			sb.WriteString(b.ID)
		}
		sb.WriteByte('\n')
		sb.WriteString(toolResultContentPreview(b.Content))
		return []Segment{{Kind: SegAdvisorToolResult, Text: strings.TrimSpace(sb.String())}}
	case "tool_result":
		var sb strings.Builder
		sb.WriteString("tool_result")
		if b.ToolUseID != "" {
			sb.WriteString(" tool_use_id=")
			sb.WriteString(b.ToolUseID)
		}
		if b.IsError != nil && *b.IsError {
			sb.WriteString(" [error]")
		}
		sb.WriteByte('\n')
		sb.WriteString(toolResultContentPreview(b.Content))
		return []Segment{{Kind: SegToolResult, Text: strings.TrimSpace(sb.String())}}
	case "thinking", "redacted_thinking":
		t := b.Thinking
		if t == "" {
			t = "[" + b.Type + "]"
		}
		return []Segment{{Kind: SegThinking, Text: t}}
	default:
		var sb strings.Builder
		sb.WriteString("block type=")
		sb.WriteString(b.Type)
		if b.Text != "" {
			sb.WriteByte('\n')
			sb.WriteString(b.Text)
		}
		if len(b.Input) > 0 {
			sb.WriteByte('\n')
			sb.WriteString(compactJSON(string(b.Input), 400))
		}
		return []Segment{{Kind: SegUnknown, Text: sb.String()}}
	}
}

func formatNamedTool(kind, name, id string, input json.RawMessage) string {
	var sb strings.Builder
	sb.WriteString(kind)
	sb.WriteByte(' ')
	if name != "" {
		sb.WriteString(name)
	} else {
		sb.WriteString("(unnamed)")
	}
	if id != "" {
		sb.WriteString(" id=")
		sb.WriteString(id)
	}
	sb.WriteByte('\n')
	if len(input) > 0 {
		sb.WriteString(compactJSON(string(input), 800))
	}
	return strings.TrimSpace(sb.String())
}

func toolResultPreview(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return compactJSON(s, 1200)
	}
	return compactJSON(string(raw), 1200)
}

func toolResultContentPreview(raw json.RawMessage) string {
	return strings.TrimSpace(toolResultPreview(raw))
}

func compactJSON(s string, max int) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
