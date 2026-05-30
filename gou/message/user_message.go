package message

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/types"

	"charm.land/lipgloss/v2"
)

// UserMessageRenderer renders user messages.
type UserMessageRenderer struct{}

const transcriptToolResultMaxLines = 10

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
		diaglog.Line("[user-message] Parse error: %v", err)
		return []string{fmt.Sprintf("Error parsing user message: %v", err)}, nil
	}

	diaglog.Line("[user-message] Parsed %d content blocks, type=%s", len(content), msg.Type)

	var lines []string
	for i, block := range content {
		blockType, _ := block["type"].(string)
		diaglog.Line("[user-message] Rendering block %d: type=%s", i, blockType)
		blockLines, err := r.renderContentBlock(block, ctx, i, len(content))
		if err != nil {
			diaglog.Line("[user-message] Error rendering block %d: %v", i, err)
			return []string{fmt.Sprintf("Error rendering block: %v", err)}, nil
		}
		lines = append(lines, blockLines...)
	}

	diaglog.Line("[user-message] Total rendered lines: %d", len(lines))
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
	// Similar to TS CompactSummary component
	// Extract summary text from message
	summary := "[Previous messages summarized]"
	if msg.Content != nil {
		var text string
		if err := json.Unmarshal(msg.Content, &text); err == nil && text != "" {
			if len(text) > 50 {
				summary = text[:50] + "..."
			} else {
				summary = text
			}
		}
	}
	return []string{"📋 " + summary}, nil
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
	if strings.Contains(text, "<command-message>") {
		return r.renderCommandMessage(text, ctx)
	}

	// Regular user prompt
	containerWidth := getContainerWidth(ctx)
	lines := renderMarkdown(text, containerWidth, ctx.Theme, ctx.Highlighter)
	return r.styleUserLines(lines, ctx), nil
}

// measureTextBlock measures a text block.
func (r *UserMessageRenderer) measureTextBlock(block map[string]interface{}, ctx *RenderContext) int {
	text, _ := block["text"].(string)

	// Special messages are usually 1 line
	if strings.Contains(text, "<bash-input>") ||
		strings.Contains(text, "<bash-stdout") ||
		strings.Contains(text, "<bash-stderr") ||
		strings.Contains(text, "<local-command-stdout") ||
		strings.Contains(text, "<local-command-stderr") ||
		strings.Contains(text, "<command-message>") {
		return 1
	}

	// Regular text
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
	return len(lines)
}

// renderBashInput renders bash input.
func (r *UserMessageRenderer) renderBashInput(text string, ctx *RenderContext) ([]string, error) {
	// Extract command from XML-like tag
	// Basic XML-like parsing
	cmdStart := strings.Index(text, ">")
	cmdEnd := strings.LastIndex(text, "<")
	cmd := "[bash command]"
	if cmdStart > 0 && cmdEnd > cmdStart {
		cmd = text[cmdStart+1 : cmdEnd]
	}

	// Create lipgloss style for user messages: gray background fills entire row
	containerWidth := getContainerWidth(ctx)
	userStyle := lipgloss.NewStyle().
		Background(ctx.Theme.UserMessageBackground).
		Foreground(ctx.Theme.UserMessageText).
		Bold(true).
		Width(containerWidth)

	// Apply styling to entire line including prefix
	fullLine := "❯ $ " + cmd
	styledLine := userStyle.Render(fullLine)
	return []string{styledLine}, nil
}

// renderBashOutput renders bash output.
func (r *UserMessageRenderer) renderBashOutput(text string, ctx *RenderContext) ([]string, error) {
	// Extract output from XML-like tag
	output := text
	outputStart := strings.Index(text, ">")
	outputEnd := strings.LastIndex(text, "<")
	if outputStart > 0 && outputEnd > outputStart {
		output = text[outputStart+1 : outputEnd]
	}
	// Truncate long output
	if len(output) > 100 {
		output = output[:100] + "..."
	}
	return []string{"  " + output}, nil
}

// renderLocalCommandOutput renders local command output.
func (r *UserMessageRenderer) renderLocalCommandOutput(text string, ctx *RenderContext) ([]string, error) {
	// Similar to bash output rendering
	output := text
	outputStart := strings.Index(text, ">")
	outputEnd := strings.LastIndex(text, "<")
	if outputStart > 0 && outputEnd > outputStart {
		output = text[outputStart+1 : outputEnd]
	}
	if len(output) > 100 {
		output = output[:100] + "..."
	}
	return []string{"  " + output}, nil
}

