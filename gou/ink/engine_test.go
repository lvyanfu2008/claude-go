package ink

import (
	"testing"

	"goc/gou/theme"
)

func TestRenderEngine_basic(t *testing.T) {
	pal := theme.ActivePalette()

	root := func(ctx *Context, p Props) VNode {
		return VNode{Type: "Box", Children: []VNode{
			{Type: "Text", Key: "msg", Props: Props{"content": "hello world"}},
		}}
	}

	tree := root(nil, Props{})
	ComputeLayout(&tree, Constraints{MinW: 0, MaxW: 80, MaxH: 24})
	screen := NewScreen(80, 24)
	Rasterize(&tree, screen)

	found := false
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			if screen.Cells[y][x].Rune == 'h' {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected 'hello world' in screen buffer")
	}
	_ = pal
}

func TestRenderLineANSI(t *testing.T) {
	cells := []TermCell{
		{Rune: 'A', Style: CellStyle{Bold: true}},
		{Rune: 'B', Style: CellStyle{Bold: true}},
	}
	line := renderLineANSI(cells)
	if len(line) == 0 {
		t.Error("expected non-empty rendered line")
	}
}
