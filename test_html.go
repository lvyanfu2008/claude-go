package main

import (
	"fmt"
	
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func main() {
	content := "<!-- comment --> @./file.md"
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader([]byte(content)))
	
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			fmt.Printf("Node type: %T\n", n)
			if textNode, ok := n.(*ast.Text); ok {
				fmt.Printf("Text: %q\n", string(textNode.Text([]byte(content))))
			}
			if htmlNode, ok := n.(*ast.RawHTML); ok {
				fmt.Printf("RawHTML: %q\n", string(htmlNode.Text([]byte(content))))
			}
		}
		return ast.WalkContinue, nil
	})
}
