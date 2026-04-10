package markdown

import "regexp"

// MD_SYNTAX_RE mirrors src/components/Markdown.tsx MD_SYNTAX_RE.
// (?m) enables ^ for line-start list markers.
var mdSyntaxRe = regexp.MustCompile(
	"(?m)(?:[#*`" + "`" + `]|[>\-_~]|\n\n|^\d+\. |\n\d+\. )`,
)

// HasMarkdownSyntax mirrors Markdown.tsx hasMarkdownSyntax (first 500 runes).
func HasMarkdownSyntax(s string) bool {
	r := []rune(s)
	if len(r) > 500 {
		s = string(r[:500])
	}
	return mdSyntaxRe.MatchString(s)
}
