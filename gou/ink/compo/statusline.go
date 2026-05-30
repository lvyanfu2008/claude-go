package compo

import (
	"fmt"

	"goc/gou/ink"
)

// StatusLine renders the status bar at the bottom of the TUI.
func StatusLine(ctx *ink.Context) ink.VNode {
	model := ctx.Store.GetMeta("model")
	if model == "" {
		model = "claude"
	}
	cwd := ctx.Store.GetMeta("cwd")
	if cwd == "" {
		cwd = "/"
	}
	info := fmt.Sprintf("claude-go  |  %s  |  %s", model, cwd)
	return ink.VNode{
		Type: "Text", Key: "statusline",
		Props: ink.Props{"content": info, "dim": true},
	}
}
