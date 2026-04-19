package message

import (
	"fmt"
	"strings"

	"goc/types"
)

// AssistantMessageRenderer renders assistant messages.
type AssistantMessageRenderer struct{}

// CanRender returns true for assistant messages.
func (r *AssistantMessageRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeAssistant
}

// Render renders an assistant message.
func (r *AssistantMessageRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	content, err := parseMessageContent(msg)
	if err != nil {
		return []string{fmt.Sprintf("Error parsing assistant message: %v", err)}, nil
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

// Measure measures an assistant message.
func (r *AssistantMessageRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	content, err := parseMessageContent(msg)
	if err != nil {
		return 1, nil
	}

	totalLines := 0
	for _, block := range content {
		blockLines := r.measureContentBlock(block, ctx)
		totalLines += blockLines
	}

	return totalLines, nil
}

// renderContentBlock renders a content block.
func (r *AssistantMessageRenderer) renderContentBlock(block map[string]interface{}, ctx *RenderContext, index, total int) ([]string, error) {
	blockType, _ := block["type"].(string)

	switch blockType {
	case "text":
		return r.renderTextBlock(block, ctx)
	case "thinking":
		return r.renderThinkingBlock(block, ctx)
	case "tool_use":
		// Tool use blocks are handled by ToolUseMessageRenderer
		return []string{"[Tool use - should be handled by tool use renderer]"}, nil
	default:
		return []string{fmt.Sprintf("[Unknown assistant block type: %s]", blockType)}, nil
	}
}

// measureContentBlock measures a content block.
func (r *AssistantMessageRenderer) measureContentBlock(block map[string]interface{}, ctx *RenderContext) int {
	blockType, _ := block["type"].(string)

	switch blockType {
	case "text":
		return r.measureTextBlock(block, ctx)
	case "thinking":
		return r.measureThinkingBlock(block, ctx)
	case "tool_use":
		// Tool use is handled separately
		return 1
	default:
		return 1
	}
}

// renderTextBlock renders a text block.
func (r *AssistantMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)

	// Check for error messages
	if isRateLimitError(text) {
		return r.renderRateLimitError(text, ctx)
	}
	if isApiError(text) {
		return r.renderApiError(text, ctx)
	}

	// Regular assistant text
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme)

	// Add dot if needed
	if ctx.ShouldShowDot && len(lines) > 0 {
		// TODO: Add black circle prefix like TS
	}

	return lines, nil
}

// measureTextBlock measures a text block.
func (r *AssistantMessageRenderer) measureTextBlock(block map[string]interface{}, ctx *RenderContext) int {
	text, _ := block["text"].(string)
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme)
	return len(lines)
}

// renderThinkingBlock renders a thinking block.
func (r *AssistantMessageRenderer) renderThinkingBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// TODO: Implement thinking block rendering
	// Similar to TS AssistantThinkingMessage
	return []string{"[Thinking...]"}, nil
}

// measureThinkingBlock measures a thinking block.
func (r *AssistantMessageRenderer) measureThinkingBlock(block map[string]interface{}, ctx *RenderContext) int {
	// Thinking blocks are usually 1 line
	return 1
}

// renderRateLimitError renders a rate limit error.
func (r *AssistantMessageRenderer) renderRateLimitError(text string, ctx *RenderContext) ([]string, error) {
	// TODO: Implement rate limit error rendering
	// Similar to TS RateLimitMessage
	return []string{"[Rate limit error]"}, nil
}

// renderApiError renders an API error.
func (r *AssistantMessageRenderer) renderApiError(text string, ctx *RenderContext) ([]string, error) {
	// TODO: Implement API error rendering
	return []string{"[API error]"}, nil
}

// Helper functions for error detection

func isRateLimitError(text string) bool {
	// TODO: Implement proper rate limit error detection
	// Similar to TS isRateLimitErrorMessage
	return strings.Contains(text, "rate limit") || strings.Contains(text, "Rate limit")
}

func isApiError(text string) bool {
	// TODO: Implement proper API error detection
	// Check for common API error prefixes
	errorPrefixes := []string{
		"Invalid API key",
		"API key",
		"Authentication",
		"Context limit",
		"Prompt too long",
	}

	for _, prefix := range errorPrefixes {
		if strings.Contains(text, prefix) {
			return true
		}
	}
	return false
}