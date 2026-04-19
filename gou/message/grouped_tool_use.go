package message

import (
	"fmt"
	"math/rand"
	"time"

	"goc/types"
)

// GroupedToolUseRenderer renders grouped tool use messages.
// Similar to TS GroupedToolUseContent component.
type GroupedToolUseRenderer struct{}

// CanRender returns true for grouped tool use messages.
func (r *GroupedToolUseRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeGroupedToolUse
}

// Render renders a grouped tool use message.
func (r *GroupedToolUseRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	if msg.ToolName == "" || len(msg.Messages) == 0 {
		return []string{"[Grouped tool use]"}, nil
	}

	var lines []string
	width := getContainerWidth(ctx)

	// Build header line
	header := fmt.Sprintf("%s ×%d", formatToolName(msg.ToolName), len(msg.Messages))
	if len(header) > width && width > 10 {
		header = header[:width-3] + "..."
	}
	lines = append(lines, header)

	// Render individual tool uses if verbose
	if ctx.Verbose || ctx.IsTranscript {
		for i, toolMsg := range msg.Messages {
			// Render individual tool uses
			line := fmt.Sprintf("  %d. %s", i+1, formatToolMessage(&toolMsg))
			if len(line) > width && width > 10 {
				line = line[:width-3] + "..."
			}
			lines = append(lines, line)
		}
	}

	return lines, nil
}

// Measure measures a grouped tool use message.
func (r *GroupedToolUseRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	if msg.ToolName == "" || len(msg.Messages) == 0 {
		return 1, nil
	}

	// Header line
	lines := 1

	// Individual tool uses if verbose
	if ctx.Verbose || ctx.IsTranscript {
		lines += len(msg.Messages)
	}

	return lines, nil
}

// formatToolName formats a tool name for display.
func formatToolName(toolName string) string {
	// Map tool names to user-friendly names
	switch toolName {
	case "Read":
		return "Read"
	case "Grep":
		return "Search"
	case "Glob":
		return "Search"
	case "Bash":
		return "Bash"
	case "Write":
		return "Write"
	case "Edit":
		return "Edit"
	case "WebFetch":
		return "Fetch"
	case "WebSearch":
		return "WebSearch"
	default:
		return toolName
	}
}

// formatToolMessage formats a tool message for display.
func formatToolMessage(msg *types.Message) string {
	// Extract and format tool use information from message
	// Simple implementation for now
	toolName := extractToolName(msg)
	if toolName != "" {
		return fmt.Sprintf("%s tool", toolName)
	}
	return fmt.Sprintf("%s tool use", msg.Type)
}

// ShouldGroupToolUses checks if tool uses should be grouped.
func ShouldGroupToolUses(toolUses []*types.Message) bool {
	if len(toolUses) < 2 {
		return false
	}

	// Check if all tool uses are for the same tool
	firstToolName := extractToolName(toolUses[0])
	if firstToolName == "" {
		return false
	}

	for i := 1; i < len(toolUses); i++ {
		if extractToolName(toolUses[i]) != firstToolName {
			return false
		}
	}

	return true
}

// CreateGroupedToolUse creates a grouped tool use message.
func CreateGroupedToolUse(toolUses []*types.Message, results []*types.Message, groupUUID string) *types.Message {
	if len(toolUses) == 0 {
		return nil
	}

	toolName := extractToolName(toolUses[0])
	if toolName == "" {
		return nil
	}

	// Find a display message (prefer first assistant message)
	var displayMessage *types.Message
	if len(toolUses) > 0 {
		displayMessage = toolUses[0]
	}

	// Convert []*types.Message to []types.Message
	var toolUsesSlice []types.Message
	for _, msg := range toolUses {
		toolUsesSlice = append(toolUsesSlice, *msg)
	}
	var resultsSlice []types.Message
	for _, msg := range results {
		resultsSlice = append(resultsSlice, *msg)
	}

	return &types.Message{
		Type:           types.MessageTypeGroupedToolUse,
		UUID:           groupUUID,
		ToolName:       toolName,
		Messages:       toolUsesSlice,
		Results:        resultsSlice,
		DisplayMessage: displayMessage,
	}
}

// extractToolName extracts the tool name from a message.
func extractToolName(msg *types.Message) string {
	if msg.Type != types.MessageTypeAssistant {
		return ""
	}

	// Parse message content to extract tool name
	content := string(msg.Content)
	if len(content) == 0 && msg.Message != nil {
		content = string(msg.Message)
	}

	// Simple string matching for common tools
	// In production, should parse JSON properly
	if contains(content, `"name":"Read"`) {
		return "Read"
	} else if contains(content, `"name":"Grep"`) {
		return "Grep"
	} else if contains(content, `"name":"Glob"`) {
		return "Glob"
	} else if contains(content, `"name":"Bash"`) {
		return "Bash"
	} else if contains(content, `"name":"Write"`) {
		return "Write"
	} else if contains(content, `"name":"Edit"`) {
		return "Edit"
	} else if contains(content, `"name":"WebFetch"`) {
		return "WebFetch"
	} else if contains(content, `"name":"WebSearch"`) {
		return "WebSearch"
	}

	return ""
}

// GroupConsecutiveToolUses groups consecutive tool uses by tool name.
func GroupConsecutiveToolUses(messages []*types.Message) []*types.Message {
	var result []*types.Message
	var currentGroup []*types.Message
	var currentToolName string

	for _, msg := range messages {
		toolName := extractToolName(msg)
		if toolName == "" {
			// Not a tool use, flush current group if any
			if len(currentGroup) > 0 {
				if len(currentGroup) == 1 {
					result = append(result, currentGroup[0])
				} else {
					group := CreateGroupedToolUse(currentGroup, nil, generateUUID())
					if group != nil {
						result = append(result, group)
					}
				}
				currentGroup = nil
				currentToolName = ""
			}
			result = append(result, msg)
			continue
		}

		if currentToolName == "" {
			// Start new group
			currentToolName = toolName
			currentGroup = []*types.Message{msg}
		} else if toolName == currentToolName {
			// Add to current group
			currentGroup = append(currentGroup, msg)
		} else {
			// Different tool, flush current group and start new one
			if len(currentGroup) == 1 {
				result = append(result, currentGroup[0])
			} else {
				group := CreateGroupedToolUse(currentGroup, nil, generateUUID())
				if group != nil {
					result = append(result, group)
				}
			}
			currentToolName = toolName
			currentGroup = []*types.Message{msg}
		}
	}

	// Flush any remaining group
	if len(currentGroup) > 0 {
		if len(currentGroup) == 1 {
			result = append(result, currentGroup[0])
		} else {
			group := CreateGroupedToolUse(currentGroup, nil, generateUUID())
			if group != nil {
				result = append(result, group)
			}
		}
	}

	return result
}

// Helper function to generate UUID
func generateUUID() string {
	// Generate a simple UUID-like string
	// In production, use github.com/google/uuid or similar
	return fmt.Sprintf("group-%x-%x-%x-%x",
		time.Now().UnixNano(),
		rand.Int63(),
		rand.Int63(),
		rand.Int63())
}