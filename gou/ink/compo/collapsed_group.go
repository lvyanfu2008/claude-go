package compo

import (
	"time"

	"goc/gou/ink"
	"goc/gou/messagerow"
)

// countSearchReadFromItems computes search/read counts and active state
// from a slice of ink.Message items for collapsed group rendering.
func countSearchReadFromItems(items []ink.Message) (isActive bool, searchCount, readCount int) {
	for _, item := range items {
		for _, block := range item.ContentBlocks {
			if block.Type == "tool_use" {
				switch block.Name {
				case "Read":
					readCount++
					if block.State != "resolved" {
						isActive = true
					}
				case "Grep", "Glob":
					searchCount++
					if block.State != "resolved" {
						isActive = true
					}
				}
			}
		}
	}
	return
}

// CollapsedReadSearch renders a collapsed group of Read/Grep/Glob tool uses.
func CollapsedReadSearch(ctx *ink.Context, msg ink.Message) ink.VNode {
	items, _ := msg.Meta["items"].([]ink.Message)
	allResolved := true
	anyError := false
	for _, item := range items {
		for _, b := range item.ContentBlocks {
			if b.State != "resolved" {
				allResolved = false
			}
			if b.State == "error" {
				anyError = true
			}
		}
	}

	var children []ink.VNode

	// Dot
	var dotNode ink.VNode
	if !allResolved {
		dotColor := ctx.Theme.ToolUse
		if anyError {
			dotColor = ctx.Theme.ToolError
		}
		dotNode = ink.VNode{Type: "Text", Props: ink.Props{
			"content": blinkDot(BLACK_CIRCLE, 600*time.Millisecond),
			"color":   dotColor,
		}}
	} else {
		dotNode = ink.VNode{Type: "Text", Props: ink.Props{"content": "  "}}
	}

	isActive, searchCount, readCount := countSearchReadFromItems(items)
	summary := messagerow.SearchReadSummaryText(isActive, searchCount, readCount, 0, 0, 0, 0, 0, 0, nil, nil, nil)
	dimmed := allResolved

	children = append(children, Row(1,
		dotNode,
		ink.VNode{Type: "Text", Props: ink.Props{"content": summary, "color": ctx.Theme.Collapsed, "dim": dimmed}},
		ink.VNode{Type: "Text", Props: ink.Props{"content": messagerow.CtrlOToExpandHint, "dim": true}},
	))

	if !allResolved {
		hint, _ := msg.Meta["hint"].(string)
		if hint == "" && len(items) > 0 {
			last := items[len(items)-1]
			for _, b := range last.ContentBlocks {
				if b.Type == "tool_use" {
					if fp, ok := b.Input["file_path"]; ok {
						if s, ok := fp.(string); ok {
							hint = s
						}
					} else if pat, ok := b.Input["pattern"]; ok {
						if s, ok := pat.(string); ok {
							hint = s
						}
					}
				}
			}
		}
		if hint != "" {
			children = append(children, ink.VNode{Type: "Text", Props: ink.Props{
				"content": "  ⎿  " + hint, "dim": true,
			}})
		}
	}

	return ink.VNode{
		Type: "Box", Key: msg.UUID,
		Props:    ink.Props{"direction": "column"},
		Children: children,
	}
}
