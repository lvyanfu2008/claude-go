package compo

import "goc/gou/ink"

// SystemMessage renders a system message with its content blocks.
func SystemMessage(ctx *ink.Context, msg ink.Message) ink.VNode {
	for _, block := range msg.ContentBlocks {
		switch block.Type {
		case "api_error":
			return ink.VNode{
				Type: "Text", Key: msg.UUID,
				Props: ink.Props{"content": "✗ " + block.Content, "color": ctx.Theme.ToolError},
			}
		case "stop_hook_summary":
			return ink.VNode{
				Type: "Text", Key: msg.UUID,
				Props: ink.Props{"content": "🨝 " + block.Content, "dim": true},
			}
		case "compact_boundary":
			return ink.VNode{
				Type: "Text", Key: msg.UUID,
				Props: ink.Props{"content": "─── Conversation compacted ───", "dim": true},
			}
		default:
			return ink.VNode{
				Type: "Text", Key: msg.UUID,
				Props: ink.Props{"content": "ℹ " + block.Content, "dim": true},
			}
		}
	}
	return ink.VNode{Type: "Text"}
}
