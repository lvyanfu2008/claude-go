// Package markdown mirrors src/components/Markdown.tsx lexer/cache concepts (marked Token subset).
package markdown

// InlineSegment is a run of plain text or inline code inside a paragraph/list_item/blockquote
// (TS-style inline `code` color + strong/emphasis for terminal).
type InlineSegment struct {
	Code   bool   `json:"code,omitempty"`
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
	Text   string `json:"text,omitempty"`
}

// Token is a block-level unit for terminal layout (marked uses richer Token trees).
// Mirrors common marked token type strings: heading, paragraph, code, list_item, blockquote, hr.
type Token struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Raw   string `json:"raw,omitempty"`
	Lang  string `json:"lang,omitempty"`
	Level int    `json:"level,omitempty"` // heading depth (1–6)
	// ListOrdered/ListIndex: ordered list items use "N. " prefix in the TUI; unordered uses "- ".
	ListOrdered bool `json:"list_ordered,omitempty"`
	ListIndex   int  `json:"list_index,omitempty"`
	// ListIndent: leading spaces before marker (nested sublists).
	ListIndent int `json:"list_indent,omitempty"`
	// ListContinuation: extra paragraphs in the same list item (no marker; aligned under text).
	ListContinuation bool `json:"list_continuation,omitempty"`
	// Segments is set for paragraph/list_item/blockquote when inline styling exists (code / strong / emphasis).
	Segments []InlineSegment `json:"segments,omitempty"`
}
