package message

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/gou/messagerow"
	"goc/types"
)

// MessageRowContext carries per-message context computed by MessageRow.
type MessageRowContext struct {
	// IsUserContinuation is true when the previous message is also a user message.
	IsUserContinuation bool
	// IsActiveCollapsedGroup is true when a collapsed group's tools are still executing.
	IsActiveCollapsedGroup bool
	// ShouldAnimate is true when the message content may still be changing.
	ShouldAnimate bool
	// IsInProgress is true when a tool_use has not yet received its result.
	IsInProgress bool
}

// MessageRowBuildOpts are the inputs for building per-message row contexts.
type MessageRowBuildOpts struct {
	TranscriptMode       bool
	Verbose              bool
	InProgressToolUseIDs map[string]struct{}
	StreamingToolUseIDs  map[string]struct{}
	ResolvedToolUseIDs   map[string]struct{}
	SearchHighlight      string
	Columns              int
	Loading              bool
}

// BuildMessageRowContexts computes per-message rendering contexts.
// Mirrors MessageRow.tsx context computation.
func BuildMessageRowContexts(
	messages []*types.Message,
	opts MessageRowBuildOpts,
) []*MessageRowContext {
	if len(messages) == 0 {
		return nil
	}

	contexts := make([]*MessageRowContext, len(messages))
	for i, msg := range messages {
		ctx := &MessageRowContext{}

		// isUserContinuation: previous message also user
		if i > 0 && messages[i-1].Type == types.MessageTypeUser {
			ctx.IsUserContinuation = msg.Type == types.MessageTypeUser
		}

		// isInProgress: tool_use IDs for this message vs resolved set
		ctx.IsInProgress = !allToolUsesResolved(msg, opts.ResolvedToolUseIDs)

		// shouldAnimate: streaming or in-progress
		ctx.ShouldAnimate = opts.Loading && (hasStreamingTools(msg, opts.StreamingToolUseIDs) || ctx.IsInProgress)

		// isActiveCollapsedGroup
		if msg.Type == types.MessageTypeCollapsedReadSearch {
			ctx.IsActiveCollapsedGroup = hasAnyToolInProgress(msg, opts.InProgressToolUseIDs) ||
				(opts.Loading && !hasContentAfterIndex(messages, i))
		}

		contexts[i] = ctx
	}
	return contexts
}

// hasContentAfterIndex checks if any non-progress content follows the given index.
func hasContentAfterIndex(messages []*types.Message, idx int) bool {
	for i := idx + 1; i < len(messages); i++ {
		msg := messages[i]
		switch msg.Type {
		case types.MessageTypeProgress, types.MessageTypeAttachment:
			continue
		case types.MessageTypeSystem:
			continue
		case types.MessageTypeUser:
			// tool_results follow collapsed groups
			content, err := parseMessageContent(msg)
			if err == nil && len(content) > 0 {
				if blockType, _ := content[0]["type"].(string); blockType == "tool_result" {
					continue
				}
			}
			continue
		default:
			return true
		}
	}
	return false
}

// allToolUsesResolved checks whether every tool_use in the message has a resolved result.
func allToolUsesResolved(msg *types.Message, resolved map[string]struct{}) bool {
	if resolved == nil || len(resolved) == 0 {
		return false
	}
	// For collapsed_read_search, check nested Messages
	if msg.Type == types.MessageTypeCollapsedReadSearch {
		for i := range msg.Messages {
			if !allToolUsesResolved(&msg.Messages[i], resolved) {
				return false
			}
		}
		return true
	}
	// For grouped_tool_use, check nested Messages
	if msg.Type == types.MessageTypeGroupedToolUse {
		for i := range msg.Messages {
			if !allToolUsesResolved(&msg.Messages[i], resolved) {
				return false
			}
		}
		return true
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return true // can't determine, assume resolved
	}
	hasToolUse := false
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			hasToolUse = true
			if id, _ := block["id"].(string); id != "" {
				if _, ok := resolved[id]; !ok {
					return false
				}
			}
		}
	}
	if !hasToolUse {
		return true // vacuous truth: no tool_use blocks means all resolved
	}
	return true // all tool_use blocks found in resolved set
}

