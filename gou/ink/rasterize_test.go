package ink

import "testing"

func TestRasterizeText(t *testing.T) {
	node := &VNode{
		Type: "Text",
		Props: Props{"content": "hello", "bold": true},
		Layout: LayoutResult{X: 0, Y: 0, W: 5, H: 1},
	}
	screen := NewScreen(10, 3)
	Rasterize(node, screen)
	if screen.Cells[0][0].Rune != 'h' {
		t.Error("expected 'h' at (0,0)")
	}
	if !screen.Cells[0][0].Style.Bold {
		t.Error("expected bold")
	}
}

func TestRasterizeBox(t *testing.T) {
	box := &VNode{
		Type: "Box",
		Layout: LayoutResult{X: 0, Y: 0, W: 10, H: 3},
		Children: []VNode{
			{Type: "Text", Props: Props{"content": "line1"}, Layout: LayoutResult{X: 0, Y: 0, W: 5, H: 1}},
			{Type: "Text", Props: Props{"content": "line2"}, Layout: LayoutResult{X: 0, Y: 1, W: 5, H: 1}},
		},
	}
	screen := NewScreen(10, 3)
	Rasterize(box, screen)
	if screen.Cells[0][0].Rune != 'l' {
		t.Error("expected 'l' at (0,0)")
	}
	if screen.Cells[1][0].Rune != 'l' {
		t.Error("expected 'l' at (1,0)")
	}
}
