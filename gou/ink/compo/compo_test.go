package compo

import (
	"testing"

	"goc/gou/ink"
	"goc/gou/theme"
)

// testStore implements ink.StoreReader for testing.
type testStore struct {
	messages     []ink.Message
	streamingText string
	streamingTools []ink.StreamingToolUse
	inputValue   string
	cursorPos    int
	isLoading    bool
	width        int
	height       int
	meta         map[string]string
}

func newTestStore() *testStore {
	return &testStore{
		meta: make(map[string]string),
	}
}

func (s *testStore) GetMessages() []ink.Message            { return s.messages }
func (s *testStore) StreamingText() string                  { return s.streamingText }
func (s *testStore) StreamingTools() []ink.StreamingToolUse { return s.streamingTools }
func (s *testStore) InputValue() string                     { return s.inputValue }
func (s *testStore) CursorPos() int                         { return s.cursorPos }
func (s *testStore) IsLoading() bool                        { return s.isLoading }
func (s *testStore) ScrollTop() int                          { return 0 }
func (s *testStore) Width() int                             { return s.width }
func (s *testStore) Height() int                            { return s.height }
func (s *testStore) GetMeta(key string) string {
	if s.meta != nil {
		return s.meta[key]
	}
	return ""
}

// Setters for test setup.
func (s *testStore) AppendMessage(msg ink.Message) {
	s.messages = append(s.messages, msg)
}

func TestMessagesComponent(t *testing.T) {
	store := newTestStore()
	store.width = 80
	store.height = 24

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
	if pal == nil {
		pal = &theme.Palette{}
	}

	ctx := &ink.Context{Theme: pal, Store: store}

	tree := Messages(ctx, ink.Props{})
	ink.ComputeLayout(&tree, ink.Constraints{MaxW: 80, MaxH: 24})

	screen := ink.NewScreen(80, 24)
	ink.Rasterize(&tree, screen)

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
	store := newTestStore()
	store.width = 80
	store.height = 24

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