// renderCommandMessage renders the visible metadata row for a slash command or skill invocation.
// TS reference: UserCommandMessage renders "❯ /command args" or "Skill(name)".
func (r *UserMessageRenderer) renderCommandMessage(text string, ctx *RenderContext) ([]string, error) {
	cmdName := extractXMLTagValue(text, "command-message")
	if cmdName == "" {
		cmdName = extractXMLTagValue(text, "command-name")
		cmdName = strings.TrimPrefix(cmdName, "/")
	}
	if cmdName == "" {
		// Fallback: treat as regular text
		containerWidth := getContainerWidth(ctx)
		lines := renderMarkdown(text, containerWidth, ctx.Theme, ctx.Highlighter)
		return r.styleUserLines(lines, ctx), nil
	}

	isSkillFormat := extractXMLTagValue(text, "skill-format") == "true"
	args := extractXMLTagValue(text, "command-args")

	display := "/" + cmdName
	if args != "" {
		display += " " + args
	}
	if isSkillFormat {
		display = "Skill(" + cmdName + ")"
	}

	containerWidth := getContainerWidth(ctx)
	userStyle := lipgloss.NewStyle().
		Background(ctx.Theme.UserMessageBackground).
		Foreground(ctx.Theme.UserMessageText).
		Bold(true).
		Width(containerWidth)
	fullLine := "❯ " + display
	return []string{userStyle.Render(fullLine)}, nil
}

// styleUserLines applies "  > " / "    " prefix + theme to each rendered line.
func (r *UserMessageRenderer) styleUserLines(lines []string, ctx *RenderContext) []string {
	containerWidth := getContainerWidth(ctx)
	userStyle := lipgloss.NewStyle().
		Background(ctx.Theme.UserMessageBackground).
		Foreground(ctx.Theme.UserMessageText).
		Bold(true).
		Width(containerWidth)
	for i, line := range lines {
		if i == 0 {
			line = "❯ " + line
		} else {
			line = "  " + line
		}
		lines[i] = userStyle.Render(line)
	}
	return lines
}

// renderImageBlock renders an image block.
func (r *UserMessageRenderer) renderImageBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// Extract image information
	source, _ := block["source"].(map[string]interface{})
	if source != nil {
		data, _ := source["data"].(string)
		mediaType, _ := source["media_type"].(string)
		if data != "" && mediaType != "" {
			// Show image info
			return []string{fmt.Sprintf("🖼 Image (%s, %d chars)", mediaType, len(data))}, nil
		}
	}
	return []string{"🖼 [Image]"}, nil
}

// renderToolUseBlock renders a tool use block.
func (r *UserMessageRenderer) renderToolUseBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// User messages shouldn't have tool_use blocks normally
	return []string{"[Tool use in user message]"}, nil
}

// renderToolResultBlock renders a tool result block.
func (r *UserMessageRenderer) renderToolResultBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// Similar to TS UserToolResultMessage
	// content can be either a string or []interface{}
	var contentItems []interface{}

	diaglog.Line("[user-message] renderToolResultBlock: isTranscript=%v, verbose=%v", ctx.IsTranscript, ctx.Verbose)

	if diffLines, ok := writeEditDiffLinesFromToolResultBlock(block); ok {
		if ctx.Theme != nil {
			diffLines = ApplyUnifiedDiffLineStyles(diffLines, BlockIsToolError(block), ctx.Theme)
		}
		return diffLines, nil
	}

	// In prompt mode, show tool result content inline (truncated), matching TS.
	if !ctx.IsTranscript && !ctx.Verbose {
		return renderToolResultInline(block, ctx)
	}


	if contentStr, ok := block["content"].(string); ok {
		// content is a string - wrap it as a text block
		diaglog.Line("[user-message] tool_result content is string, length=%d", len(contentStr))
		if contentStr != "" {
			contentItems = []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": contentStr,
				},
			}
		}
	} else if contentArr, ok := block["content"].([]interface{}); ok {
		diaglog.Line("[user-message] tool_result content is array, length=%d", len(contentArr))
		contentItems = contentArr
	} else {
		diaglog.Line("[user-message] tool_result content is unknown type: %T", block["content"])
	}

	if len(contentItems) == 0 {
		diaglog.Line("[user-message] tool_result empty content")
		return []string{"↳ [Empty result]"}, nil
	}

	var lines []string
	for _, item := range contentItems {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); text != "" {
					// Render text content with wrapping instead of markdown to preserve newlines
					textLines := wrapText(text, getContainerWidth(ctx)-2)
					for _, tl := range textLines {
						lines = append(lines, "  "+tl)
					}
				}
			}
		}
	}

	if len(lines) == 0 {
		return []string{"↳ [Result]"}, nil
	}
	if ctx.IsTranscript && len(lines) > transcriptToolResultMaxLines {
		lines = append([]string(nil), lines[:transcriptToolResultMaxLines]...)
		lines[transcriptToolResultMaxLines-1] += " ..."
	}

	return lines, nil
}

