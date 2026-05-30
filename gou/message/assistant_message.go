package message

import (
	"fmt"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/compactservice"
	"goc/types"
)

// maxAPIErrorChars mirrors MAX_API_ERROR_CHARS in AssistantTextMessage.tsx.
const maxAPIErrorChars = 1000

// AssistantMessageRenderer renders assistant messages.
type AssistantMessageRenderer struct {
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
	case "tool_result":
		// Assistant rows often interleave tool_use + tool_result; same chrome as [ToolUseMessageRenderer].
		if r.toolUseRenderer == nil {
			r.toolUseRenderer = &ToolUseMessageRenderer{}
		}
		diaglog.Line("[assistant-message] rendering tool_result block")
		return r.toolUseRenderer.RenderToolResultBlock(block, ctx)
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
	case "tool_result":
		if r.toolUseRenderer == nil {
			r.toolUseRenderer = &ToolUseMessageRenderer{}
		}
		return r.toolUseRenderer.MeasureToolResultBlock(block, ctx)
	default:
		return 1
	}
}

// renderTextBlock renders a text block.
func (r *AssistantMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)
	trimmed := strings.TrimSpace(text)

	// Match AssistantTextMessage.tsx: rate limits via startsWith prefixes, API errors via "API Error" prefix only.
	if compactservice.IsRateLimitErrorMessage(text) {
		return r.renderRateLimitError(text, ctx)
	}
	if compactservice.StartsWithApiErrorPrefix(trimmed) {
		return r.renderApiError(text, ctx)
	}

	// Regular assistant text
	contentWidth := getContainerWidth(ctx) - 3
	if contentWidth < 20 {
		contentWidth = 20
	}
	lines := renderMarkdown(text, contentWidth, ctx.Theme, ctx.Highlighter)

	// Add "⏺ " prefix to assistant messages and indent continuation lines by 2 spaces
	for i, line := range lines {
		if i == 0 {
			lines[i] = "⏺ " + line
		} else {
			lines[i] = "  " + line
		}
	}

	return lines, nil
}

// measureTextBlock measures a text block.
func (r *AssistantMessageRenderer) measureTextBlock(block map[string]interface{}, ctx *RenderContext) int {
	text, _ := block["text"].(string)
	trimmed := strings.TrimSpace(text)
	if compactservice.IsRateLimitErrorMessage(text) || compactservice.StartsWithApiErrorPrefix(trimmed) {
		return 1
	}
	contentWidth := getContainerWidth(ctx) - 3
	if contentWidth < 20 {
		contentWidth = 20
	}
	lines := renderMarkdown(text, contentWidth, ctx.Theme, ctx.Highlighter)
	return len(lines)
}

// renderThinkingBlock renders a thinking block.
func (r *AssistantMessageRenderer) renderThinkingBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	thinkingBody := thinkingBlockString(block)
	// "∴ Thinking" — dim + italic, matching TS AssistantThinkingMessage
	const thinkingLabel = "\x1b[2;3m∴ Thinking\x1b[0m"

	// In verbose mode or transcript, show full thinking content.
	if ctx.Verbose || ctx.IsTranscript {
		if thinkingBody != "" {
			contentWidth := getContainerWidth(ctx) - 2
			if contentWidth < 20 {
				contentWidth = 20
			}
			lines := renderMarkdown(thinkingBody, contentWidth, ctx.Theme, ctx.Highlighter)
			var result []string
			result = append(result, thinkingLabel)
			for _, line := range lines {
				result = append(result, "  " + line)
			}
			return result, nil
		}
		return []string{thinkingLabel}, nil
	}

	// Normal mode: show only the thinking label, no content preview.
	return []string{thinkingLabel}, nil
}
// sentenceEnd checks whether r is a sentence-ending rune.
func sentenceEnd(r rune) bool {
	switch r {
	case '.', '!', '?', '。', '！', '？':
		return true
	}
	return false
}

// FirstSentenceOf returns the first sentence of s. A sentence ends at `.`, `!`,
// `?`, `。`, `！`, `？` followed by a space, newline, or end-of-string.
// If the entire string is one sentence, returns "" to signal "no further text".
// Numbered prefixes like "1." or "12." at the start are not treated as sentence ends.
func FirstSentenceOf(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if sentenceEnd(r) {
			// Check what follows the punctuation.
			if i+1 >= len(runes) {
				// End of string — this is the only sentence.
				return ""
			}
			next := runes[i+1]
			if next == ' ' || next == '\n' || next == '\r' {
				// Skip numbered-list prefixes (e.g. "1.", "12.") — treat them
				// as part of the first sentence, not a sentence boundary.
				if r == '.' && isOnlyDigits(runes[:i]) {
					continue
				}
				return string(runes[:i+1]) + "..."
			}
		}
	}
	return ""
}

// isOnlyDigits reports whether runes consists entirely of ASCII digits.
func isOnlyDigits(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// measureThinkingBlock measures a thinking block.
func (r *AssistantMessageRenderer) measureThinkingBlock(block map[string]interface{}, ctx *RenderContext) int {
	// Thinking blocks are usually 1 line in normal mode
	if !ctx.Verbose && !ctx.IsTranscript {
		return 1
	}

	if thinkingBody := thinkingBlockString(block); thinkingBody != "" {
		contentWidth := getContainerWidth(ctx) - 5
		if contentWidth < 20 {
			contentWidth = 20
		}
		lines := renderMarkdown(thinkingBody, contentWidth, ctx.Theme, ctx.Highlighter)
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
	errorMsg := "API error"
	if text != "" {
		errorMsg = text
	}
	if !ctx.Verbose && len(errorMsg) > maxAPIErrorChars {
		errorMsg = errorMsg[:maxAPIErrorChars] + "…"
	}
	return []string{"⚠ " + errorMsg}, nil
}

// thinkingBlockString returns displayable thinking body (Anthropic uses "thinking", not "text").
func thinkingBlockString(block map[string]interface{}) string {
	if s, ok := block["thinking"].(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	if s, ok := block["text"].(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return ""
}
