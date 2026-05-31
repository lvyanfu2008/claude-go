package compo

import "goc/gou/ink"

func Spinner(ctx *ink.Context, isLoading bool, tip string) ink.VNode {
	if !isLoading {
		return ink.VNode{Type: "Text"}
	}
	return ink.VNode{
		Type: "Box", Key: "spinner",
		Props: ink.Props{"direction": "row"},
		Children: []ink.VNode{
			{Type: "Text", Props: ink.Props{"content": "...", "color": ctx.Theme.ToolUse}},
			{Type: "Text", Props: ink.Props{"content": " " + tip, "dim": true}},
		},
	}
}
