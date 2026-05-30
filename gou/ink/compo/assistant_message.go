package compo

import "goc/gou/ink"

// AssistantMessage renders an assistant message with its content blocks.
func AssistantMessage(ctx *ink.Context, msg ink.Message) ink.VNode {
	var children []ink.VNode
	for _, block := range msg.ContentBlocks {
		switch block.Type {
		case "thinking":
			children = append(children, ink.VNode{
				Type: "Text",
				Props: ink.Props{
					"content": "💭 " + block.Content,
					"dim":     true,
					"italic":  true,
				},
			})
		case "text":
			children = append(children, Markdown(ctx, block.Content, ctx.Store.Width))
		case "tool_use":
			children = append(children, AssistantToolUse(ctx, block))
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
