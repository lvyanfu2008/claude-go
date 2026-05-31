package compo

import (
	"time"

	"goc/gou/ink"
)

var spinnerFrames = []string{"●", "○", "◌", "○"}

func Spinner(ctx *ink.Context, isLoading bool, tip string) ink.VNode {
	if !isLoading {
		return ink.VNode{Type: "Text"}
	}
	frame := spinnerFrames[(time.Now().UnixMilli()/200)%int64(len(spinnerFrames))]
	return ink.VNode{
		Type: "Box", Key: "spinner",
		Props: ink.Props{"direction": "row"},
		Children: []ink.VNode{
			{Type: "Text", Props: ink.Props{"content": frame, "color": ctx.Theme.ToolUse}},
			{Type: "Text", Props: ink.Props{"content": " " + tip, "dim": true}},
		},
	}
}
