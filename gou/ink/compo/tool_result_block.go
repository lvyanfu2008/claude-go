package compo

import "goc/gou/ink"

// ToolResultBlock renders a tool result with expand/collapse based on transcriptShowAll.
// In collapsed mode, it shows a one-line summary. In expanded mode, it renders full content.
func ToolResultBlock(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	showAll := ctx.Store.GetMeta("transcriptShowAll") == "1"

	prefix := "  ⎿ "
	switch block.State {
	case "error":
		return Row(1,
			ink.VNode{Type: "Text", Props: ink.Props{"content": prefix + "✗ Error:", "color": ctx.Theme.ToolError}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": truncateLine(block.Content, ctx.Store.Width()-12), "dim": true}},
		)
	case "rejected":
		return ink.VNode{Type: "Text", Props: ink.Props{"content": prefix + "Permission denied", "dim": true}}
	default:
		if showAll {
			// Full content — render with markdown
			return Row(1,
				ink.VNode{Type: "Text", Props: ink.Props{"content": prefix, "dim": true}},
				Markdown(ctx, block.Content, ctx.Store.Width()-6),
			)
		}
		// Collapsed: one-line summary
		summary := truncateLine(stripMarkdown(block.Content), ctx.Store.Width()-10)
		return Row(1,
			ink.VNode{Type: "Text", Props: ink.Props{"content": prefix, "dim": true}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": summary, "dim": true}},
			ink.VNode{Type: "Text", Props: ink.Props{"content": "  (ctrl+o to expand)", "dim": true}},
		)
	}
}
