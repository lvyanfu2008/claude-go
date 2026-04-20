package main

import (
	"fmt"
	"strings"

	"goc/gou/markdown"
)

func main() {
	// 简单的Markdown示例
	md := `# 代码块渲染演示

这是一个Go函数示例：

` + "```go\n" + `func add(a, b int) int {
	return a + b
}

func main() {
	result := add(3, 5)
	fmt.Println("结果:", result)
}
` + "```\n" + `

这是一个JavaScript示例：

` + "```javascript\n" + `function greet(name) {
	console.log(\`Hello, \${name}!\`)
}

greet("World")
` + "```\n" + `

这是一个Shell命令：

` + "```bash\n" + `#!/bin/bash
echo "当前目录:"
pwd
ls -la
` + "```\n" + `

这是带内联代码的文本：使用 ` + "`fmt.Println()`" + ` 输出内容。`

	fmt.Println("=== Markdown内容 ===")
	fmt.Println(md)
	fmt.Println("\n" + strings.Repeat("=", 50))

	// 解析Markdown
	tokens := markdown.CachedLexer(md)

	fmt.Println("\n=== 解析结果 ===")
	for i, tok := range tokens {
		fmt.Printf("Token %d: %s", i, tok.Type)
		if tok.Type == "code" {
			fmt.Printf(" (语言: %q)", tok.Lang)
		}
		fmt.Println()
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// 纯文本渲染
	fmt.Println("\n=== 纯文本渲染 ===")
	plain := markdown.RenderTokensPlain(tokens)
	fmt.Println(plain)

	fmt.Println("\n" + strings.Repeat("=", 50))

	// 检查高亮器
	config := markdown.DefaultHighlightConfig()
	highlighter, err := markdown.NewHighlighter(config)
	if err != nil {
		fmt.Printf("创建高亮器失败: %v\n", err)
		return
	}

	// 测试单个代码块高亮
	fmt.Println("\n=== Go代码高亮示例 ===")
	goCode := `func add(a, b int) int {
	return a + b
}`
	highlighted, err := highlighter.HighlightCode(goCode, "go")
	if err != nil {
		fmt.Printf("高亮失败: %v\n", err)
	} else {
		fmt.Println(highlighted)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// 显示支持的语言
	fmt.Println("\n=== 支持的语言 ===")
	langs := []string{"go", "javascript", "python", "bash", "sh", "shell", "html", "css", "json", "yaml", "toml", "markdown", "md"}
	for _, lang := range langs {
		if highlighter.SupportsLanguage(lang) {
			fmt.Printf("✓ %s\n", lang)
		} else {
			fmt.Printf("✗ %s\n", lang)
		}
	}
}