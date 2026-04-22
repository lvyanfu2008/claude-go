package markdown

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var tableSeparatorPattern = regexp.MustCompile(`^\s*\|?[\s:-]+(?:\|[\s:-]+)+\|?\s*$`)

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

func joinInlineSegments(segs []InlineSegment) string {
	var b strings.Builder
	for i := range segs {
		b.WriteString(segs[i].Text)
	}
	return b.String()
}

func trimInlineSegmentEdges(segs []InlineSegment) {
	if len(segs) == 0 {
		return
	}
	segs[0].Text = strings.TrimLeft(segs[0].Text, " \t\n\r")
	segs[len(segs)-1].Text = strings.TrimRight(segs[len(segs)-1].Text, " \t\n\r")
}

func hasRichSegments(segs []InlineSegment) bool {
	for i := range segs {
		s := segs[i]
		if s.Code || s.Bold || s.Italic {
			return true
		}
	}
	return false
}

func segmentText(t *ast.Text, src []byte) string {
	seg := t.Segment
	if seg.Start >= 0 && seg.Stop <= len(src) {
		return string(src[seg.Start:seg.Stop])
	}
	return ""
}

func canMergeInline(a, b InlineSegment) bool {
	if a.Code != b.Code {
		return false
	}
	return a.Bold == b.Bold && a.Italic == b.Italic
}

func appendMerged(out []InlineSegment, seg InlineSegment) []InlineSegment {
	if seg.Text == "" && !seg.Code {
		return out
	}
	if len(out) > 0 && canMergeInline(out[len(out)-1], seg) {
		out[len(out)-1].Text += seg.Text
		return out
	}
	out = append(out, seg)
	return out
}

// appendInlineNode flattens goldmark inline nodes into styled segments (code, strong, emphasis, links as text).
func appendInlineNode(out []InlineSegment, n ast.Node, src []byte, bold, italic bool) []InlineSegment {
	switch t := n.(type) {
	case *ast.Text:
		s := segmentText(t, src)
		if s == "" {
			return out
		}
		return appendMerged(out, InlineSegment{Text: s, Bold: bold, Italic: italic})
	case *ast.CodeSpan:
		return appendMerged(out, InlineSegment{Code: true, Text: textOf(t, src)})
	case *ast.Emphasis:
		b, i := bold, italic
		switch {
		case t.Level >= 3:
			b, i = true, true
		case t.Level == 2:
			b = true
		default:
			i = true
		}
		for c := t.FirstChild(); c != nil; c = c.NextSibling() {
			out = appendInlineNode(out, c, src, b, i)
		}
		return out
	case *ast.Link:
		for c := t.FirstChild(); c != nil; c = c.NextSibling() {
			out = appendInlineNode(out, c, src, bold, italic)
		}
		return out
	case *ast.AutoLink:
		label := string(t.Label(src))
		return appendMerged(out, InlineSegment{Text: label, Bold: bold, Italic: italic})
	case *ast.Image:
		return out
	default:
		s := textOf(t, src)
		if s != "" {
			return appendMerged(out, InlineSegment{Text: s, Bold: bold, Italic: italic})
		}
		return out
	}
}

// paragraphInlineSegments splits paragraph inline content into styled runs (goldmark AST).
func paragraphInlineSegments(p *ast.Paragraph, src []byte) []InlineSegment {
	var out []InlineSegment
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		out = appendInlineNode(out, c, src, false, false)
	}
	return out
}

// textBlockInlineSegments handles list-item TextBlock (goldmark uses TextBlock instead of Paragraph in tight lists).
func textBlockInlineSegments(tb *ast.TextBlock, src []byte) []InlineSegment {
	var out []InlineSegment
	for c := tb.FirstChild(); c != nil; c = c.NextSibling() {
		out = appendInlineNode(out, c, src, false, false)
	}
	if len(out) > 0 {
		return out
	}
	s := strings.TrimSpace(string(tb.Text(src)))
	if s == "" {
		return nil
	}
	return []InlineSegment{{Text: s}}
}

func paragraphTokenFromParagraph(p *ast.Paragraph, src []byte) []Token {
	segs := paragraphInlineSegments(p, src)
	if len(segs) == 0 {
		return []Token{{Type: "paragraph", Text: strings.TrimSpace(textOf(p, src))}}
	}
	trimInlineSegmentEdges(segs)
	fullTrim := strings.TrimSpace(joinInlineSegments(segs))
	if !hasRichSegments(segs) {
		return []Token{{Type: "paragraph", Text: fullTrim}}
	}
	return []Token{{Type: "paragraph", Text: fullTrim, Segments: segs}}
}

func headingTokenFromHeading(h *ast.Heading, src []byte) []Token {
	var segs []InlineSegment
	for c := h.FirstChild(); c != nil; c = c.NextSibling() {
		segs = appendInlineNode(segs, c, src, false, false)
	}
	if len(segs) == 0 {
		return []Token{{Type: "heading", Level: h.Level, Text: strings.TrimSpace(textOf(h, src))}}
	}
	trimInlineSegmentEdges(segs)
	fullTrim := strings.TrimSpace(joinInlineSegments(segs))
	if !hasRichSegments(segs) {
		return []Token{{Type: "heading", Level: h.Level, Text: fullTrim}}
	}
	return []Token{{Type: "heading", Level: h.Level, Text: fullTrim, Segments: segs}}
}

