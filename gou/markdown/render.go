package markdown

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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

// RenderTokensWithHighlight 将token渲染为带ANSI高亮的文本。
// theme 用于标题与普通内联（粗体/斜体）；inlineCode 专用于 `反引号` 内联代码（与 theme.Palette.InlineCode 对齐）。
func RenderTokensWithHighlight(tokens []Token, highlighter *Highlighter, theme lipgloss.Style, inlineCode lipgloss.Style) string {
	var b strings.Builder
	for _, t := range tokens {
		switch t.Type {
		case "heading":
			// 应用标题样式，不输出#
			lv := t.Level
			if lv < 1 {
				lv = 1
			}
			if lv > 6 {
				lv = 6
			}
			levelPad := strings.Repeat(" ", (lv-1)*2)
			headingStyle := theme.Copy().Bold(true)

			var headingText string
			if len(t.Segments) > 0 {
				var segText strings.Builder
				for _, s := range t.Segments {
					segText.WriteString(applyInlineStyle(s, theme, inlineCode))
				}
				headingText = segText.String()
			} else {
				headingText = t.Text
			}

			// 应用标题样式
			rendered := headingStyle.Render(strings.TrimSpace(headingText))
			b.WriteString(levelPad + rendered + "\n\n")

		case "paragraph":
			if len(t.Segments) > 0 {
				for _, s := range t.Segments {
					b.WriteString(applyInlineStyle(s, theme, inlineCode))
				}
			} else {
				b.WriteString(t.Text)
			}
			b.WriteString("\n\n")
		case "code":
			// 代码块高亮
			var highlighted string
			var highlightErr error
			if highlighter != nil {
				highlighted, highlightErr = highlighter.HighlightCode(t.Text, t.Lang)
			}

			if highlighted != "" && highlightErr == nil {
				// 如果有高亮器且高亮成功，显示高亮后的代码（没有围栏）
				b.WriteString(highlighted)
			} else {
				// 如果没有高亮器或高亮失败，显示带围栏的代码并应用淡色样式
				codeStyle := theme.Copy().Faint(true)
				// 处理代码块中的...
				codeText := t.Text
				if strings.Contains(codeText, "...") {
					codeText = strings.ReplaceAll(codeText, "...", "█")
				}
				cb := "```" + t.Lang + "\n" + codeText
				if codeText != "" && !strings.HasSuffix(codeText, "\n") {
					cb += "\n"
				}
				cb += "```"
				b.WriteString(codeStyle.Render(cb))
			}
			b.WriteString("\n\n")
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
					b.WriteString(applyInlineStyle(s, theme, inlineCode))
				}
			} else {
				b.WriteString(t.Text)
			}
			b.WriteByte('\n')
		case "blockquote":
			// 应用引用块样式
			quoteStyle := theme.Copy().Italic(true)
			var quoteText string
			if len(t.Segments) > 0 {
				var segText strings.Builder
				for _, s := range t.Segments {
					segText.WriteString(applyInlineStyle(s, theme, inlineCode))
				}
				quoteText = segText.String()
			} else {
				quoteText = t.Text
			}

			// 添加>前缀并应用样式
			quoted := "> " + strings.ReplaceAll(quoteText, "\n", "\n> ")
			b.WriteString(quoteStyle.Render(quoted) + "\n\n")
		case "hr":
			// 应用淡色样式
			hrStyle := theme.Copy().Faint(true)
			b.WriteString(hrStyle.Render("---") + "\n\n")
		default:
			if t.Text != "" {
				b.WriteString(t.Text)
				b.WriteString("\n\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// applyInlineStyle 应用内联样式（粗体、斜体、代码）
func applyInlineStyle(seg InlineSegment, theme, inlineCode lipgloss.Style) string {
	text := seg.Text

	if seg.Code {
		// 内联代码使用主题色（与 gou-demo styleMarkdownInlineSegments / theme.InlineCode 一致）
		if strings.Contains(text, "...") {
			text = strings.ReplaceAll(text, "...", "█")
		}
		return inlineCode.Render(text)
	}
	
	// 处理...这类数据，渲染为淡蓝色的█
	if text == "..." {
		// 使用淡蓝色（颜色代码87）
		return theme.Copy().Foreground(lipgloss.Color("87")).Render("█")
	}
	
	// 检查文本中是否包含...（即使不是单独的segment）
	if strings.Contains(text, "...") {
		// 将...替换为淡蓝色的█
		// 我们需要更精细的处理来保持其他样式
		parts := strings.Split(text, "...")
		var result strings.Builder
		for i, part := range parts {
			if part != "" {
				// 应用当前段的样式，从theme开始
				style := theme.Copy()
				if seg.Bold {
					style = style.Bold(true)
				}
				if seg.Italic {
					style = style.Italic(true)
				}
				result.WriteString(style.Render(part))
			}
			if i < len(parts)-1 {
				// 添加淡蓝色的█
				result.WriteString(theme.Copy().Foreground(lipgloss.Color("87")).Render("█"))
			}
		}
		return result.String()
	}

	style := theme.Copy()
	if seg.Bold {
		style = style.Bold(true)
	}
	if seg.Italic {
		style = style.Italic(true)
	}

	return style.Render(text)
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
