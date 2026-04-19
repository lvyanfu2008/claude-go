package message

import (
	"fmt"
	"math/rand"
	"time"

	"goc/types"
)

// Processor handles message processing (collapsing, grouping, etc.).
type Processor struct {
	// Configuration
	CollapseReadSearch bool
	GroupToolUses      bool
	CollapseDelay      time.Duration // Delay before collapsing
}

// NewProcessor creates a new message processor.
func NewProcessor() *Processor {
	return &Processor{
		CollapseReadSearch: true,
		GroupToolUses:      true,
		CollapseDelay:      2 * time.Second,
	}
}

// Process processes messages for display.
func (p *Processor) Process(messages []*types.Message, isTranscript bool) []*types.Message {
	if len(messages) == 0 {
		return messages
	}

	processed := make([]*types.Message, len(messages))
	copy(processed, messages)

	// Apply transformations
	if p.GroupToolUses {
		processed = p.groupToolUses(processed)
	}

	if p.CollapseReadSearch && !isTranscript {
		processed = p.collapseReadSearch(processed)
	}

	return processed
}

// groupToolUses groups consecutive tool uses by tool name.
func (p *Processor) groupToolUses(messages []*types.Message) []*types.Message {
	var result []*types.Message
	var currentGroup []*types.Message
	var currentToolName string

	for _, msg := range messages {
		toolName := p.extractToolName(msg)
		if toolName == "" {
			// Not a tool use, flush current group if any
			result = append(result, p.flushToolGroup(currentGroup, currentToolName)...)
			currentGroup = nil
			currentToolName = ""
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
			result = append(result, p.flushToolGroup(currentGroup, currentToolName)...)
			currentToolName = toolName
			currentGroup = []*types.Message{msg}
		}
	}

	// Flush any remaining group
	result = append(result, p.flushToolGroup(currentGroup, currentToolName)...)

	return result
}

// flushToolGroup flushes a tool group, creating a grouped message if needed.
func (p *Processor) flushToolGroup(group []*types.Message, toolName string) []*types.Message {
	if len(group) == 0 {
		return nil
	}

	if len(group) == 1 {
		return group
	}

	// Convert []*types.Message to []types.Message
	var groupSlice []types.Message
	for _, msg := range group {
		groupSlice = append(groupSlice, *msg)
	}

	// Create grouped tool use message
	grouped := &types.Message{
		Type:     types.MessageTypeGroupedToolUse,
		UUID:     generateGroupUUID(),
		ToolName: toolName,
		Messages: groupSlice,
	}

	return []*types.Message{grouped}
}

// collapseReadSearch collapses consecutive read/search operations.
func (p *Processor) collapseReadSearch(messages []*types.Message) []*types.Message {
	var result []*types.Message
	var currentBatch []*types.Message

	for _, msg := range messages {
		if p.isReadSearchOperation(msg) {
			// Check if we should start a new batch
			if len(currentBatch) == 0 {
				// Start new batch
				currentBatch = []*types.Message{msg}
				// Timestamp extraction would go here
				// batchStartTime = extractTimestamp(msg)
			} else {
				// Check if this message belongs to current batch
				// Based on timing and operation type
				// currentTime := extractTimestamp(msg)
				// if batchStartTime != nil && currentTime.Sub(*batchStartTime) < p.CollapseDelay {
				currentBatch = append(currentBatch, msg)
				// } else {
				//     // Time gap, flush current batch and start new one
				//     result = append(result, p.flushCollapseBatch(currentBatch)...)
				//     currentBatch = []*types.Message{msg}
				//     batchStartTime = &currentTime
				// }
			}
		} else {
			// Not a read/search operation, flush current batch if any
			result = append(result, p.flushCollapseBatch(currentBatch)...)
			currentBatch = nil
			result = append(result, msg)
		}
	}

	// Flush any remaining batch
	result = append(result, p.flushCollapseBatch(currentBatch)...)

	return result
}

// flushCollapseBatch flushes a collapse batch, creating a collapsed group if needed.
func (p *Processor) flushCollapseBatch(batch []*types.Message) []*types.Message {
	if len(batch) == 0 {
		return nil
	}

	if len(batch) == 1 {
		return batch
	}

	// Create collapsed group
	collapsed := p.createCollapsedGroup(batch)
	if collapsed == nil {
		return batch
	}

	return []*types.Message{collapsed}
}

// createCollapsedGroup creates a collapsed read/search group.
func (p *Processor) createCollapsedGroup(messages []*types.Message) *types.Message {
	if len(messages) == 0 {
		return nil
	}

	group := &types.Message{
		Type: types.MessageTypeCollapsedReadSearch,
		UUID: generateGroupUUID(),
	}

	// Count operations
	for _, msg := range messages {
		p.countOperation(msg, group)
	}

	// Set display message (usually the first message)
	if len(messages) > 0 {
		// Convert []*types.Message to []types.Message
		var msgSlice []types.Message
		for _, msg := range messages {
			msgSlice = append(msgSlice, *msg)
		}
		group.Messages = msgSlice
	}

	return group
}

// countOperation counts an operation in the collapsed group.
func (p *Processor) countOperation(msg *types.Message, group *types.Message) {
	// Parse message content to determine operation type
	// Using simple heuristics for now

	// Check message content for operation hints
	content := string(msg.Content)
	if len(content) == 0 && msg.Message != nil {
		content = string(msg.Message)
	}

	// Simple string matching (in production, parse JSON)
	if contains(content, "Read") || contains(content, "file_path") {
		group.ReadCount++
	} else if contains(content, "Grep") || contains(content, "Glob") || contains(content, "pattern") {
		group.SearchCount++
	} else if contains(content, "Bash") || contains(content, "command") {
		if group.BashCount == nil {
			group.BashCount = new(int)
		}
		*group.BashCount++
	}
	// More operation types can be added here
}

// extractToolName extracts the tool name from a message.
func (p *Processor) extractToolName(msg *types.Message) string {
	if msg.Type != types.MessageTypeAssistant {
		return ""
	}

	// Parse message content to extract tool name
	// Using simple string matching for now
	content := string(msg.Content)
	if len(content) == 0 && msg.Message != nil {
		content = string(msg.Message)
	}

	// Simple string matching (in production, parse JSON)
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

// isReadSearchOperation checks if a message is a read/search operation.
func (p *Processor) isReadSearchOperation(msg *types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}

	toolName := p.extractToolName(msg)
	switch toolName {
	case "Read", "Grep", "Glob":
		return true
	default:
		return false
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}

func generateGroupUUID() string {
	// Generate a simple UUID-like string
	return fmt.Sprintf("group-%x-%x-%x-%x",
		time.Now().UnixNano(),
		rand.Int63(),
		rand.Int63(),
		rand.Int63())
}