// listItemTokensFromListItem walks ListItem children: paragraphs, text blocks, nested lists,
// and fenced/indented code blocks (CommonMark).
func listItemTokensFromListItem(li *ast.ListItem, src []byte, ordered bool, listIndex int, nestDepth int) []Token {
	var out []Token
	seenPara := false
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		switch x := c.(type) {
		case *ast.Paragraph:
			segs := paragraphInlineSegments(x, src)
			trimInlineSegmentEdges(segs)
			fullTrim := strings.TrimSpace(joinInlineSegments(segs))
			if fullTrim == "" {
				continue
			}
			tok := Token{
				Type:       "list_item",
				ListIndent: nestDepth * 2,
				Text:       fullTrim,
			}
			if !seenPara {
				tok.ListOrdered = ordered
				tok.ListIndex = listIndex
				seenPara = true
			} else {
				tok.ListContinuation = true
			}
			if hasRichSegments(segs) {
				tok.Segments = segs
			}
			out = append(out, tok)
		case *ast.TextBlock:
			segs := textBlockInlineSegments(x, src)
			if len(segs) == 0 {
				continue
			}
			trimInlineSegmentEdges(segs)
			fullTrim := strings.TrimSpace(joinInlineSegments(segs))
			if fullTrim == "" {
				continue
			}
			tok := Token{
				Type:       "list_item",
				ListIndent: nestDepth * 2,
				Text:       fullTrim,
			}
			if !seenPara {
				tok.ListOrdered = ordered
				tok.ListIndex = listIndex
				seenPara = true
			} else {
				tok.ListContinuation = true
			}
			if hasRichSegments(segs) {
				tok.Segments = segs
			}
			out = append(out, tok)
		case *ast.List:
			nestedOrdered := x.IsOrdered()
			n := x.Start
			if !nestedOrdered {
				n = 0
			}
			for li2 := x.FirstChild(); li2 != nil; li2 = li2.NextSibling() {
				if item2, ok := li2.(*ast.ListItem); ok {
					out = append(out, listItemTokensFromListItem(item2, src, nestedOrdered, n, nestDepth+1)...)
					if nestedOrdered {
						n++
					}
				}
			}
		case *ast.FencedCodeBlock:
			// CommonMark: list items may contain fenced code blocks (previously dropped).
			out = append(out, blockTokens(x, src)...)
		case *ast.CodeBlock:
			out = append(out, blockTokens(x, src)...)
		}
	}
	if len(out) == 0 {
		return []Token{{
			Type:        "list_item",
			Text:        strings.TrimSpace(textOf(li, src)),
			ListIndent:  nestDepth * 2,
			ListOrdered: ordered,
			ListIndex:   listIndex,
		}}
	}
	return out
}

func blockquoteTokens(b *ast.Blockquote, src []byte) []Token {
	var paras []*ast.Paragraph
	for c := b.FirstChild(); c != nil; c = c.NextSibling() {
		if p, ok := c.(*ast.Paragraph); ok {
			paras = append(paras, p)
		}
	}
	if len(paras) == 0 {
		return []Token{{Type: "blockquote", Text: strings.TrimSpace(textOf(b, src))}}
	}
	parts := make([]string, 0, len(paras))
	for _, p := range paras {
		parts = append(parts, strings.TrimSpace(textOf(p, src)))
	}
	full := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if len(paras) == 1 {
		segs := paragraphInlineSegments(paras[0], src)
		trimInlineSegmentEdges(segs)
		if hasRichSegments(segs) {
			return []Token{{Type: "blockquote", Text: full, Segments: segs}}
		}
	}
	return []Token{{Type: "blockquote", Text: full}}
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
		return headingTokenFromHeading(n, src)
	case *ast.Paragraph:
		return paragraphTokenFromParagraph(n, src)
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
		return blockquoteTokens(n, src)
	case *ast.ThematicBreak:
		return []Token{{Type: "hr", Text: "---"}}
	case *ast.List:
		ordered := n.IsOrdered()
		itemNum := n.Start
		if !ordered {
			itemNum = 0
		}
		var items []Token
		for li := n.FirstChild(); li != nil; li = li.NextSibling() {
			if item, ok := li.(*ast.ListItem); ok {
				items = append(items, listItemTokensFromListItem(item, src, ordered, itemNum, 0)...)
				if ordered {
					itemNum++
				}
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

func isTableHeaderLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || !strings.Contains(trimmed, "|") {
		return false
	}
	// Must include at least one non-pipe cell character.
	return strings.Trim(trimmed, " |-:\t") != ""
}

func isTableSeparatorLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}
	return tableSeparatorPattern.MatchString(trimmed)
}

func isTableDataLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed != "" && strings.Contains(trimmed, "|")
}

// wrapMarkdownTablesAsCodeBlocks protects GFM table blocks from being flattened
// when using a goldmark build without the table extension in vendor/.
func wrapMarkdownTablesAsCodeBlocks(content string) string {
	if !strings.Contains(content, "|") {
		return content
	}

	lines := strings.Split(content, "\n")
	var out []string
	inFence := false
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			out = append(out, line)
			i++
			continue
		}
		if inFence {
			out = append(out, line)
			i++
			continue
		}

		if i+1 < len(lines) && isTableHeaderLine(lines[i]) && isTableSeparatorLine(lines[i+1]) {
			j := i + 2
			for j < len(lines) && isTableDataLine(lines[j]) {
				j++
			}
			out = append(out, "```text")
			out = append(out, lines[i:j]...)
			out = append(out, "```")
			i = j
			continue
		}

		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
}

// ParseWithGoldmark runs goldmark parser (GFM-style via default New()).
func ParseWithGoldmark(content string) []Token {
	normalized := wrapMarkdownTablesAsCodeBlocks(content)
	src := []byte(normalized)
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
