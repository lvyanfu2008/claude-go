package compo

import (
	"fmt"
	"strings"

	"goc/gou/ink"
)

// UserMessage renders a user message with its content blocks.
func UserMessage(ctx *ink.Context, msg ink.Message) ink.VNode {
	var children []ink.VNode
	for _, block := range msg.ContentBlocks {
		switch block.Type {
		case "text":
			children = append(children, userTextContent(ctx, block))
		case "tool_result":
			children = append(children, ToolResultBlock(ctx, block))
		}
	}
	if len(children) == 0 {
		return ink.VNode{Type: "Text"}
	}
	return ink.VNode{
		Type: "Box", Key: msg.UUID,
		Props:    ink.Props{"direction": "column"},
		Children: children,
	}
}

func userTextContent(ctx *ink.Context, block ink.ContentBlock) ink.VNode {
	prefix := ink.VNode{
		Type: "Text",
		Props: ink.Props{
			"content": fmt.Sprintf("%-5s", "⏺"),
			"color":   ctx.Theme.User,
			"bold":    true,
		},
	}
	return Row(1, prefix, Markdown(ctx, block.Content, ctx.Store.Width()-5))
}

func truncateLine(s string, maxW int) string {
	if maxW < 10 { maxW = 40 }
	line := firstLine(s)
	if len(line) > maxW {
		return line[:maxW-1] + "…"
	}
	return line
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' || r == '\r' {
			return s[:i]
		}
	}
	return s
}

func stripMarkdown(s string) string {
	// Remove common markdown formatting for plain-text display
	s = strings.ReplaceAll(s, "```", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	return s
}
