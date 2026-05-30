package compo

import (
	"testing"

	"goc/gou/ink"
	"goc/gou/theme"
)

func TestMessagesComponent(t *testing.T) {
	store := ink.NewStore()
	store.Width = 80
	store.Height = 24

	store.AppendMessage(ink.Message{
		UUID: "u1", Type: "user",
		ContentBlocks: []ink.ContentBlock{
			{Type: "text", Content: "read main.go"},
		},
	})
	store.AppendMessage(ink.Message{
		UUID: "a1", Type: "assistant",
		ContentBlocks: []ink.ContentBlock{
			{Type: "tool_use", Name: "Read", State: "resolved",
				Result: &ink.ContentBlock{Type: "tool_result", Content: "package main"},
			},
		},
	})
	store.AppendMessage(ink.Message{
		UUID: "a2", Type: "assistant",
		ContentBlocks: []ink.ContentBlock{
			{Type: "text", Content: "The file contains a main function."},
		},
	})

	pal := theme.ActivePalette()
	// Try to get a palette
	if pal == nil {
		// fallback: use Default if that exists
		pal = &theme.Palette{}
	}

	ctx := &ink.Context{Theme: pal, Store: store}

	tree := Messages(ctx, ink.Props{})
	ink.ComputeLayout(&tree, ink.Constraints{MaxW: 80, MaxH: 24})

	screen := ink.NewScreen(80, 24)
	ink.Rasterize(&tree, screen)

	// Verify user message text appears
	found := false
	for y := 0; y < 24; y++ {
		for x := 0; x < 75; x++ {
			if screen.Cells[y][x].Rune == 'r' &&
				screen.Cells[y][x+1].Rune == 'e' &&
				screen.Cells[y][x+2].Rune == 'a' &&
				screen.Cells[y][x+3].Rune == 'd' {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected 'read main.go' to appear in rendered output")
	}
}

func TestFullREPLRender(t *testing.T) {
	store := ink.NewStore()
	store.Width = 80
	store.Height = 24

	pal := theme.ActivePalette()
	if pal == nil {
		pal = &theme.Palette{}
	}

	store.AppendMessage(ink.Message{
		UUID: "u1", Type: "user",
		ContentBlocks: []ink.ContentBlock{{Type: "text", Content: "hello"}},
	})

	ctx := &ink.Context{Theme: pal, Store: store}
	tree := REPL(ctx, ink.Props{})
	ink.ComputeLayout(&tree, ink.Constraints{MaxW: 80, MaxH: 24})
	screen := ink.NewScreen(80, 24)
	ink.Rasterize(&tree, screen)

	// Should have rendered without error and have some content
	hasContent := false
	for y := 0; y < 24 && !hasContent; y++ {
		for x := 0; x < 80 && !hasContent; x++ {
			if screen.Cells[y][x].Rune != 0 {
				hasContent = true
			}
		}
	}
	if !hasContent {
		t.Error("expected some content in rendered REPL")
	}
}
