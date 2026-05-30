package compo

import (
	"fmt"
	"time"

	"goc/gou/ink"
	"goc/gou/messagerow"
)

const BLACK_CIRCLE = "●"

// AssistantToolUse renders tool use with TS-accurate states:
//
//	Queued:   static ● (dimmed)
//	Running:  ● blinking at 600ms (dimmed)
//	Resolved: solid ● (green/success, not dimmed)
//	Error:    solid ● (red/error, not dimmed)
func AssistantToolUse(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	activity := messagerow.ActivityLineForToolUse(block.Name, nil)
	if activity == "" {
		activity = block.Name
	}

	var dot string
	color := ctx.Theme.ToolUse
	dimmed := false

	switch block.State {
	case "queued":
		dot = BLACK_CIRCLE
		dimmed = true
	case "running":
		dot = blinkDot(BLACK_CIRCLE, 600*time.Millisecond)
		dimmed = true
	case "resolved":
		dot = BLACK_CIRCLE
	case "error":
		dot = BLACK_CIRCLE
		color = ctx.Theme.ToolError
	default:
		dot = BLACK_CIRCLE
		dimmed = true
	}

	activityText := activity
	if block.State == "running" || block.State == "queued" {
		activityText += "…"
	}

	children := []ink.VNode{
		Row(1,
			ink.VNode{Type: "Text", Props: ink.Props{"content": dot, "color": color, "dim": dimmed}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": activityText, "color": color, "dim": dimmed}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": messagerow.CtrlOToExpandHint, "dim": true}},
		),
	}

	if block.State == "resolved" && block.Result != nil {
		children = append(children, userToolResult(ctx, *block.Result))
	}

	return ink.VNode{
		Type: "Box", Key: fmt.Sprintf("tool-%s", block.Name),
		Props:    ink.Props{"direction": "column"},
		Children: children,
	}
}

func blinkDot(dot string, period time.Duration) string {
	t := time.Now().UnixMilli()
	if (t%int64(period)) < int64(period)/2 {
		return dot
	}
	return " "
}
