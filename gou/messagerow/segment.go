// Package messagerow maps types.Message to terminal-friendly segments
// (MessageRow.tsx / Message.tsx: content blocks + grouped_tool_use + collapsed_read_search).
package messagerow

import (
	"encoding/json"
	"os"
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
	// SegToolUseSummaryLine is a single dim line for standalone Grep/Glob/Read (SearchReadSummaryText-style).
	SegToolUseSummaryLine
	SegDisplayHint
	SegSkillListingAvailable // Num = skill count; TS AttachmentMessage skill_listing
	SegUnknown
)

// Segment is one rendered unit (plain text inside; caller applies lipgloss / markdown).
type Segment struct {
	Kind SegmentKind
	Text string
	// Num is used when Kind == SegSkillListingAvailable (TS bold skillCount + " skills available").
	Num int
	// ToolUseID / ToolFacing / ToolParen / ToolHint mirror TS AssistantToolUseMessage + ⎿ hint row.
	ToolUseID  string
	ToolFacing string
	ToolParen  string
	ToolHint   string
	// IsToolError is set for tool_result / advisor blocks when TS is_error is true (OutputLine error styling).
	IsToolError bool
	// ToolBodyOmitted is true when RenderOpts.FoldToolResultBody omitted block output (prompt stub; ctrl+o opens transcript).
	ToolBodyOmitted bool
}

const maxDisplayNest = 8

// RenderOpts optional flags for transcript-style rendering (TS Messages.tsx showAllInTranscript).
type RenderOpts struct {
	// ShowAllInTranscript expands collapsed_read_search and grouped_tool_use bodies (gou-demo ctrl+e in transcript).
	ShowAllInTranscript bool
	// VerboseCollapsedReadSearch renders nested assistant/user tool rows inside collapsed_read_search (TS verbose || isTranscriptMode).
	VerboseCollapsedReadSearch bool
	// FoldToolResultBody hides tool_result / advisor_tool_result payload in the main prompt (gou-demo: ctrl+o opens transcript to read).
	FoldToolResultBody bool
	// CollapsedReadSearchActive is true only for the in-flight tail collapsed_read_search row (TS MessageRow isActiveCollapsedGroup).
	CollapsedReadSearchActive bool
	// GroupedAgentLookups provides resolved/error states for grouped_tool_use items.
	GroupedAgentLookups *GroupedAgentLookups
	// ResolvedToolUseIDs is tool_use_id values that already have a user tool_result (for summary-line active/past tense).
	ResolvedToolUseIDs map[string]struct{}
}

// SegmentsFromMessage handles message.type + content[] blocks (TS RenderableMessage / MessageRow displayMsg).
func SegmentsFromMessage(msg types.Message) []Segment {
	return SegmentsFromMessageOpts(msg, nil)
}

// SegmentsFromMessageOpts is like [SegmentsFromMessage] with optional transcript expand (nil opts == default).
func SegmentsFromMessageOpts(msg types.Message, opts *RenderOpts) []Segment {
	return segmentsFromMessageDepthOpts(msg, 0, opts)
}

func segmentsFromMessageDepthOpts(msg types.Message, depth int, opts *RenderOpts) []Segment {
	if depth > maxDisplayNest {
		return []Segment{{Kind: SegUnknown, Text: "…"}}
	}
	msg = NormalizeMessageJSON(msg)
	switch msg.Type {
	case types.MessageTypeAttachment:
		return segmentsFromAttachment(msg)
	case types.MessageTypeGroupedToolUse:
		return segmentsGroupedToolUse(msg, depth, opts)
	case types.MessageTypeCollapsedReadSearch:
		return segmentsCollapsedReadSearch(msg, depth, opts)
	default:
		return segmentsFromContentArray(msg, opts)
	}
}

func segmentsGroupedToolUse(msg types.Message, depth int, opts *RenderOpts) []Segment {
	if opts != nil && opts.ShowAllInTranscript {
		var out []Segment
		out = append(out, Segment{Kind: SegGroupedToolUse, Text: "grouped_tool_use"})
		for i := range msg.Messages {
			out = append(out, segmentsFromMessageDepthOpts(msg.Messages[i], depth+1, opts)...)
		}
		for i := range msg.Results {
			out = append(out, segmentsFromMessageDepthOpts(msg.Results[i], depth+1, opts)...)
		}
		return out
	}
	
	// Format as multiple line segments or summary using FormatGroupedAgentToolUse
	var lookups *GroupedAgentLookups
	if opts != nil {
		lookups = opts.GroupedAgentLookups
	}
	
	return FormatGroupedAgentToolUse(msg, lookups)
}

