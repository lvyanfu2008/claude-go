// Package markdown mirrors src/components/Markdown.tsx lexer/cache concepts (marked Token subset).
package markdown

// Token is a block-level unit for terminal layout (marked uses richer Token trees).
// Mirrors common marked token type strings: heading, paragraph, code, list_item, blockquote, hr.
type Token struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Raw   string `json:"raw,omitempty"`
	Lang  string `json:"lang,omitempty"`
	Level int    `json:"level,omitempty"` // heading depth (1–6)
}
