package message

import (
	"encoding/json"
	"fmt"
	"strings"

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
	_, _ = block["id"].(string) // id is unused
	name, _ := block["name"].(string)
	inputRaw, _ := block["input"].(map[string]interface{})

	// Convert input to JSON
	inputJSON, err := json.Marshal(inputRaw)
	if err != nil {
		return []string{fmt.Sprintf("[Tool use error: %v]", err)}, nil
	}

	// Get tool chrome parts
	facing, paren, hint := messagerow.ToolChromeParts(name, inputJSON)

	var lines []string
	width := getContainerWidth(ctx)

	// Build the tool use line
	var lineBuilder strings.Builder

	// Add loader or status indicator
	if isInProgress {
		lineBuilder.WriteString("⟳ ") // Loading symbol
	} else {
		lineBuilder.WriteString("✓ ") // Completed symbol
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
	// TODO: Implement tool result rendering
	// Similar to TS tool result display
	_, _ = block["tool_use_id"].(string) // toolUseID is unused
	content, _ := block["content"].([]interface{})

	if len(content) == 0 {
		return []string{"  ↳ [Empty result]"}, nil
	}

	var lines []string
	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); text != "" {
					// Render text content
					textLines := renderMarkdown(text, getContainerWidth(ctx)-2, ctx.Theme)
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
	// In prompt mode, tool results are usually collapsed
	if !ctx.IsTranscript && !ctx.Verbose {
		return 1
	}

	// Calculate actual height
	content, _ := block["content"].([]interface{})
	if len(content) == 0 {
		return 1
	}

	totalLines := 0
	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); text != "" {
					textLines := renderMarkdown(text, getContainerWidth(ctx)-2, ctx.Theme)
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