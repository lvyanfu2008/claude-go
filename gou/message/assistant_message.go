package message

import (
	"fmt"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/types"
)

// AssistantMessageRenderer renders assistant messages.
type AssistantMessageRenderer struct{
	toolUseRenderer *ToolUseMessageRenderer
}

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

	diaglog.Line("[assistant-message] renderContentBlock: type=%s, index=%d/%d, isTranscript=%v, verbose=%v",
		blockType, index, total, ctx.IsTranscript, ctx.Verbose)

	switch blockType {
	case "text":
		return r.renderTextBlock(block, ctx)
	case "thinking":
		return r.renderThinkingBlock(block, ctx)
	case "tool_use":
		// Tool use blocks are handled by ToolUseMessageRenderer
		if r.toolUseRenderer == nil {
			r.toolUseRenderer = &ToolUseMessageRenderer{}
		}
		// Check if this tool use is in progress (streaming)
		isInProgress := false // TODO: Determine if tool use is in progress
		diaglog.Line("[assistant-message] rendering tool_use block, isInProgress=%v", isInProgress)
		return r.toolUseRenderer.RenderToolUseBlock(block, ctx, isInProgress)
	default:
		diaglog.Line("[assistant-message] unknown block type: %s", blockType)
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
		if r.toolUseRenderer == nil {
			r.toolUseRenderer = &ToolUseMessageRenderer{}
		}
		isInProgress := false // TODO: Determine if tool use is in progress
		return r.toolUseRenderer.MeasureToolUseBlock(block, ctx, isInProgress)
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
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
	
	// Add "⏺ " prefix to assistant messages and indent all lines by 2 spaces
	for i, line := range lines {
		if i == 0 {
			lines[i] = "  ⏺ " + line
		} else {
			lines[i] = "    " + line
		}
	}
	
	return lines, nil
}

// measureTextBlock measures a text block.
func (r *AssistantMessageRenderer) measureTextBlock(block map[string]interface{}, ctx *RenderContext) int {
	text, _ := block["text"].(string)
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
	return len(lines)
}

// renderThinkingBlock renders a thinking block.
func (r *AssistantMessageRenderer) renderThinkingBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// Similar to TS AssistantThinkingMessage
	// Thinking blocks show a simple indicator
	thinkingText := "[Thinking...]"

	// In verbose mode or transcript, we might show more detail
	if ctx.Verbose || ctx.IsTranscript {
		if text, ok := block["text"].(string); ok && text != "" {
			// Show the actual thinking text
			lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
			// Add thinking prefix to first line
			if len(lines) > 0 {
				lines[0] = "💭 " + lines[0]
			}
			return lines, nil
		}
	}

	return []string{"💭 " + thinkingText}, nil
}

// measureThinkingBlock measures a thinking block.
func (r *AssistantMessageRenderer) measureThinkingBlock(block map[string]interface{}, ctx *RenderContext) int {
	// Thinking blocks are usually 1 line in normal mode
	if !ctx.Verbose && !ctx.IsTranscript {
		return 1
	}

	// In verbose mode or transcript, measure actual content
	if text, ok := block["text"].(string); ok && text != "" {
		lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
		return len(lines)
	}

	return 1
}

// renderRateLimitError renders a rate limit error.
func (r *AssistantMessageRenderer) renderRateLimitError(text string, ctx *RenderContext) ([]string, error) {
	// Similar to TS RateLimitMessage
	return []string{"⏳ Rate limit exceeded. Please wait and try again."}, nil
}

// renderApiError renders an API error.
func (r *AssistantMessageRenderer) renderApiError(text string, ctx *RenderContext) ([]string, error) {
	// Extract error message from text
	errorMsg := "API error"
	if len(text) > 100 {
		errorMsg = text[:100] + "..."
	} else if text != "" {
		errorMsg = text
	}
	return []string{"⚠ " + errorMsg}, nil
}

// Helper functions for error detection

func isRateLimitError(text string) bool {
	// Similar to TS isRateLimitErrorMessage
	return strings.Contains(text, "rate limit") || strings.Contains(text, "Rate limit")
}

func isApiError(text string) bool {
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