package compo

import (
	"fmt"

	"goc/gou/ink"
)

// GroupedToolUse renders a grouped tool use message.
func GroupedToolUse(ctx *ink.Context, msg ink.Message) ink.VNode {
	count := len(msg.ContentBlocks)
	first := ""
	if count > 0 {
		first = msg.ContentBlocks[0].Name
	}
	return ink.VNode{
		Type: "Text", Key: msg.UUID,
		Props: ink.Props{
			"content": fmt.Sprintf("Grouped: %s ×%d", first, count),
			"color":   ctx.Theme.Grouped,
		},
	}
}
