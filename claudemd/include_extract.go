package claudemd

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var includeRegex = regexp.MustCompile(`(?:^|\s)@((?:[^\s\\]|\\ )+)`)
var invalidAtStart = regexp.MustCompile(`^[#%^&*()]+`)
var validPathStart = regexp.MustCompile(`^[a-zA-Z0-9._-]`)

// ExtractIncludePathsFromMarkdown mirrors extractIncludePathsFromTokens behavior using goldmark AST.
func ExtractIncludePathsFromMarkdown(src []byte, basePath string) []string {
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(src))
	baseDir := filepath.Dir(basePath)
	seen := map[string]struct{}{}
	var order []string
	add := func(abs string) {
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		order = append(order, abs)
	}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.CodeSpan:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			t := n.(*ast.Text)
			txt := string(t.Text(src))
			for _, p := range pathsFromIncludeText(txt, baseDir) {
				add(p)
			}
		case *ast.RawHTML:
			r := n.(*ast.RawHTML)
			raw := string(r.Text(src))
			stripped := htmlCommentSpan.ReplaceAllString(raw, "")
			if strings.TrimSpace(stripped) == "" {
				return ast.WalkContinue, nil
			}
			for _, p := range pathsFromIncludeText(stripped, baseDir) {
				add(p)
			}
		}
		return ast.WalkContinue, nil
	})
	return order
}

func pathsFromIncludeText(textContent, baseDir string) []string {
	var out []string
	for _, m := range includeRegex.FindAllStringSubmatchIndex(textContent, -1) {
		if len(m) < 4 {
			continue
		}
		path := textContent[m[2]:m[3]]
		path = strings.ReplaceAll(path, `\ `, " ")
		if i := strings.Index(path, "#"); i >= 0 {
			path = path[:i]
		}
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		valid := strings.HasPrefix(path, "./") || strings.HasPrefix(path, "~/") ||
			(strings.HasPrefix(path, "/") && path != "/") ||
			(!strings.HasPrefix(path, "@") && !invalidAtStart.MatchString(path) &&
				len(path) > 0 && validPathStart.MatchString(path[:1]))
		if !valid {
			continue
		}
		abs := ExpandPath(path, baseDir)
		out = append(out, abs)
	}
	return out
}
