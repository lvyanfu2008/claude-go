package compo

import (
	"goc/gou/ink"
	"goc/gou/markdown"
)

// HighlightedCode renders a standalone code block with syntax highlighting.
// If language is empty, it uses auto-detection.
func HighlightedCode(ctx *ink.Context, code, language string) ink.VNode {
	w := ctx.Store.Width() - 4
	if w < 20 {
		w = 80
	}

	fenced := "```" + language + "\n" + code + "\n```"
	tokens := markdown.CachedLexer(fenced)
	rendered := markdown.RenderTokensPlain(tokens)

	return ink.VNode{
		Type: "Text", Key: "code-block",
		Props: ink.Props{
			"content": rendered,
			"color":   ctx.Theme.InlineCode,
		},
	}
}
