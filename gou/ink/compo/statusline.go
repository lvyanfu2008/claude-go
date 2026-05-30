package compo

import "goc/gou/ink"

// StatusLine renders the status bar at the bottom of the TUI.
func StatusLine(ctx *ink.Context) ink.VNode {
	return ink.VNode{
		Type: "Text", Key: "statusline",
		Props: ink.Props{"content": "gou-demo  |  Opus 4.7  |  /home/user  |  12k tokens", "dim": true},
	}
}
