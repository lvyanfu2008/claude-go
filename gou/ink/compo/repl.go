package compo

import "goc/gou/ink"

// REPL renders the full REPL layout: messages, status line, and prompt.
func REPL(ctx *ink.Context, p ink.Props) ink.VNode {
	return ink.VNode{
		Type: "Box", Key: "repl",
		Props: ink.Props{"direction": "column"},
		Children: []ink.VNode{
			{
				Type: "ScrollBox", Key: "messages-scroll",
				Props: ink.Props{"stickyBottom": true},
				Children: []ink.VNode{Messages(ctx, p)},
			},
			StatusLine(ctx),
			PromptInput(ctx),
		},
	}
}
