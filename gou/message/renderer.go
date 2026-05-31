// Package message implements TS-style message rendering for Go TUI.
// Architecture mirrors claude-code-best/src/components/Message.tsx.
package message

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"goc/gou/layout"
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

	// NEW: per-message context (computed by MessageRow)
	IsActiveCollapsedGroup bool
	IsInProgress           bool

	// NEW: shared state (same across all messages in a render pass)
	InProgressToolUseIDs map[string]struct{}
	StreamingToolUseIDs  map[string]struct{}
	ResolvedToolUseIDs   map[string]struct{}

	// NEW: transcript features
	SearchHighlight       string
	ShowToolUseCtrlOHint  bool
	ShowResolvedToolStats bool

	// NEW: streaming state
	StreamingText         string
	StreamingThinkingText string
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
			&AttachmentMessageRenderer{},
			// After Messages.tsx filter; no-op if a progress row still reaches the dispatcher.
			&ProgressMessageRenderer{},
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
	if text == "" {
		return []string{""}
	}
	// Preserve explicit newlines from tool output, then wrap each visual line.
	wrapped := layout.WrapForViewport(text, width)
	return strings.Split(wrapped, "\n")
}

// renderMarkdown renders markdown text with theme using the styled renderer.
func renderMarkdown(text string, width int, palette *theme.Palette, highlighter *markdown.Highlighter) []string {
	if text == "" {
		return []string{""}
	}

	tokens := markdown.ParseWithGoldmark(text)

	baseStyle := lipgloss.NewStyle().Foreground(palette.Default)
	codeStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Foreground(palette.Heading).Bold(true)
	italicStyle := lipgloss.NewStyle().Italic(true)
	inlineCodeStyle := lipgloss.NewStyle().Foreground(palette.InlineCode)

	rendered := markdown.RenderTokensStyled(
		tokens, highlighter, width,
		baseStyle, codeStyle, boldStyle, italicStyle, inlineCodeStyle,
	)

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

// getContainerWidth returns the effective container width.
func getContainerWidth(ctx *RenderContext) int {
	if ctx.ContainerWidth != nil {
		return *ctx.ContainerWidth
	}
	return ctx.Width
}

// highlightSearchPlain wraps matching substrings with ANSI reverse-video.
func highlightSearchPlain(haystack, needle string) string {
	if strings.TrimSpace(needle) == "" {
		return haystack
	}
	hlStyle := "\x1b[7m"
	reset := "\x1b[0m"
	lower := strings.ToLower(haystack)
	needleLower := strings.ToLower(needle)
	var b strings.Builder
	idx := 0
	for {
		pos := strings.Index(lower[idx:], needleLower)
		if pos < 0 {
			b.WriteString(haystack[idx:])
			break
		}
		b.WriteString(haystack[idx : idx+pos])
		b.WriteString(hlStyle)
		b.WriteString(haystack[idx+pos : idx+pos+len(needle)])
		b.WriteString(reset)
		idx += pos + len(needle)
	}
	return b.String()
}
