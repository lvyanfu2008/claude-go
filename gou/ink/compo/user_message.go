package compo

import (
	"fmt"

	"goc/gou/ink"
)

// UserMessage renders a user message with its content blocks.
func UserMessage(ctx *ink.Context, msg ink.Message) ink.VNode {
	var children []ink.VNode
	for _, block := range msg.ContentBlocks {
		switch block.Type {
		case "text":
			children = append(children, userTextContent(ctx, block))
		case "tool_result":
			children = append(children, userToolResult(ctx, block))
		}
	}
	if len(children) == 0 {
		return ink.VNode{Type: "Text"}
	}
	return ink.VNode{
		Type: "Box", Key: msg.UUID,
		Props:    ink.Props{"direction": "column"},
		Children: children,
	}
}

func userTextContent(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	prefix := ink.VNode{
		Type: "Text",
		Props: ink.Props{
			"content": fmt.Sprintf("%-5s", "⏺"),
			"color":   ctx.Theme.User,
			"bold":    true,
		},
	}
	return Row(1, prefix, Markdown(ctx, block.Content, ctx.Store.Width-5))
}

func userToolResult(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	prefix := "  ⎿ "
	switch block.State {
	case "error":
		return Row(1,
			ink.VNode{Type: "Text", Props: ink.Props{"content": prefix + "✗ Error:", "color": ctx.Theme.ToolError}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": block.Content, "dim": true}},
		)
	case "rejected":
		return ink.VNode{Type: "Text", Props: ink.Props{"content": prefix + "Permission denied", "dim": true}}
	default:
		return Row(1,
			ink.VNode{Type: "Text", Props: ink.Props{"content": prefix, "dim": true}},
			Markdown(ctx, block.Content, ctx.Store.Width-6),
		)
	}
}
