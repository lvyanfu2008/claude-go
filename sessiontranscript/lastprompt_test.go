package sessiontranscript

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"goc/types"
)

func TestExtractTag_simple(t *testing.T) {
	got := extractTag(`  <command-name>/m</command-name>  `, "command-name")
	if got != "/m" {
		t.Fatalf("got %q", got)
	}
}

func TestFirstMeaningfulUser_skipsMetaAndIdePrefix(t *testing.T) {
	meta := true
	ms := []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", IsMeta: &meta, Message: json.RawMessage(`{"role":"user","content":"hidden"}`)},
		{Type: types.MessageTypeUser, UUID: "u2", Message: json.RawMessage(`{"role":"user","content":[{"type":"text","text":"<ide_opened_file path=\"a\" />"},{"type":"text","text":"real prompt"}]}`)},
	}
	got := FirstMeaningfulUserMessageTextContent(ms)
	if got != "real prompt" {
		t.Fatalf("got %q", got)
	}
}

func TestFirstMeaningfulUser_textBlocksArray(t *testing.T) {
	ms := []types.Message{{
		Type: types.MessageTypeUser,
		UUID: "u1",
		Message: json.RawMessage(`{"role":"user","content":[{"type":"text","text":"<ide_selection path=\"x\" />"},{"type":"text","text":"second"}]}`),
	}}
	got := FirstMeaningfulUserMessageTextContent(ms)
	if got != "second" {
		t.Fatalf("got %q", got)
	}
}

func TestFlattenLastPromptCache_truncates(t *testing.T) {
	s := strings.Repeat("a", 250)
	got := FlattenLastPromptCache(s)
	if utf8.RuneCountInString(got) != 201 { // 200 + … (one rune)
		t.Fatalf("rune len %d: %q", utf8.RuneCountInString(got), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("want ellipsis suffix: %q", got)
	}
}
