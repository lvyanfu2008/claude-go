// Package compo provides application-level VNode components for the ink TUI.
package compo

import (
	"charm.land/lipgloss/v2"

	"goc/gou/ink"
	"goc/gou/markdown"
)

// Markdown renders GFM markdown to a Text VNode with ANSI-styled content.
func Markdown(ctx *ink.Context, content string, width int) ink.VNode {
	w := width
	if w <= 0 {
		w = 80
	}

	var rendered string
	if markdown.HasMarkdownSyntax(content) {
		tokens := markdown.CachedLexer(content)
		baseStyle := lipgloss.NewStyle().Foreground(ctx.Theme.Default)
		codeStyle := lipgloss.NewStyle().Foreground(ctx.Theme.InlineCode)
		rendered = markdown.RenderTokensWithHighlight(tokens, nil, baseStyle, codeStyle)
	} else {
		rendered = content
	}

	return ink.VNode{
		Type: "Text",
		Key:  "md",
		Props: ink.Props{
			"content": rendered,
			"color":   ctx.Theme.Default,
		},
	}
}

// Row creates a horizontal Box with gap.
func Row(gap int, children ...ink.VNode) ink.VNode {
	return ink.VNode{
		Type: "Box",
		Props: ink.Props{
			"direction": "row",
			"gap":       gap,
		},
		Children: children,
	}
}

// Col creates a vertical Box.
func Col(children ...ink.VNode) ink.VNode {
	return ink.VNode{
		Type: "Box",
		Props: ink.Props{
			"direction": "column",
		},
		Children: children,
	}
}