// hasAnyToolInProgress checks if any tool_use in the message is in the in-progress set.
func hasAnyToolInProgress(msg *types.Message, inProgress map[string]struct{}) bool {
	if inProgress == nil || len(inProgress) == 0 {
		return false
	}
	// For collapsed_read_search, check nested Messages
	if msg.Type == types.MessageTypeCollapsedReadSearch {
		for i := range msg.Messages {
			if hasAnyToolInProgress(&msg.Messages[i], inProgress) {
				return true
			}
		}
		return false
	}
	// For grouped_tool_use, check nested Messages
	if msg.Type == types.MessageTypeGroupedToolUse {
		for i := range msg.Messages {
			if hasAnyToolInProgress(&msg.Messages[i], inProgress) {
				return true
			}
		}
		return false
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return false
	}
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			if id, _ := block["id"].(string); id != "" {
				if _, ok := inProgress[id]; ok {
					return true
				}
			}
		}
	}
	return false
}

// hasStreamingTools checks if any tool_use in the message is in the streaming set.
func hasStreamingTools(msg *types.Message, streaming map[string]struct{}) bool {
	if streaming == nil || len(streaming) == 0 {
		return false
	}
	// For collapsed_read_search, check nested Messages
	if msg.Type == types.MessageTypeCollapsedReadSearch {
		for i := range msg.Messages {
			if hasStreamingTools(&msg.Messages[i], streaming) {
				return true
			}
		}
		return false
	}
	// For grouped_tool_use, check nested Messages
	if msg.Type == types.MessageTypeGroupedToolUse {
		for i := range msg.Messages {
			if hasStreamingTools(&msg.Messages[i], streaming) {
				return true
			}
		}
		return false
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return false
	}
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			if id, _ := block["id"].(string); id != "" {
				if _, ok := streaming[id]; ok {
					return true
				}
			}
		}
	}
	return false
}

// StreamingToolUse represents an in-flight tool_use from the store.
type StreamingToolUse struct {
	Name  string
	Input string
}

// RenderStreamingTail renders streaming content (text, thinking, tool uses) that
// hasn't been committed to messages yet. Mirrors TS behavior where streaming
// appears as continuation of the last assistant message.
func RenderStreamingTail(
	streamingText string,
	streamingThinking string,
	streamingToolUses []StreamingToolUse,
	ctx *RenderContext,
) []string {
	var lines []string

	// Streaming thinking
	if strings.TrimSpace(streamingThinking) != "" {
		lines = append(lines, "\x1b[2;3m∴ Thinking\x1b[0m")
	}

	// Streaming text — only first line gets the ⏺ bullet (matches committed AssistantMessageRenderer)
	if strings.TrimSpace(streamingText) != "" {
		textLines := renderMarkdown(streamingText, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
		for i, l := range textLines {
			if i == 0 {
				lines = append(lines, "⏺ "+l)
			} else {
				lines = append(lines, "  "+l)
			}
		}
	}

	// Streaming tool uses — when multiple are active, rotate through them
	// one at a time to keep output height stable (avoids layout jumping).
	if len(streamingToolUses) > 0 {
		if len(streamingToolUses) == 1 {
			tu := streamingToolUses[0]
			lines = append(lines, formatStreamingToolLine(tu, ctx))
		} else {
			idx := int(time.Now().UnixMilli()/2000) % len(streamingToolUses)
			tu := streamingToolUses[idx]
			line := formatStreamingToolLine(tu, ctx)
			line += fmt.Sprintf("  +%d more", len(streamingToolUses)-1)
			lines = append(lines, line)
		}
	}

	return lines
}

// formatStreamingToolLine formats a single streaming tool use as a display line.
func formatStreamingToolLine(tu StreamingToolUse, ctx *RenderContext) string {
	facing, paren, hint := messagerow.ToolChromeParts(tu.Name, json.RawMessage(tu.Input))
	line := "  ⎿ " + facing
	if paren != "" {
		line += " (" + paren + ")"
	}
	line += "…"
	if hint != "" {
		line += "\n     " + hint
	}
	if ctx.ShowToolUseCtrlOHint {
		line += " (ctrl+o to expand)"
	}
	return line
}
