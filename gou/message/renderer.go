// Package message implements TS-style message rendering for Go TUI.
// Architecture mirrors claude-code-best/src/components/Message.tsx.
package message

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"goc/gou/markdown"
	"goc/gou/theme"
	"goc/types"
)

// RenderContext contains rendering context information.
type RenderContext struct {
	Width           int
	Verbose         bool
	Theme           *theme.Palette
	IsTranscript    bool
	IsStatic        bool
	ShouldAnimate   bool
	ShouldShowDot   bool
	AddMargin       bool
	ContainerWidth  *int
	Style           string // "condensed" or empty
	IsUserContinuation bool
	Highlighter     *markdown.Highlighter
}

// Renderer is the interface for message renderers.
type Renderer interface {
	// CanRender returns true if this renderer can render the given message.
	CanRender(msg *types.Message) bool

	// Render renders the message and returns the rendered lines.
	Render(msg *types.Message, ctx *RenderContext) ([]string, error)

	// Measure returns the number of lines this message will occupy.
	Measure(msg *types.Message, ctx *RenderContext) (int, error)
}

// Dispatcher routes messages to appropriate renderers.
type Dispatcher struct {
	renderers []Renderer
}

// NewDispatcher creates a new message dispatcher with default renderers.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		renderers: []Renderer{
			&UserMessageRenderer{},
			&AssistantMessageRenderer{},
			&SystemMessageRenderer{},
			&ToolUseMessageRenderer{},
			&CollapsedGroupRenderer{},
			&GroupedToolUseRenderer{},
		},
	}
}

// Render renders a message using the appropriate renderer.
func (d *Dispatcher) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	for _, renderer := range d.renderers {
		if renderer.CanRender(msg) {
			return renderer.Render(msg, ctx)
		}
	}
	return []string{fmt.Sprintf("Unknown message type: %s", msg.Type)}, nil
}

// Measure measures a message using the appropriate renderer.
func (d *Dispatcher) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	for _, renderer := range d.renderers {
		if renderer.CanRender(msg) {
			return renderer.Measure(msg, ctx)
		}
	}
	return 1, nil // Default to 1 line for unknown messages
}

// Helper functions

// wrapText wraps text to the given width, preserving ANSI codes.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	// ANSI-aware wrapping
	var lines []string
	var currentLine strings.Builder
	currentLineVisibleLen := 0

	// Split by words, but preserve ANSI sequences
	// Simple approach: split by space, but this doesn't handle ANSI codes well
	// For now, use the old logic but with visible length calculation
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	// Helper to calculate visible length (ignoring ANSI escape sequences)
	visibleLen := func(s string) int {
		// Remove ANSI escape sequences for length calculation
		// Simple regex: \x1b\[[0-9;]*[A-Za-z]
		inEscape := false
		visible := 0
		for i := 0; i < len(s); i++ {
			if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
				inEscape = true
				i++ // Skip '['
				continue
			}
			if inEscape {
				// Check for end of escape sequence
				if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
					inEscape = false
				}
				continue
			}
			visible++
		}
		return visible
	}

	currentLine.WriteString(words[0])
	currentLineVisibleLen = visibleLen(words[0])

	for _, word := range words[1:] {
		wordVisibleLen := visibleLen(word)
		if currentLineVisibleLen+1+wordVisibleLen <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
			currentLineVisibleLen += 1 + wordVisibleLen
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
			currentLineVisibleLen = wordVisibleLen
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// renderMarkdown renders markdown text with theme.
func renderMarkdown(text string, width int, palette *theme.Palette, highlighter *markdown.Highlighter) []string {
	if text == "" {
		return []string{""}
	}

	// Parse markdown
	tokens := markdown.ParseWithGoldmark(text)

	// Convert palette to lipgloss style for markdown rendering
	style := paletteToLipglossStyle(palette)

	// Render with highlighting
	rendered := markdown.RenderTokensWithHighlight(tokens, highlighter, style)

	// Split into lines and wrap if needed
	lines := strings.Split(rendered, "\n")
	var result []string

	for _, line := range lines {
		// Check if line contains ANSI escape sequences (likely code)
		hasAnsi := strings.Contains(line, "\x1b[")

		if hasAnsi {
			// For code with ANSI, never wrap - keep as is
			result = append(result, line)
		} else {
			// For plain text, calculate visible length
			visibleLen := len(line)
			if width > 0 && visibleLen > width {
				// Wrap long lines
				wrapped := wrapText(line, width)
				result = append(result, wrapped...)
			} else {
				result = append(result, line)
			}
		}
	}

	return result
}

// paletteToLipglossStyle converts a theme palette to a lipgloss style for markdown rendering.
func paletteToLipglossStyle(palette *theme.Palette) lipgloss.Style {
	// Create a basic style with heading color
	style := lipgloss.NewStyle().Foreground(palette.Heading)
	return style
}

// getContainerWidth returns the effective container width.
func getContainerWidth(ctx *RenderContext) int {
	if ctx.ContainerWidth != nil {
		return *ctx.ContainerWidth
	}
	return ctx.Width
}