// measureToolResultBlock measures a tool result block.
func (r *UserMessageRenderer) measureToolResultBlock(block map[string]interface{}, ctx *RenderContext) int {
	if diffLines, ok := writeEditDiffLinesFromToolResultBlock(block); ok {
		return len(diffLines)
	}
	// Tool results are usually collapsed to 1 line in prompt mode
	if !ctx.IsTranscript && !ctx.Verbose {
		inlineLines, _ := renderToolResultInline(block, ctx)
		return len(inlineLines)
	}
	diaglog.Line("[user-message] measureToolResultBlock: showing full content (transcript=%v, verbose=%v)", ctx.IsTranscript, ctx.Verbose)
	// In transcript or verbose mode, show full content
	// content can be either a string or []interface{}
	var contentItems []interface{}

	if contentStr, ok := block["content"].(string); ok {
		// content is a string - wrap it as a text block
		if contentStr != "" {
			contentItems = []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": contentStr,
				},
			}
		}
	} else if contentArr, ok := block["content"].([]interface{}); ok {
		contentItems = contentArr
	}

	if len(contentItems) == 0 {
		return 1
	}

	totalLines := 0
	for _, item := range contentItems {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); text != "" {
					textLines := wrapText(text, getContainerWidth(ctx)-2)
					totalLines += len(textLines)
				}
			}
		}
	}

	if totalLines == 0 {
		return 1
	}
	if ctx.IsTranscript && totalLines > transcriptToolResultMaxLines {
		return transcriptToolResultMaxLines
	}

	return totalLines
}

// Helper function to parse message content
func parseMessageContent(msg *types.Message) ([]map[string]interface{}, error) {
	var content []map[string]interface{}
	// Try Content field first
	if len(msg.Content) > 0 {
		diaglog.Line("[parseMessageContent] Trying to parse Content field, length=%d", len(msg.Content))
		if err := json.Unmarshal(msg.Content, &content); err == nil {
			diaglog.Line("[parseMessageContent] Successfully parsed %d blocks from Content", len(content))
			return content, nil
		} else {
			diaglog.Line("[parseMessageContent] Failed to parse Content: %v", err)
		}
	}

	// Try Message field
	if len(msg.Message) > 0 {
		diaglog.Line("[parseMessageContent] Trying to parse Message field, length=%d", len(msg.Message))
		var messageObj struct {
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(msg.Message, &messageObj); err == nil && len(messageObj.Content) > 0 {
			// First try to parse as array of blocks
			if err := json.Unmarshal(messageObj.Content, &content); err == nil {
				diaglog.Line("[parseMessageContent] Successfully parsed %d blocks from Message.content", len(content))
				return content, nil
			} else {
				diaglog.Line("[parseMessageContent] Failed to parse Message.content as array: %v", err)
				// Try to parse as string
				var text string
				if err := json.Unmarshal(messageObj.Content, &text); err == nil {
					diaglog.Line("[parseMessageContent] Successfully parsed Message.content as string, length=%d", len(text))
					return []map[string]interface{}{{"type": "text", "text": text}}, nil
				} else {
					diaglog.Line("[parseMessageContent] Failed to parse Message.content as string: %v", err)
				}
			}
		}
	}

	// Fallback: treat as single text block
	if msg.Content != nil {
		diaglog.Line("[parseMessageContent] Trying fallback: parse Content as string")
		var text string
		if err := json.Unmarshal(msg.Content, &text); err == nil {
			diaglog.Line("[parseMessageContent] Fallback successful: text length=%d", len(text))
			return []map[string]interface{}{{"type": "text", "text": text}}, nil
		} else {
			diaglog.Line("[parseMessageContent] Fallback failed: %v", err)
		}
	}

	diaglog.Line("[parseMessageContent] Could not parse message content")
	return nil, fmt.Errorf("could not parse message content")
}


// extractXMLTagValue extracts the text content of the first occurrence of a named XML tag.
// e.g. extractXMLTagValue("<command-name>/init</command-name>", "command-name") returns "/init".
func extractXMLTagValue(text, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	start := strings.Index(text, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(text[start:], close)
	if end < 0 {
		return ""
	}
	return text[start : start+end]
}
