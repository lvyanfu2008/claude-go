package messages

import (
	"encoding/json"
	"strings"

	"goc/gou/messagerow"
	"goc/types"
)

// userMessageHasPromptText is true when a user message carries non-empty text content.
func UserMessageHasPromptText(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	var blocks []types.MessageContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

// UserMessageRendersOnlyFoldedToolStubs is true when this user row would render only folded
// tool_result/advisor_tool_result stubs (ToolBodyOmitted), with no other visible segments.
// Matches actual [SegmentsFromMessageOpts] output so unknown content block types or API quirks
// do not leave a lone "↩ tool_result tool_use_id=..." line under assistant ⎿ summaries.
func UserMessageRendersOnlyFoldedToolStubs(msg types.Message, ropts *messagerow.RenderOpts) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	segs := messagerow.SegmentsFromMessageOpts(msg, ropts)
	if len(segs) == 0 {
		return false
	}
	hasFoldedTool := false
	for _, s := range segs {
		switch s.Kind {
		case messagerow.SegTextMarkdown:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		case messagerow.SegToolResult, messagerow.SegAdvisorToolResult:
			if !s.ToolBodyOmitted {
				return false
			}
			hasFoldedTool = true
		case messagerow.SegThinking:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		default:
			return false
		}
	}
	return hasFoldedTool
}

// SkipFoldedToolResultStubInPrompt hides user messages that only render folded tool_result stubs
// (prompt always; transcript unless ctrl+e show-all or dump).
// The assistant tool_use row already shows the summary; omitting avoids duplicate ↩ tool_result lines.
// Parameters mirror the model state needed without direct model access.
func SkipFoldedToolResultStubInPrompt(msg types.Message, verboseToolOutput bool, ropts *messagerow.RenderOpts, isPrompt, isTranscript, showAll, dumpMode bool) bool {
	if verboseToolOutput {
		return false
	}
	if !UserMessageRendersOnlyFoldedToolStubs(msg, ropts) {
		return false
	}
	if isPrompt {
		return true
	}
	if isTranscript && !showAll && !dumpMode {
		return true
	}
	return false
}

// PriorNonEmptyAssistantText reports whether any earlier segment is non-empty assistant markdown.
// One ⏺/● marks the start of the assistant "paragraph"; tool title lines after that omit the lead glyph.
func PriorNonEmptyAssistantText(segs []messagerow.Segment, idx int) bool {
	for j := 0; j < idx && j < len(segs); j++ {
		if segs[j].Kind == messagerow.SegTextMarkdown && strings.TrimSpace(segs[j].Text) != "" {
			return true
		}
	}
	return false
}

// toolUseResolved reports whether a tool_use id is in the resolved set.
func toolUseResolved(resolved map[string]struct{}, toolUseID string) bool {
	if resolved == nil || toolUseID == "" {
		return false
	}
	_, ok := resolved[toolUseID]
	return ok
}

// ToolUseResolvedForDisplay treats a tool as resolved if it is in the resolved map, or (when detail is on)
// if tool_result payload exists for that id — avoids stale resolved maps skipping ⏺+⎿ stats.
func ToolUseResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, toolUseID string, allowResultPayloadAsResolved bool) bool {
	if toolUseID == "" {
		return false
	}
	if resolved != nil {
		if _, ok := resolved[toolUseID]; ok {
			return true
		}
	}
	if allowResultPayloadAsResolved && toolResultByID != nil {
		raw, ok := toolResultByID[toolUseID]
		if ok && len(raw) > 0 {
			return true
		}
	}
	return false
}

// ToolUseSummaryLineResolvedForDisplay is true when every merged tool_use id in a SegToolUseSummaryLine
// has a result (or resolved map entry).
func ToolUseSummaryLineResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, toolUseIDs []string, toolUseID string, allowResultPayloadAsResolved bool) bool {
	ids := toolUseIDs
	if len(ids) == 0 {
		return ToolUseResolvedForDisplay(resolved, toolResultByID, toolUseID, allowResultPayloadAsResolved)
	}
	for _, id := range ids {
		if !ToolUseResolvedForDisplay(resolved, toolResultByID, id, allowResultPayloadAsResolved) {
			return false
		}
	}
	return true
}
