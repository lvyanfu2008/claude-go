package markdown

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	SetGlobalCacheForTest(NewTokenCache(TokenCacheMax))
	os.Exit(code)
}

func TestHasMarkdownSyntax(t *testing.T) {
	if HasMarkdownSyntax("plain sentence") {
		t.Fatal("plain should miss")
	}
	if !HasMarkdownSyntax("## hi") {
		t.Fatal("heading should hit")
	}
	if !HasMarkdownSyntax("a\n\nb") {
		t.Fatal("double newline should hit")
	}
}

func TestCachedLexer_plain(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("no md here")
	if len(toks) != 1 || toks[0].Type != "paragraph" {
		t.Fatalf("%+v", toks)
	}
}

func TestCachedLexer_heading(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("## Title\n\nbody")
	if len(toks) < 2 {
		t.Fatalf("want heading+para, got %+v", toks)
	}
	if toks[0].Type != "heading" || toks[0].Level != 2 || !strings.Contains(toks[0].Text, "Title") {
		t.Fatalf("%+v", toks[0])
	}
}

func TestCachedLexer_heading_inline_bold(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("## **bold** end")
	if len(toks) != 1 || toks[0].Type != "heading" || len(toks[0].Segments) < 2 {
		t.Fatalf("want heading with segments, got %+v", toks[0])
	}
	var sawBold bool
	for _, s := range toks[0].Segments {
		if s.Bold && strings.Contains(s.Text, "bold") {
			sawBold = true
		}
	}
	if !sawBold {
		t.Fatalf("want bold segment in heading, got %+v", toks[0].Segments)
	}
}

func TestCachedLexer_fence(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("```go\nx := 1\n```")
	var code *Token
	for i := range toks {
		if toks[i].Type == "code" {
			code = &toks[i]
			break
		}
	}
	if code == nil || code.Lang != "go" || !strings.Contains(code.Text, "x := 1") {
		t.Fatalf("%+v", toks)
	}
}

func TestCachedLexer_inline_code_segments(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("让我查看一下 `openai_stream_gate.go` 文件来了解")
	if len(toks) != 1 || toks[0].Type != "paragraph" {
		t.Fatalf("%+v", toks)
	}
	if len(toks[0].Segments) < 2 || !strings.Contains(toks[0].Text, "openai_stream_gate.go") {
		t.Fatalf("want segments + full text, got %+v", toks[0])
	}
	var code bool
	for _, s := range toks[0].Segments {
		if s.Code && strings.Contains(s.Text, "openai_stream_gate.go") {
			code = true
		}
	}
	if !code {
		t.Fatalf("want inline code segment, got %+v", toks[0].Segments)
	}
}

func TestCachedLexer_bold_italic_segments(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("**bold** and *italic*")
	if len(toks) != 1 || toks[0].Type != "paragraph" {
		t.Fatalf("%+v", toks)
	}
	var sawBold, sawItalic bool
	for _, s := range toks[0].Segments {
		if s.Bold && strings.Contains(s.Text, "bold") {
			sawBold = true
		}
		if s.Italic && strings.Contains(s.Text, "italic") {
			sawItalic = true
		}
	}
	if !sawBold || !sawItalic {
		t.Fatalf("want bold+italic segments, got %+v", toks[0].Segments)
	}
}

func TestCachedLexer_ordered_list_index(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("1. first\n2. second")
	if len(toks) != 2 {
		t.Fatalf("want 2 list_item, got %+v", toks)
	}
	if !toks[0].ListOrdered || toks[0].ListIndex != 1 || !toks[1].ListOrdered || toks[1].ListIndex != 2 {
		t.Fatalf("ordered indices: %+v, %+v", toks[0], toks[1])
	}
}

func TestCachedLexer_nested_list_under_ordered(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	// Tight list: first line is a TextBlock + nested List (goldmark), not a Paragraph.
	md := "1. intro line\n    - nested a\n    - nested b\n2. second top"
	toks := CachedLexer(md)
	if len(toks) < 4 {
		t.Fatalf("want ordered item + 2 nested + second ordered, got %d: %+v", len(toks), toks)
	}
	if toks[0].Type != "list_item" || toks[0].ListIndex != 1 || toks[0].ListIndent != 0 {
		t.Fatalf("first token: %+v", toks[0])
	}
	if toks[1].ListIndent != 2 || toks[1].ListIndex != 0 || toks[1].ListOrdered {
		t.Fatalf("nested a: %+v", toks[1])
	}
	if toks[2].ListIndent != 2 || !strings.Contains(toks[2].Text, "nested b") {
		t.Fatalf("nested b: %+v", toks[2])
	}
	if toks[3].ListIndex != 2 || toks[3].ListIndent != 0 {
		t.Fatalf("second top: %+v", toks[3])
	}
}

func TestCachedLexer_blockquote_inline_code_segments(t *testing.T) {
	SetGlobalCacheForTest(NewTokenCache(10))
	toks := CachedLexer("> see `code` here")
	if len(toks) != 1 || toks[0].Type != "blockquote" {
		t.Fatalf("%+v", toks)
	}
	if len(toks[0].Segments) < 2 {
		t.Fatalf("want segments for code in blockquote, got %+v", toks[0])
	}
	var code bool
	for _, s := range toks[0].Segments {
		if s.Code && s.Text == "code" {
			code = true
		}
	}
	if !code {
		t.Fatalf("want inline code segment in blockquote, got %+v", toks[0].Segments)
	}
}

func TestNormalizeStreamingForLexer(t *testing.T) {
	s := "```go\nfmt.Println("
	if strings.Count(NormalizeStreamingForLexer(s), "```")%2 != 0 {
		t.Fatal("should balance fences")
	}
}
