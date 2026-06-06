// Package messages provides message rendering components extracted from gou-demo's main.go
// (markdown styling, inline segments, message filtering, spacing, and row rendering).
package messages

import (
	"fmt"
	"runtime"
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/app/config"
	"goc/gou/markdown"
	"goc/gou/theme"
)

// BaseMsgStyle adds the user-message row background so nested lipgloss.Render calls
// do not reset ANSI and punch holes in the gray bar.
func BaseMsgStyle(userRow bool) lipgloss.Style {
	s := lipgloss.NewStyle()
	if userRow {
		s = s.Background(theme.UserMessageBackground())
	}
	return s
}

// headingMarkdownStyle is bold + heading color; level spacing is leading spaces
// only (no # in output).
func HeadingMarkdownStyle(userRow bool) lipgloss.Style {
	return BaseMsgStyle(userRow).Bold(true).Foreground(theme.MarkdownHeading())
}

// toolRowLeadPrefix returns the dim tool lead glyph (⏺ / ●) for the first row
// of tool_use or assistant text.
func ToolRowLeadPrefix(userRow bool) string {
	glyph := "● " // ● — TS figures.BLACK_CIRCLE non-darwin
	if runtime.GOOS == "darwin" {
		glyph = "⏺ " // ⏺ — TS figures.BLACK_CIRCLE on darwin
	}
	return BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(glyph)
}

// prefixToolGlyphFirstLine prepends the dim tool lead (⏺ / ●) to the first line
// of rendered assistant text.
func PrefixToolGlyphFirstLine(body string) string {
	if body == "" {
		return ToolRowLeadPrefix(false)
	}
	p := ToolRowLeadPrefix(false)
	i := strings.IndexByte(body, '\n')
	if i < 0 {
		return p + body
	}
	return p + body[:i] + body[i:]
}

// StyleMarkdownTokens applies lipgloss to block tokens (mirrors Markdown.tsx roles,
// terminal-only). hl is the optional syntax highlighter for code blocks.
func StyleMarkdownTokens(hl *markdown.Highlighter, toks []markdown.Token, cols int, userRow bool) string {
	if len(toks) == 0 {
		return ""
	}
	var parts []string
	for _, t := range toks {
		switch t.Type {
		case "heading":
			lv := min(max(t.Level, 1), 6)
			levelPad := strings.Repeat(" ", (lv-1)*2)
			hst := HeadingMarkdownStyle(userRow)
			if len(t.Segments) > 0 {
				inner := StyleMarkdownInlineSegments(t.Segments, "", userRow)
				rendered := hst.Render(inner)
				parts = append(parts, wrapHeadingForMessagePane(rendered, levelPad, cols))
			} else {
				plain := strings.TrimSpace(t.Text)
				wrapped := wrapHeadingForMessagePane(plain, levelPad, cols)
				lines := strings.Split(wrapped, "\n")
				var hb strings.Builder
				for i, ln := range lines {
					if i > 0 {
						hb.WriteByte('\n')
					}
					hb.WriteString(hst.Render(ln))
				}
				parts = append(parts, hb.String())
			}
		case "code":
			// Apply syntax highlighting if highlighter is available
			var highlightedCode string
			if hl != nil {
				highlighted, err := hl.HighlightCode(t.Text, t.Lang)
				if err == nil && highlighted != "" {
					highlightedCode = highlighted
				}
			}

			// If highlighting failed or highlighter is disabled, use plain code
			if highlightedCode == "" {
				cb := "```" + t.Lang + "\n" + t.Text
				if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
					cb += "\n"
				}
				cb += "```"
				parts = append(parts, BaseMsgStyle(userRow).Faint(true).Render(cb))
			} else {
				// For highlighted code, just show the highlighted content without backticks
				parts = append(parts, BaseMsgStyle(userRow).Render(highlightedCode))
			}
		case "list_item":
			indent := strings.Repeat(" ", t.ListIndent)
			var prefix string
			if t.ListContinuation {
				prefix = indent + "   "
			} else if t.ListOrdered && t.ListIndex > 0 {
				prefix = indent + fmt.Sprintf("%d. ", t.ListIndex)
			} else {
				prefix = indent + "- "
			}
			if len(t.Segments) > 0 {
				parts = append(parts, StyleMarkdownInlineSegments(t.Segments, prefix, userRow))
			} else if userRow {
				parts = append(parts, BaseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(prefix+t.Text))
			} else {
				parts = append(parts, BaseMsgStyle(userRow).Render(prefix+t.Text))
			}
		case "blockquote":
			if len(t.Segments) > 0 {
				inner := StyleMarkdownInlineSegments(t.Segments, "", userRow)
				pref := "> " + strings.ReplaceAll(inner, "\n", "\n> ")
				parts = append(parts, pref)
			} else if userRow {
				parts = append(parts, BaseMsgStyle(userRow).Foreground(theme.UserMessageText()).Italic(true).Bold(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			} else {
				parts = append(parts, BaseMsgStyle(userRow).Italic(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			}
		case "hr":
			parts = append(parts, BaseMsgStyle(userRow).Faint(true).Render("---"))
		case "paragraph":
			if len(t.Segments) > 0 {
				parts = append(parts, StyleMarkdownInlineSegments(t.Segments, "", userRow))
			} else {
				if userRow {
					parts = append(parts, BaseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
				} else {
					parts = append(parts, t.Text)
				}
			}
		default:
			if userRow {
				parts = append(parts, BaseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
			} else {
				parts = append(parts, t.Text)
			}
		}
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			if toks[i-1].Type == "list_item" {
				b.WriteByte('\n')
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(part)
	}
	return strings.TrimSpace(b.String())
}

// wrapHeadingForMessagePane delegates to config.WrapHeadingForMessagePane.
func wrapHeadingForMessagePane(content string, levelPad string, cols int) string {
	return config.WrapHeadingForMessagePane(content, levelPad, cols)
}
