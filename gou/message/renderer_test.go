package message

import (
	"strings"
	"testing"

	"goc/gou/theme"
	"goc/types"
)

func TestUserMessageRenderer_Basic(t *testing.T) {
	r := &UserMessageRenderer{}
	msg := &types.Message{
		Type:    types.MessageTypeUser,
		Content: []byte(`[{"type":"text","text":"hello"}]`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	if !strings.Contains(lines[0], "hello") {
		t.Errorf("expected 'hello' in first line, got: %s", lines[0])
	}
}

func TestUserMessageRenderer_Continuation(t *testing.T) {
	r := &UserMessageRenderer{}
	msg := &types.Message{
		Type:    types.MessageTypeUser,
		Content: []byte(`[{"type":"text","text":"hello"}]`),
	}
	ctx := &RenderContext{
		Width:              80,
		Theme:              theme.ActivePalette(),
		IsUserContinuation: true,
	}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(lines[0], "❯") {
		t.Error("continuation should not have ❯ prefix")
	}
}

func TestAssistantMessageRenderer_Basic(t *testing.T) {
	r := &AssistantMessageRenderer{}
	msg := &types.Message{
		Type:    types.MessageTypeAssistant,
		Content: []byte(`[{"type":"text","text":"Hello **world**"}]`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
}

func TestSystemMessageRenderer_Informational(t *testing.T) {
	r := &SystemMessageRenderer{}
	s := "informational"
	msg := &types.Message{
		Type:    types.MessageTypeSystem,
		Subtype: &s,
		Content: []byte(`"test info"`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lines[0], "ℹ") {
		t.Errorf("expected ℹ prefix, got: %s", lines[0])
	}
}
