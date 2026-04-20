package main

import (
	"fmt"
	"strings"

	"goc/gou/markdown"
	"charm.land/lipgloss/v2"
)

func main() {
	// 测试...的渲染
	testCases := []string{
		"这是一个测试...看看效果",
		"函数调用: `fmt.Println(...)`",
		"代码块中的...:\n```go\nfunc test(...interface{}) {\n    // ...\n}\n```",
		"单独的...",
		"**粗体...** 和 *斜体...*",
	}

	for i, md := range testCases {
		fmt.Printf("\n=== 测试用例 %d ===\n", i+1)
		fmt.Printf("Markdown: %s\n", md)
		
		// 解析Markdown
		tokens := markdown.CachedLexer(md)
		
		// 纯文本渲染
		plain := markdown.RenderTokensPlain(tokens)
		fmt.Printf("纯文本渲染:\n%s\n", plain)
		
		// 带高亮的渲染
		config := markdown.DefaultHighlightConfig()
		highlighter, err := markdown.NewHighlighter(config)
		if err != nil {
			fmt.Printf("创建高亮器失败: %v\n", err)
			continue
		}
		
		theme := lipgloss.NewStyle()
		highlighted := markdown.RenderTokensWithHighlight(tokens, highlighter, theme)
		
		// 检查highlighted中是否包含ANSI转义序列
		hasAnsi := strings.Contains(highlighted, "\x1b[") || strings.Contains(highlighted, "\033[")
		fmt.Printf("带高亮渲染 (包含ANSI: %v):\n", hasAnsi)
		
		// 显示前200个字符
		if len(highlighted) > 200 {
			fmt.Printf("%.200s...\n", highlighted)
		} else {
			fmt.Println(highlighted)
		}
		
		// 检查token信息
		fmt.Println("Token分析:")
		for j, tok := range tokens {
			fmt.Printf("  Token %d: Type=%q", j, tok.Type)
			if tok.Type == "code" {
				fmt.Printf(", Lang=%q", tok.Lang)
			}
			if len(tok.Segments) > 0 {
				fmt.Printf(", Segments=%d", len(tok.Segments))
				for k, seg := range tok.Segments {
					fmt.Printf("\n    Segment %d: Text=%q, Bold=%v, Italic=%v, Code=%v", 
						k, seg.Text, seg.Bold, seg.Italic, seg.Code)
				}
			}
			fmt.Println()
		}
	}
}