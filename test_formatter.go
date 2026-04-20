package main

import (
	"fmt"
	"strings"

	// "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func main() {
	// Test chroma directly
	fmt.Println("Testing chroma formatters...")

	// Try to get terminal256 formatter
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		fmt.Println("ERROR: formatters.Get(\"terminal256\") returned nil")
		// Try terminal8
		formatter = formatters.Get("terminal8")
		if formatter == nil {
			fmt.Println("ERROR: formatters.Get(\"terminal8\") also returned nil")
		} else {
			fmt.Println("Got terminal8 formatter")
		}
	} else {
		fmt.Println("Got terminal256 formatter")
	}

	// Try to get monokai style
	style := styles.Get("monokai")
	if style == nil {
		fmt.Println("ERROR: styles.Get(\"monokai\") returned nil")
		// Try fallback
		style = styles.Fallback
		if style == nil {
			fmt.Println("ERROR: styles.Fallback also nil")
			// Try github
			style = styles.Get("github")
			if style == nil {
				fmt.Println("ERROR: styles.Get(\"github\") also nil")
			} else {
				fmt.Println("Got github style")
			}
		} else {
			fmt.Println("Got fallback style")
		}
	} else {
		fmt.Println("Got monokai style")
	}

	// Try to get Go lexer
	lexer := lexers.Get("go")
	if lexer == nil {
		fmt.Println("ERROR: lexers.Get(\"go\") returned nil")
	} else {
		fmt.Printf("Got Go lexer: %v\n", lexer.Config().Name)
	}

	// Test highlighting if we have all components
	if formatter != nil && style != nil && lexer != nil {
		fmt.Println("\nTesting highlighting...")
		code := `var wg sync.WaitGroup`

		iterator, err := lexer.Tokenise(nil, code)
		if err != nil {
			fmt.Printf("Tokenise error: %v\n", err)
		} else {
			var builder strings.Builder
			err = formatter.Format(&builder, style, iterator)
			if err != nil {
				fmt.Printf("Format error: %v\n", err)
			} else {
				result := builder.String()
				fmt.Printf("Result: %q\n", result)
				if strings.Contains(result, "\x1b[") {
					fmt.Println("Contains ANSI codes - success!")
				} else {
					fmt.Println("No ANSI codes - failure!")
				}
			}
		}
	}

	// List available styles
	fmt.Println("\nAvailable styles:")
	// Note: styles.Registry() might not be available in this version
	// Just try a few common ones
	commonStyles := []string{"monokai", "github", "vs", "pygments", "solarized", "solarized-dark", "solarized-light"}
	for _, name := range commonStyles {
		if s := styles.Get(name); s != nil {
			fmt.Printf("  %s: available\n", name)
		} else {
			fmt.Printf("  %s: NOT available\n", name)
		}
	}

	// List available formatters
	fmt.Println("\nAvailable formatters:")
	commonFormatters := []string{"terminal256", "terminal8", "terminal16", "terminal16m"}
	for _, name := range commonFormatters {
		if f := formatters.Get(name); f != nil {
			fmt.Printf("  %s: available\n", name)
		} else {
			fmt.Printf("  %s: NOT available\n", name)
		}
	}
}