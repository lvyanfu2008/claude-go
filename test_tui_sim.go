package main

import (
	"fmt"
	"strings"

	"goc/gou/markdown"
	"goc/gou/theme"
	"charm.land/lipgloss/v2"
)

func main() {
	// Initialize theme
	theme.InitFromThemeName("default")
	palette := theme.ActivePalette()

	fmt.Printf("Theme palette: Heading=%v\n", palette.Heading)

	// Create highlighter
	config := markdown.DefaultHighlightConfig()
	highlighter, err := markdown.NewHighlighter(config)
	if err != nil {
		fmt.Printf("Error creating highlighter: %v\n", err)
		highlighter = nil
	}

	// Test 1: With highlighter
	fmt.Println("\n=== Test 1: With highlighter ===")
	testRendering(palette, highlighter)

	// Test 2: Without highlighter (nil)
	fmt.Println("\n=== Test 2: Without highlighter ===")
	testRendering(palette, nil)
}

func testRendering(palette *theme.Palette, highlighter *markdown.Highlighter) {
	// Test markdown with Go code block
	md := "Here is some Go code:\n\n```go\nvar wg sync.WaitGroup\nvar mu sync.Mutex\n```\n\nAnd some more text."

	// Parse markdown
	tokens := markdown.ParseWithGoldmark(md)

	// Convert palette to lipgloss style
	style := lipgloss.NewStyle().Foreground(palette.Heading)
	inline := lipgloss.NewStyle().Foreground(palette.InlineCode)

	// Render with highlighting
	rendered := markdown.RenderTokensWithHighlight(tokens, highlighter, style, inline)

	// Check for ANSI codes
	if strings.Contains(rendered, "\x1b[") {
		fmt.Println("Contains ANSI escape codes (highlighting is working)")
		// Show first few lines
		lines := strings.Split(rendered, "\n")
		for i := 0; i < min(5, len(lines)); i++ {
			fmt.Printf("Line %d: %q\n", i, lines[i])
		}
	} else {
		fmt.Println("No ANSI escape codes found")
		// Show the rendered output
		fmt.Println("Rendered output:")
		fmt.Println(rendered)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}