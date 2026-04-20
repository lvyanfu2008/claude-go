package main

import (
	"fmt"
	"strings"

	"goc/gou/markdown"
	"charm.land/lipgloss/v2"
)

func main() {
	// Test markdown with Go code block
	md := "Here is some Go code:\n\n```go\nvar wg sync.WaitGroup\nvar mu sync.Mutex\n```\n\nAnd some more text."

	// Parse markdown
	tokens := markdown.ParseWithGoldmark(md)
	fmt.Printf("Parsed %d tokens\n", len(tokens))

	for i, t := range tokens {
		fmt.Printf("Token %d: Type=%s, Lang=%s, Text=%q\n", i, t.Type, t.Lang, t.Text)
		if t.Type == "code" {
			fmt.Printf("  Code block language: %q\n", t.Lang)
		}
	}

	// Create highlighter
	config := markdown.DefaultHighlightConfig()
	highlighter, err := markdown.NewHighlighter(config)
	if err != nil {
		fmt.Printf("Error creating highlighter: %v\n", err)
		return
	}

	// Create a simple style
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // White
	inline := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	// Render with highlighting
	rendered := markdown.RenderTokensWithHighlight(tokens, highlighter, style, inline)

	fmt.Println("\nRendered output:")
	fmt.Println(rendered)

	// Check for ANSI codes
	if strings.Contains(rendered, "\x1b[") {
		fmt.Println("\nContains ANSI escape codes (highlighting is working)")
	} else {
		fmt.Println("\nNo ANSI escape codes found (highlighting may not be working)")
	}

	// Also test a simple highlight directly
	fmt.Println("\nDirect highlight test:")
	goCode := `var wg sync.WaitGroup`
	highlighted, err := highlighter.HighlightCode(goCode, "go")
	if err != nil {
		fmt.Printf("Error highlighting: %v\n", err)
	} else {
		fmt.Printf("Highlighted: %q\n", highlighted)
		if strings.Contains(highlighted, "\x1b[") {
			fmt.Println("Direct highlighting works with ANSI codes")
		}
	}
}