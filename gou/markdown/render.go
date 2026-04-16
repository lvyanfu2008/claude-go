package markdown

import (
	"fmt"
	"strings"
)

// RenderTokensPlain turns block tokens into plain text for layout height / terminal preview
// (no ANSI — lipgloss applied in the TUI layer).
func RenderTokensPlain(tokens []Token) string {
	var b strings.Builder
	for _, t := range tokens {
		switch t.Type {
		case "heading":
			for range t.Level {
				b.WriteString("#")
			}
			b.WriteString(" ")
			if len(t.Segments) > 0 {
				for _, s := range t.Segments {
					b.WriteString(s.Text)
				}
			} else {
				b.WriteString(t.Text)
			}
			b.WriteString("\n\n")
		case "paragraph":
			if len(t.Segments) > 0 {
				for _, s := range t.Segments {
					b.WriteString(s.Text)
				}
			} else {
				b.WriteString(t.Text)
			}
			b.WriteString("\n\n")
		case "code":
			b.WriteString("```")
			b.WriteString(t.Lang)
			b.WriteByte('\n')
			b.WriteString(t.Text)
			if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
				b.WriteByte('\n')
			}
			b.WriteString("```\n\n")
		case "list_item":
			b.WriteString(strings.Repeat(" ", t.ListIndent))
			if t.ListContinuation {
				b.WriteString("   ")
			} else if t.ListOrdered && t.ListIndex > 0 {
				fmt.Fprintf(&b, "%d. ", t.ListIndex)
			} else {
				b.WriteString("- ")
			}
			if len(t.Segments) > 0 {
				for _, s := range t.Segments {
					b.WriteString(s.Text)
				}
			} else {
				b.WriteString(t.Text)
			}
			b.WriteByte('\n')
		case "blockquote":
			b.WriteString("> ")
			b.WriteString(strings.ReplaceAll(t.Text, "\n", "\n> "))
			b.WriteString("\n\n")
		case "hr":
			b.WriteString("---\n\n")
		default:
			if t.Text != "" {
				b.WriteString(t.Text)
				b.WriteString("\n\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// NormalizeStreamingForLexer closes an odd number of ``` fences so goldmark can parse
// in-flight assistant output (Markdown.tsx StreamingMarkdown incomplete tree pattern).
func NormalizeStreamingForLexer(s string) string {
	if strings.Count(s, "```")%2 == 0 {
		return s
	}
	return s + "\n```\n"
}

// CachedLexerStreaming runs lexer on normalized streaming buffer.
func CachedLexerStreaming(s string) []Token {
	if s == "" {
		return nil
	}
	norm := NormalizeStreamingForLexer(s)
	return CachedLexer(norm)
}
