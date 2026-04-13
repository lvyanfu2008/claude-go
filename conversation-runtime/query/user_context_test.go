package query

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestAppendSystemContext(t *testing.T) {
	sys := AsSystemPrompt([]string{"A", "B"})
	got := AppendSystemContext(sys, map[string]string{"gitStatus": "clean"})
	if len(got) != 3 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0] != "A" || got[1] != "B" {
		t.Fatalf("prefix %#v", got)
	}
	if got[2] != "gitStatus: clean" {
		t.Fatalf("context line %q", got[2])
	}
}

func TestAppendSystemContext_gitStatusBeforeCacheBreaker(t *testing.T) {
	sys := AsSystemPrompt([]string{"base"})
	got := AppendSystemContext(sys, map[string]string{
		"cacheBreaker": "[x]",
		"gitStatus":    "clean",
	})
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[1] != "gitStatus: clean\ncacheBreaker: [x]" {
		t.Fatalf("want TS object order, got %q", got[1])
	}
}

func TestPrependUserContextSkipsWhenEmpty(t *testing.T) {
	base := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Message: json.RawMessage(`{"role":"user","content":"x"}`)}}
	got := PrependUserContext(base, nil)
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
}

func TestPrependUserContextInsertsMeta(t *testing.T) {
	base := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Message: json.RawMessage(`{"role":"user","content":"x"}`)}}
	got := PrependUserContext(base, map[string]string{"cwd": "/tmp"})
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Type != types.MessageTypeUser || got[0].IsMeta == nil || !*got[0].IsMeta {
		t.Fatalf("first msg %#v", got[0])
	}
	if got[1].UUID != "1" {
		t.Fatalf("second uuid %q", got[1].UUID)
	}
}

func TestPrependUserContextSkipTestFlag(t *testing.T) {
	SkipUserContextInTest = true
	t.Cleanup(func() { SkipUserContextInTest = false })
	base := []types.Message{{Type: types.MessageTypeUser, UUID: "1"}}
	got := PrependUserContext(base, map[string]string{"k": "v"})
	if len(got) != 1 {
		t.Fatalf("len %d", len(got))
	}
}
