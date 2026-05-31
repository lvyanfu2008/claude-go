package compo

import (
	"strings"

	"goc/gou/ink"
)

const (
	promptPrefix      = "❯ " // ❯
	promptPlaceholder = ""
)

// PromptInput renders the input area matching claude-code TS:
//   ─────────────────────────────  (border line)
//   ❯ hello wo█rld                (prefix + text with inverted cursor)
//   model | cwd                    (status line)
func PromptInput(ctx *ink.Context) ink.VNode {
	val := ctx.Store.InputValue()
	cursor := ctx.Store.CursorPos()
	if cursor > len([]rune(val)) {
		cursor = len([]rune(val))
	}

	// Build the rendered line with inverse cursor
	rendered := buildCursorLine(val, cursor)
	prefix := promptPrefix

	return ink.VNode{
		Type: "Box", Key: "prompt-input",
		Props: ink.Props{"direction": "column"},
		Children: []ink.VNode{
			// Top border line
			borderLine(ctx),
			// Prompt row: prefix + input text
			{
				Type: "Box", Key: "prompt-row",
				Props: ink.Props{"direction": "row"},
				Children: []ink.VNode{
					// Prefix: ❯ with accent color
					{
						Type: "Text", Key: "prefix",
						Props: ink.Props{
							"content": prefix,
							"color":   ctx.Theme.ToolUse,
							"bold":    true,
						},
					},
					// Input text with cursor
					{
						Type: "Text", Key: "input-text",
						Props: ink.Props{
							"content": rendered,
							"color":   ctx.Theme.Default,
						},
					},
				},
			},
		},
	}
}

func borderLine(ctx *ink.Context) ink.VNode {
	w := ctx.Store.Width()
	if w < 10 {
		w = 80
	}
	return ink.VNode{
		Type: "Text", Key: "prompt-border",
		Props: ink.Props{
			"content": "╭" + strings.Repeat("─", w-2) + "╮",
			"dim":     true,
		},
	}
}

// buildCursorLine renders the input value with a reverse-video cursor
// at the given position. Empty input shows a dim placeholder with
// inverted first character (matching TS behavior).
func buildCursorLine(val string, cursor int) string {
	runes := []rune(val)
	if len(runes) == 0 {
		// Empty: show placeholder with first char inverted
		ph := []rune(promptPlaceholder)
		if len(ph) == 0 {
			return ""
		}
		return "\x1b[7m" + string(ph[0]) + "\x1b[27m" + "\x1b[2m" + string(ph[1:]) + "\x1b[22m"
	}

	// Insert inverse-video cursor at position
	var b strings.Builder
	for i, r := range runes {
		if i == cursor {
			b.WriteString("\x1b[7m")
			b.WriteRune(r)
			b.WriteString("\x1b[27m")
		} else {
			b.WriteRune(r)
		}
	}
	// If cursor at end, show inverted space
	if cursor >= len(runes) {
		b.WriteString("\x1b[7m \x1b[27m")
	}
	return b.String()
}
