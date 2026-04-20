package main

import (
	"fmt"
	"strings"

	"goc/gou/markdown"
	"charm.land/lipgloss/v2"
)

func main() {
	// 测试代码块渲染
	md := `# 代码块示例

这是一个Go代码块：

` + "```go\n" + `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
` + "```\n" + `
这是一个Python代码块：

` + "```python\n" + `def hello():
    print("Hello from Python!")
    
hello()
` + "```\n" + `
这是没有语言指定的代码块：

` + "```\n" + `plain text code block
` + "```\n" + `
这是带内联代码的段落：使用 ` + "`fmt.Println()`" + ` 函数。`

	// 解析Markdown
	tokens := markdown.CachedLexer(md)
	
	fmt.Println("=== 原始Markdown ===")
	fmt.Println(md)
	fmt.Println("\n=== 解析后的Tokens ===")
	for i, tok := range tokens {
		fmt.Printf("Token %d: Type=%q, Lang=%q\n", i, tok.Type, tok.Lang)
		if tok.Type == "code" {
			fmt.Printf("  Code content (first 50 chars): %.50s\n", tok.Text)
		}
	}
	
	fmt.Println("\n=== 纯文本渲染 ===")
	plain := markdown.RenderTokensPlain(tokens)
	fmt.Println(plain)
	
	fmt.Println("\n=== 带高亮的渲染 ===")
	// 创建高亮器
	config := markdown.DefaultHighlightConfig()
	highlighter, err := markdown.NewHighlighter(config)
	if err != nil {
		fmt.Printf("创建高亮器失败: %v\n", err)
		return
	}
	
	theme := lipgloss.NewStyle()
	highlighted := markdown.RenderTokensWithHighlight(tokens, highlighter, theme)
	
	// 显示前500个字符
	if len(highlighted) > 500 {
		fmt.Printf("%.500s...\n", highlighted)
	} else {
		fmt.Println(highlighted)
	}
	
	// 检查高亮器支持的语言
	fmt.Println("\n=== 高亮器支持的语言 ===")
	languages := []string{"go", "python", "javascript", "java", "c", "cpp", "rust", "bash"}
	for _, lang := range languages {
		if highlighter.SupportsLanguage(lang) {
			fmt.Printf("✓ 支持: %s\n", lang)
		} else {
			fmt.Printf("✗ 不支持: %s\n", lang)
		}
	}
	
	// 测试代码高亮
	fmt.Println("\n=== 测试Go代码高亮 ===")
	goCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
	
	highlightedGo, err := highlighter.HighlightCode(goCode, "go")
	if err != nil {
		fmt.Printf("高亮失败: %v\n", err)
	} else {
		// 显示前200个字符
		if len(highlightedGo) > 200 {
			fmt.Printf("%.200s...\n", highlightedGo)
		} else {
			fmt.Println(highlightedGo)
		}
		
		// 检查是否包含ANSI转义序列
		if strings.Contains(highlightedGo, "\x1b[") || strings.Contains(highlightedGo, "\033[") {
			fmt.Println("✓ 包含ANSI转义序列（颜色）")
		} else {
			fmt.Println("✗ 不包含ANSI转义序列")
		}
	}
}