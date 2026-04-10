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

func TestNormalizeStreamingForLexer(t *testing.T) {
	s := "```go\nfmt.Println("
	if strings.Count(NormalizeStreamingForLexer(s), "```")%2 != 0 {
		t.Fatal("should balance fences")
	}
}
