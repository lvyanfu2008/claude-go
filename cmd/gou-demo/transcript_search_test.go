package main

import (
	"strings"
	"testing"

	"goc/types"
)

func TestPlainMessageSearchText_collapsedPaths(t *testing.T) {
	msg := types.Message{
		Type:          types.MessageTypeCollapsedReadSearch,
		ReadFilePaths: []string{"src/foo.go"},
		SearchArgs:    []string{"TODO"},
	}
	s := plainMessageSearchText(msg)
	if !strings.Contains(s, "src/foo.go") || !strings.Contains(s, "todo") {
		t.Fatalf("got %q", s)
	}
}
