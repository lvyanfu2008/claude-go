package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// paragraphToken mirrors Markdown.tsx cachedLexer fast path (single paragraph).
func paragraphToken(content string) Token {
	return Token{Type: "paragraph", Text: content, Raw: content}
}

// textOf collects Text / String segments under n (mirrors marked inline text).
func textOf(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := node.(type) {
		case *ast.Text:
			seg := t.Segment
			if seg.Start >= 0 && seg.Stop <= len(src) {
				b.Write(src[seg.Start:seg.Stop])
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func extractBlocks(doc *ast.Document, src []byte) []Token {
	var out []Token
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, blockTokens(c, src)...)
	}
	return out
}

func blockTokens(n ast.Node, src []byte) []Token {
	switch n := n.(type) {
	case *ast.Heading:
		return []Token{{
			Type:  "heading",
			Level: n.Level,
			Text:  strings.TrimSpace(textOf(n, src)),
		}}
	case *ast.Paragraph:
		return []Token{{Type: "paragraph", Text: strings.TrimSpace(textOf(n, src))}}
	case *ast.FencedCodeBlock:
		lang := strings.TrimSpace(string(n.Language(src)))
		var code strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			code.Write((&seg).Value(src))
		}
		return []Token{{Type: "code", Lang: lang, Text: code.String()}}
	case *ast.CodeBlock:
		var code strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			code.Write((&seg).Value(src))
		}
		return []Token{{Type: "code", Lang: "", Text: code.String()}}
	case *ast.Blockquote:
		return []Token{{Type: "blockquote", Text: strings.TrimSpace(textOf(n, src))}}
	case *ast.ThematicBreak:
		return []Token{{Type: "hr", Text: "---"}}
	case *ast.List:
		var items []Token
		for li := n.FirstChild(); li != nil; li = li.NextSibling() {
			if item, ok := li.(*ast.ListItem); ok {
				items = append(items, Token{Type: "list_item", Text: strings.TrimSpace(textOf(item, src))})
			}
		}
		return items
	default:
		t := strings.TrimSpace(textOf(n, src))
		if t != "" {
			return []Token{{Type: "paragraph", Text: t}}
		}
		return nil
	}
}

// ParseWithGoldmark runs goldmark parser (GFM-style via default New()).
func ParseWithGoldmark(content string) []Token {
	src := []byte(content)
	md := goldmark.New()
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)
	root, ok := doc.(*ast.Document)
	if !ok {
		return []Token{paragraphToken(content)}
	}
	toks := extractBlocks(root, src)
	if len(toks) == 0 {
		return []Token{paragraphToken(content)}
	}
	return toks
}

// CachedLexer mirrors Markdown.tsx cachedLexer (fast path + hash cache + goldmark).
func CachedLexer(content string) []Token {
	if !HasMarkdownSyntax(content) {
		return []Token{paragraphToken(content)}
	}
	key := HashContent(content)
	if toks, ok := globalCache.Get(key); ok {
		return toks
	}
	toks := ParseWithGoldmark(content)
	globalCache.Put(key, toks)
	return toks
}
