package message

import (
	"encoding/json"
	"fmt"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/gou/messagerow"
	"goc/types"
)

// ToolUseMessageRenderer renders tool use messages.
// Similar to TS AssistantToolUseMessage component.
type ToolUseMessageRenderer struct{}

// CanRender returns true for tool use blocks within assistant messages.
func (r *ToolUseMessageRenderer) CanRender(msg *types.Message) bool {
	// This renderer handles tool_use blocks, not whole messages
	// The dispatcher should route tool_use blocks here
	return false // Not used directly, blocks are routed separately
}

// Render is not used for this renderer.
func (r *ToolUseMessageRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	return nil, fmt.Errorf("ToolUseMessageRenderer should not render whole messages")
}

// Measure is not used for this renderer.
func (r *ToolUseMessageRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	return 0, fmt.Errorf("ToolUseMessageRenderer should not measure whole messages")
}

// RenderToolUseBlock renders a single tool_use block.
func (r *ToolUseMessageRenderer) RenderToolUseBlock(block map[string]interface{}, ctx *RenderContext, isInProgress bool) ([]string, error) {
	id, _ := block["id"].(string)
	name, _ := block["name"].(string)
	inputRaw, _ := block["input"].(map[string]interface{})

	diaglog.Line("[tool-use] RenderToolUseBlock: id=%s, name=%s, isInProgress=%v, isTranscript=%v, verbose=%v",
		id, name, isInProgress, ctx.IsTranscript, ctx.Verbose)

	// Convert input to JSON
	inputJSON, err := json.Marshal(inputRaw)
	if err != nil {
		diaglog.Line("[tool-use] Error marshaling input: %v", err)
		return []string{fmt.Sprintf("[Tool use error: %v]", err)}, nil
	}

	// Get tool chrome parts
	facing, paren, hint := messagerow.ToolChromeParts(name, inputJSON)
	diaglog.Line("[tool-use] Tool chrome: facing=%s, paren=%s, hint=%s", facing, paren, hint)

	var lines []string
	width := getContainerWidth(ctx)

	// Build the tool use line
	var lineBuilder strings.Builder

	// Add loader or status indicator
	if isInProgress {
		lineBuilder.WriteString("⟳ ") // Loading symbol
		diaglog.Line("[tool-use] Showing loading symbol (in progress)")
	} else {
		lineBuilder.WriteString("✓ ") // Completed symbol
		diaglog.Line("[tool-use] Showing completed symbol")
	}

	// Add tool name
	lineBuilder.WriteString(facing)

	// Add parenthetical detail if available
	if paren != "" {
		lineBuilder.WriteString(" (")
		lineBuilder.WriteString(paren)
		lineBuilder.WriteString(")")
	}

	// Truncate if too long
	line := lineBuilder.String()
	if len(line) > width && width > 10 {
		line = line[:width-3] + "..."
	}

	lines = append(lines, line)

	// Add hint line if in progress and hint available
	if isInProgress && hint != "" && ctx.ShouldAnimate {
		hintLine := fmt.Sprintf("  ⎿ %s", hint)
		if len(hintLine) > width && width > 10 {
			hintLine = hintLine[:width-3] + "..."
		}
		lines = append(lines, hintLine)
	}

	return lines, nil
}

// MeasureToolUseBlock measures a tool_use block.
func (r *ToolUseMessageRenderer) MeasureToolUseBlock(block map[string]interface{}, ctx *RenderContext, isInProgress bool) int {
	// Tool use blocks are 1 line when completed, 2 lines when in progress with hint
	if isInProgress && ctx.ShouldAnimate {
		return 2
	}
	return 1
}

// RenderToolResultBlock renders a tool_result block.
func (r *ToolUseMessageRenderer) RenderToolResultBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// Similar to TS tool result display
	toolUseID, _ := block["tool_use_id"].(string)
	// content can be either a string or []interface{}
	var contentItems []interface{}

	diaglog.Line("[tool-use] RenderToolResultBlock: tool_use_id=%s, isTranscript=%v, verbose=%v",
		toolUseID, ctx.IsTranscript, ctx.Verbose)

	if diffLines, ok := writeEditDiffLinesFromToolResultBlock(block); ok {
		if ctx.Theme != nil {
			diffLines = ApplyUnifiedDiffLineStyles(diffLines, BlockIsToolError(block), ctx.Theme)
		}
		return diffLines, nil
	}

	// In prompt mode, tool results are usually collapsed to 1 line
	if !ctx.IsTranscript && !ctx.Verbose {
		diaglog.Line("[tool-use] RenderToolResultBlock: collapsed to 1 line (prompt mode)")
		// Generate meaningful summary instead of generic "[Result]"
		summary := GenerateToolResultSummary(block)
		// Add (ctrl+o to expand) hint for consistency with other TUI messages
		return []string{"  ↳ " + summary + " (ctrl+o to expand)"}, nil
	}

	if contentStr, ok := block["content"].(string); ok {
		// content is a string - wrap it as a text block
		diaglog.Line("[tool-use] tool_result content is string, length=%d", len(contentStr))
		if contentStr != "" {
			contentItems = []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": contentStr,
				},
			}
		}
	} else if contentArr, ok := block["content"].([]interface{}); ok {
		diaglog.Line("[tool-use] tool_result content is array, length=%d", len(contentArr))
		contentItems = contentArr
	} else {
		diaglog.Line("[tool-use] tool_result content is unknown type: %T", block["content"])
	}

	if len(contentItems) == 0 {
		diaglog.Line("[tool-use] tool_result empty content")
		return []string{"  ↳ [Empty result]"}, nil
	}

	var lines []string
	for _, item := range contentItems {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); text != "" {
					// Render text content
					textLines := renderMarkdown(text, getContainerWidth(ctx)-2, ctx.Theme, ctx.Highlighter)
					for _, tl := range textLines {
						lines = append(lines, "  "+tl)
					}
				}
			}
		}
	}

	if len(lines) == 0 {
		return []string{"  ↳ [Result]"}, nil
	}

	return lines, nil
}

// MeasureToolResultBlock measures a tool_result block.
func (r *ToolUseMessageRenderer) MeasureToolResultBlock(block map[string]interface{}, ctx *RenderContext) int {
	if diffLines, ok := writeEditDiffLinesFromToolResultBlock(block); ok {
		return len(diffLines)
	}
	// In prompt mode, tool results are usually collapsed
	if !ctx.IsTranscript && !ctx.Verbose {
		diaglog.Line("[tool-use] MeasureToolResultBlock: collapsed to 1 line (prompt mode)")
		return 1
	}
	diaglog.Line("[tool-use] MeasureToolResultBlock: showing full content (transcript=%v, verbose=%v)", ctx.IsTranscript, ctx.Verbose)

	// Calculate actual height
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
					textLines := renderMarkdown(text, getContainerWidth(ctx)-2, ctx.Theme, ctx.Highlighter)
					totalLines += len(textLines)
				}
			}
		}
	}

	if totalLines == 0 {
		return 1
	}

	return totalLines
}
