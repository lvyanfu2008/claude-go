// Helpers to decode assistant/user content blocks from top-level Content or nested message.content
// (after NormalizeMessageJSON). Collapse logic follows TS getFirstContentItem: only the first block
// decides collapsible tool_use / text breaker / thinking skip; tail rollup still requires a single
// tool_use with that tool_use as blocks[0].
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

// FirstContentBlock returns blocks[0] when present (TS getFirstContentItem).
func FirstContentBlock(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := MessageContentBlocks(msg)
	if len(blocks) == 0 {
		return types.MessageContentBlock{}, false
	}
	return blocks[0], true
}

// assistantSingleToolUse returns the tool_use when blocks[0] is the only valid tool_use in content
// (TS tail pair: text or thinking before tool_use breaks the suffix).
func assistantSingleToolUse(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := MessageContentBlocks(msg)
	if len(blocks) == 0 {
		return types.MessageContentBlock{}, false
	}
	b0 := blocks[0]
	if b0.Type != "tool_use" || strings.TrimSpace(b0.Name) == "" || strings.TrimSpace(b0.ID) == "" {
		return types.MessageContentBlock{}, false
	}
	n := 0
	for _, b := range blocks {
		if b.Type == "tool_use" && strings.TrimSpace(b.Name) != "" && strings.TrimSpace(b.ID) != "" {
			n++
		}
	}
	if n != 1 {
		return types.MessageContentBlock{}, false
	}
	return b0, true
}

// userSingleToolResult returns the tool_result when blocks[0] is the only valid tool_result in content.
func userSingleToolResult(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := MessageContentBlocks(msg)
	if len(blocks) == 0 {
		return types.MessageContentBlock{}, false
	}
	b0 := blocks[0]
	if b0.Type != "tool_result" || strings.TrimSpace(b0.ToolUseID) == "" {
		return types.MessageContentBlock{}, false
	}
	n := 0
	for _, b := range blocks {
		if b.Type == "tool_result" && strings.TrimSpace(b.ToolUseID) != "" {
			n++
		}
	}
	if n != 1 {
		return types.MessageContentBlock{}, false
	}
	return b0, true
}
