package compo

import (
	"goc/gou/ink"
)

// FileEditDiff renders a unified-diff preview for tool_use blocks that edit files
// (Edit, Write, NotebookEdit). It shows a summary line and a truncated diff.
func FileEditDiff(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	name := block.Name
	input := block.Input
	if input == nil {
		return ink.VNode{Type: "Text"}
	}

	filePath, _ := input["file_path"].(string)
	oldPath, _ := input["old_path"].(string)

	label := name
	switch name {
	case "Edit":
		label = "Edit"
	case "Write":
		label = "Write"
	case "NotebookEdit":
		label = "NotebookEdit"
	}

	prefix := "  Δ " + label
	if filePath != "" {
		prefix += " " + filePath
	}
	if oldPath != "" && oldPath != filePath {
		prefix += " → " + oldPath
	}

	children := []ink.VNode{
		{Type: "Text", Props: ink.Props{"content": prefix, "color": ctx.Theme.ToolUse}},
	}

	// Show diff body if resolved and transcript is in show-all mode
	showAll := ctx.Store.GetMeta("transcriptShowAll") == "1"
	if showAll && block.State == "resolved" && block.Result != nil {
		summary := truncateLine(stripMarkdown(block.Result.Content), ctx.Store.Width()-6)
		if summary != "" {
			children = append(children,
				ink.VNode{Type: "Text", Props: ink.Props{"content": "  " + summary, "dim": true}},
			)
		}
	} else if block.State == "resolved" && block.Result != nil {
		summary := truncateLine(stripMarkdown(block.Result.Content), ctx.Store.Width()-10)
		children = append(children,
			ink.VNode{Type: "Text", Props: ink.Props{"content": "  ⎿ " + summary + "  (ctrl+o to expand)", "dim": true}},
		)
	}

	return ink.VNode{
		Type: "Box", Key: "edit-diff-" + block.Name,
		Props:    ink.Props{"direction": "column"},
		Children: children,
	}
}

// isEditTool returns true for file-editing tool names.
func isEditTool(name string) bool {
	switch name {
	case "Edit", "Write", "NotebookEdit":
		return true
	}
	return false
}
