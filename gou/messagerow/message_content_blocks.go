// Helpers to decode assistant/user content blocks from top-level Content or nested message.content
// (after NormalizeMessageJSON). Prefer scanning for tool_use / tool_result instead of assuming blocks[0].
package messagerow

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// MessageContentBlocks returns content blocks from msg after normalizing embedded JSON (TS message.content).
func MessageContentBlocks(msg types.Message) []types.MessageContentBlock {
	msg = NormalizeMessageJSON(msg)
	raw := msg.Content
	if len(raw) == 0 && len(msg.Message) > 0 {
		var env struct {
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(msg.Message, &env) == nil && len(env.Content) > 0 {
			raw = env.Content
		}
	}
	if len(raw) == 0 {
		return nil
	}
	var blocks []types.MessageContentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return nil
	}
	return blocks
}

// FirstToolUseBlock returns the first tool_use block with name and id (scan order).
func FirstToolUseBlock(msg types.Message) (types.MessageContentBlock, bool) {
	for _, b := range MessageContentBlocks(msg) {
		if b.Type != "tool_use" {
			continue
		}
		if strings.TrimSpace(b.Name) == "" || strings.TrimSpace(b.ID) == "" {
			continue
		}
		return b, true
	}
	return types.MessageContentBlock{}, false
}

// AssistantHasToolUse reports whether any tool_use block exists (with name and id).
func AssistantHasToolUse(msg types.Message) bool {
	_, ok := FirstToolUseBlock(msg)
	return ok
}

// AssistantHasNonEmptyText reports whether any text block has non-whitespace content.
func AssistantHasNonEmptyText(msg types.Message) bool {
	for _, b := range MessageContentBlocks(msg) {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

// FirstNonEmptyTextBlock returns the first text block with non-empty trimmed body.
func FirstNonEmptyTextBlock(msg types.Message) (types.MessageContentBlock, bool) {
	for _, b := range MessageContentBlocks(msg) {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return b, true
		}
	}
	return types.MessageContentBlock{}, false
}

// CountToolUseBlocks counts tool_use blocks with non-empty id (non-grouped assistant turns).
func CountToolUseBlocks(msg types.Message) int {
	n := 0
	for _, b := range MessageContentBlocks(msg) {
		if b.Type == "tool_use" && strings.TrimSpace(b.ID) != "" {
			n++
		}
	}
	return n
}

// AllToolUseIDsFromAssistant collects tool_use ids in block order.
func AllToolUseIDsFromAssistant(msg types.Message) []string {
	var ids []string
	for _, b := range MessageContentBlocks(msg) {
		if b.Type == "tool_use" {
			id := strings.TrimSpace(b.ID)
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// assistantSingleToolUse returns the sole tool_use block when exactly one valid tool_use exists in content.
func assistantSingleToolUse(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := MessageContentBlocks(msg)
	var found *types.MessageContentBlock
	n := 0
	for i := range blocks {
		b := blocks[i]
		if b.Type != "tool_use" {
			continue
		}
		if strings.TrimSpace(b.Name) == "" || strings.TrimSpace(b.ID) == "" {
			continue
		}
		n++
		if found == nil {
			found = &blocks[i]
		}
	}
	if n != 1 || found == nil {
		return types.MessageContentBlock{}, false
	}
	return *found, true
}

// userSingleToolResult returns the sole tool_result block when exactly one valid tool_result exists in content.
func userSingleToolResult(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := MessageContentBlocks(msg)
	var found *types.MessageContentBlock
	n := 0
	for i := range blocks {
		b := blocks[i]
		if b.Type != "tool_result" {
			continue
		}
		if strings.TrimSpace(b.ToolUseID) == "" {
			continue
		}
		n++
		if found == nil {
			found = &blocks[i]
		}
	}
	if n != 1 || found == nil {
		return types.MessageContentBlock{}, false
	}
	return *found, true
}
