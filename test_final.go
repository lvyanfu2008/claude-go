package main

import (
	"fmt"
	"os"

	"goc/gou/markdown"
)

func main() {
	config := markdown.DefaultHighlightConfig()
	fmt.Printf("Config: Enabled=%v, StyleName=%s, FormatterName=%s\n",
		config.Enabled, config.StyleName, config.FormatterName)

	highlighter, err := markdown.NewHighlighter(config)
	if err != nil {
		fmt.Printf("Error creating highlighter: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Highlighter created successfully")

	// 测试简单的 Go 代码
	goCode := `var wg sync.WaitGroup`
	highlighted, err := highlighter.HighlightCode(goCode, "go")
	if err != nil {
		fmt.Printf("Error highlighting: %v\n", err)
	} else {
		fmt.Println("Highlighted code:")
		fmt.Println(highlighted)
	}
}