func segmentsCollapsedReadSearch(msg types.Message, depth int, opts *RenderOpts) []Segment {
	isActive := opts != nil && opts.CollapsedReadSearchActive
	summary := SearchReadSummaryTextFromMessage(isActive, msg)
	if opts != nil && opts.VerboseCollapsedReadSearch && len(msg.Messages) > 0 {
		return segmentsCollapsedReadSearchVerbose(msg, depth, opts, isActive, summary)
	}
	if opts != nil && opts.ShowAllInTranscript {
		var parts []string
		if strings.TrimSpace(summary) != "" {
			parts = append(parts, strings.TrimSpace(summary))
		}
		if len(msg.ReadFilePaths) > 0 {
			parts = append(parts, "Files:\n- "+strings.Join(msg.ReadFilePaths, "\n- "))
		}
		if len(msg.SearchArgs) > 0 {
			parts = append(parts, "Search terms:\n- "+strings.Join(msg.SearchArgs, "\n- "))
		}
		text := strings.Join(parts, "\n")
		if text == "" {
			text = "…"
		}
		out := []Segment{{Kind: SegCollapsedReadSearch, Text: text}}
		// Transcript show-all: keep ⎿ hint for expanded inspection (TS focuses prompt parity on active-only).
		if msg.LatestDisplayHint != nil {
			h := strings.TrimSpace(*msg.LatestDisplayHint)
			if h != "" {
				h = strings.ReplaceAll(h, "\r\n", "\n")
				h = strings.ReplaceAll(h, "\n", " ")
				if len(h) > 400 {
					h = h[:400] + "…"
				}
				out = append(out, Segment{Kind: SegDisplayHint, Text: "  ⎿  " + h})
			}
		}
		if msg.DisplayMessage != nil {
			out = append(out, segmentsFromMessageDepthOpts(*msg.DisplayMessage, depth+1, opts)...)
		}
		return out
	}
	if summary == "" {
		summary = "…"
	}
	line := summary + CtrlOToExpandHint
	out := []Segment{{Kind: SegCollapsedReadSearch, Text: line}}
	// TS CollapsedReadSearchContent: ⎿ + latestDisplayHint only when isActiveGroup.
	if isActive && msg.LatestDisplayHint != nil {
		h := strings.TrimSpace(*msg.LatestDisplayHint)
		if h != "" {
			h = strings.ReplaceAll(h, "\r\n", "\n")
			h = strings.ReplaceAll(h, "\n", " ")
			if len(h) > 400 {
				h = h[:400] + "…"
			}
			out = append(out, Segment{Kind: SegDisplayHint, Text: "  ⎿  " + h})
		}
	}
	if msg.DisplayMessage != nil {
		out = append(out, segmentsFromMessageDepthOpts(*msg.DisplayMessage, depth+1, opts)...)
	}
	return out
}

// segmentsCollapsedReadSearchVerbose mirrors CollapsedReadSearchContent.tsx verbose / transcript tool lines.
func segmentsCollapsedReadSearchVerbose(msg types.Message, depth int, opts *RenderOpts, isActive bool, summary string) []Segment {
	if strings.TrimSpace(summary) == "" {
		summary = "…"
	}
	line := summary + CtrlOToExpandHint
	out := []Segment{{Kind: SegCollapsedReadSearch, Text: line}}
	if isActive && msg.LatestDisplayHint != nil {
		h := strings.TrimSpace(*msg.LatestDisplayHint)
		if h != "" {
			h = strings.ReplaceAll(h, "\r\n", "\n")
			h = strings.ReplaceAll(h, "\n", " ")
			if len(h) > 400 {
				h = h[:400] + "…"
			}
			out = append(out, Segment{Kind: SegDisplayHint, Text: "  ⎿  " + h})
		}
	}
	for i := range msg.Messages {
		out = append(out, segmentsFromMessageDepthOpts(msg.Messages[i], depth+1, opts)...)
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

func segmentsFromContentArray(msg types.Message, opts *RenderOpts) []Segment {
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
		out = append(out, segmentFromBlock(b, opts)...)
	}
	if len(out) == 0 && (msg.Type == types.MessageTypeAssistant || msg.Type == types.MessageTypeUser) {
		// e.g. content: [{type:"text",text:""}] or only unrecognized blocks — avoid a blank body under the role header.
		return []Segment{{Kind: SegTextMarkdown, Text: "[" + string(msg.Type) + " · no visible text in content blocks]"}}
	}
	return out
}

func segmentFromBlock(b types.MessageContentBlock, opts *RenderOpts) []Segment {
	switch b.Type {
	case "text":
		if strings.TrimSpace(b.Text) == "" {
			return nil
		}
		return []Segment{{Kind: SegTextMarkdown, Text: b.Text}}
	case "tool_use":
		return ActivitySegmentForToolBlock(b, SegToolUse, opts)
	case "server_tool_use":
		return ActivitySegmentForToolBlock(b, SegServerToolUse, opts)
	case "advisor_tool_result":
		var sb strings.Builder
		sb.WriteString("advisor_tool_result")
		if b.ID != "" {
			sb.WriteString(" id=")
			sb.WriteString(b.ID)
		}
		isErr := b.IsError != nil && *b.IsError
		fold := opts != nil && opts.FoldToolResultBody && !VerboseToolOutputEnabled()
		if fold {
			return []Segment{{Kind: SegAdvisorToolResult, Text: strings.TrimSpace(sb.String()), IsToolError: isErr, ToolBodyOmitted: true}}
		}
		sb.WriteByte('\n')
		sb.WriteString(toolResultContentPreview(b.Content))
		return []Segment{{Kind: SegAdvisorToolResult, Text: strings.TrimSpace(sb.String()), IsToolError: isErr}}
	case "tool_result":
		var sb strings.Builder
		sb.WriteString("tool_result")
		if b.ToolUseID != "" {
			sb.WriteString(" tool_use_id=")
			sb.WriteString(b.ToolUseID)
		}
		isErr := b.IsError != nil && *b.IsError
		if isErr {
			sb.WriteString(" [error]")
		}
		fold := opts != nil && opts.FoldToolResultBody && !VerboseToolOutputEnabled()
		if fold {
			return []Segment{{Kind: SegToolResult, Text: strings.TrimSpace(sb.String()), IsToolError: isErr, ToolBodyOmitted: true}}
		}
		sb.WriteByte('\n')
		sb.WriteString(toolResultContentPreview(b.Content))
		return []Segment{{Kind: SegToolResult, Text: strings.TrimSpace(sb.String()), IsToolError: isErr}}
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

func toolResultPreviewMax() int {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_VERBOSE_TOOL_OUTPUT")))
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return 24000
	}
	return 1200
}

func toolResultPreview(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	max := toolResultPreviewMax()
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return compactJSON(s, max)
	}
	return compactJSON(string(raw), max)
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
