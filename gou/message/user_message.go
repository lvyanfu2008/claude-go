package message

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/types"
)

// UserMessageRenderer renders user messages.
type UserMessageRenderer struct{}

// CanRender returns true for user messages.
func (r *UserMessageRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeUser
}

// Render renders a user message.
func (r *UserMessageRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	if msg.IsCompactSummary != nil && *msg.IsCompactSummary {
		return r.renderCompactSummary(msg, ctx)
	}

	// Parse message content
	content, err := parseMessageContent(msg)
	if err != nil {
		return []string{fmt.Sprintf("Error parsing user message: %v", err)}, nil
	}

	var lines []string
	for i, block := range content {
		blockLines, err := r.renderContentBlock(block, ctx, i, len(content))
		if err != nil {
			return []string{fmt.Sprintf("Error rendering block: %v", err)}, nil
		}
		lines = append(lines, blockLines...)
	}

	return lines, nil
}

// Measure measures a user message.
func (r *UserMessageRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	if msg.IsCompactSummary != nil && *msg.IsCompactSummary {
		return 1, nil // Compact summary is always 1 line
	}

	content, err := parseMessageContent(msg)
	if err != nil {
		return 1, nil
	}

	totalLines := 0
	for i, block := range content {
		blockLines := r.measureContentBlock(block, ctx, i, len(content))
		totalLines += blockLines
	}

	return totalLines, nil
}

// renderCompactSummary renders a compact summary message.
func (r *UserMessageRenderer) renderCompactSummary(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// TODO: Implement compact summary rendering
	// Similar to TS CompactSummary component
	return []string{"[Compact summary]"}, nil
}

// renderContentBlock renders a content block.
func (r *UserMessageRenderer) renderContentBlock(block map[string]interface{}, ctx *RenderContext, index, total int) ([]string, error) {
	blockType, _ := block["type"].(string)

	switch blockType {
	case "text":
		return r.renderTextBlock(block, ctx)
	case "image":
		return r.renderImageBlock(block, ctx)
	case "tool_use":
		return r.renderToolUseBlock(block, ctx)
	case "tool_result":
		return r.renderToolResultBlock(block, ctx)
	default:
		return []string{fmt.Sprintf("[Unknown block type: %s]", blockType)}, nil
	}
}

// measureContentBlock measures a content block.
func (r *UserMessageRenderer) measureContentBlock(block map[string]interface{}, ctx *RenderContext, index, total int) int {
	blockType, _ := block["type"].(string)

	switch blockType {
	case "text":
		return r.measureTextBlock(block, ctx)
	case "image":
		return 1 // Image placeholder is 1 line
	case "tool_use":
		return 1 // Tool use placeholder
	case "tool_result":
		return r.measureToolResultBlock(block, ctx)
	default:
		return 1
	}
}

// renderTextBlock renders a text block.
func (r *UserMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)

	// Check for special message types
	if strings.Contains(text, "<bash-input>") {
		return r.renderBashInput(text, ctx)
	}
	if strings.Contains(text, "<bash-stdout") || strings.Contains(text, "<bash-stderr") {
		return r.renderBashOutput(text, ctx)
	}
	if strings.Contains(text, "<local-command-stdout") || strings.Contains(text, "<local-command-stderr") {
		return r.renderLocalCommandOutput(text, ctx)
	}

	// Regular user prompt
	return renderMarkdown(text, getContainerWidth(ctx), ctx.Theme), nil
}

// measureTextBlock measures a text block.
func (r *UserMessageRenderer) measureTextBlock(block map[string]interface{}, ctx *RenderContext) int {
	text, _ := block["text"].(string)

	// Special messages are usually 1 line
	if strings.Contains(text, "<bash-input>") ||
		strings.Contains(text, "<bash-stdout") ||
		strings.Contains(text, "<bash-stderr") ||
		strings.Contains(text, "<local-command-stdout") ||
		strings.Contains(text, "<local-command-stderr") {
		return 1
	}

	// Regular text
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme)
	return len(lines)
}

// renderBashInput renders bash input.
func (r *UserMessageRenderer) renderBashInput(text string, ctx *RenderContext) ([]string, error) {
	// Extract command from XML-like tag
	// TODO: Parse properly
	cmdStart := strings.Index(text, ">")
	cmdEnd := strings.LastIndex(text, "<")
	if cmdStart > 0 && cmdEnd > cmdStart {
		cmd := text[cmdStart+1 : cmdEnd]
		return []string{fmt.Sprintf("$ %s", cmd)}, nil
	}
	return []string{"$ [bash command]"}, nil
}

// renderBashOutput renders bash output.
func (r *UserMessageRenderer) renderBashOutput(text string, ctx *RenderContext) ([]string, error) {
	// TODO: Parse and render bash output properly
	return []string{"[bash output]"}, nil
}

// renderLocalCommandOutput renders local command output.
func (r *UserMessageRenderer) renderLocalCommandOutput(text string, ctx *RenderContext) ([]string, error) {
	// TODO: Parse and render local command output
	return []string{"[local command output]"}, nil
}

// renderImageBlock renders an image block.
func (r *UserMessageRenderer) renderImageBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// TODO: Implement image rendering
	return []string{"[Image]"}, nil
}

// renderToolUseBlock renders a tool use block.
func (r *UserMessageRenderer) renderToolUseBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// User messages shouldn't have tool_use blocks normally
	return []string{"[Tool use in user message]"}, nil
}

// renderToolResultBlock renders a tool result block.
func (r *UserMessageRenderer) renderToolResultBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// TODO: Implement tool result rendering
	// Similar to TS UserToolResultMessage
	return []string{"[Tool result]"}, nil
}

// measureToolResultBlock measures a tool result block.
func (r *UserMessageRenderer) measureToolResultBlock(block map[string]interface{}, ctx *RenderContext) int {
	// Tool results are usually collapsed to 1 line in prompt mode
	if !ctx.IsTranscript && !ctx.Verbose {
		return 1
	}
	// In transcript or verbose mode, show full content
	// TODO: Calculate actual height
	return 3
}

// Helper function to parse message content
func parseMessageContent(msg *types.Message) ([]map[string]interface{}, error) {
	var content []map[string]interface{}

	// Try Content field first
	if len(msg.Content) > 0 {
		if err := json.Unmarshal(msg.Content, &content); err == nil {
			return content, nil
		}
	}

	// Try Message field
	if len(msg.Message) > 0 {
		var messageObj struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(msg.Message, &messageObj); err == nil && len(messageObj.Content) > 0 {
			if err := json.Unmarshal(messageObj.Content, &content); err == nil {
				return content, nil
			}
		}
	}

	// Fallback: treat as single text block
	if msg.Content != nil {
		var text string
		if err := json.Unmarshal(msg.Content, &text); err == nil {
			return []map[string]interface{}{{"type": "text", "text": text}}, nil
		}
	}

	return nil, fmt.Errorf("could not parse message content")
}