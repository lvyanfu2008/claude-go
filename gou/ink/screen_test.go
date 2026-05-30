package ink

import (
	"image/color"
	"testing"
)

func TestNewScreen(t *testing.T) {
	s := NewScreen(10, 5)
	if s.Width != 10 || s.Height != 5 {
		t.Errorf("expected 10x5, got %dx%d", s.Width, s.Height)
	}
	if len(s.Cells) != 5 || len(s.Cells[0]) != 10 {
		t.Error("cell grid dimensions mismatch")
	}
}

func TestScreenPut(t *testing.T) {
	s := NewScreen(3, 1)
	style := CellStyle{FG: color.RGBA{255, 0, 0, 255}, Bold: true}
	s.PutString(0, 0, "hi", style)
	if s.Cells[0][0].Rune != 'h' || s.Cells[0][1].Rune != 'i' {
		t.Error("PutString mismatch")
	}
	if !s.Cells[0][0].Style.Bold {
		t.Error("expected bold style")
	}
}

func TestScreenResize(t *testing.T) {
	s := NewScreen(2, 2)
	s.PutString(0, 0, "AB", CellStyle{})
	s.Resize(4, 3)
	if s.Width != 4 || s.Height != 3 {
		t.Errorf("expected 4x3, got %dx%d", s.Width, s.Height)
	}
	if s.Cells[0][0].Rune != 'A' || s.Cells[0][1].Rune != 'B' {
		t.Error("resize should preserve existing cells")
	}
}

func TestCellStyleEquals(t *testing.T) {
	a := CellStyle{FG: color.RGBA{255, 0, 0, 255}, Bold: true}
	b := CellStyle{FG: color.RGBA{255, 0, 0, 255}, Bold: true}
	c := CellStyle{FG: color.RGBA{0, 255, 0, 255}, Bold: true}
	if !a.Equals(b) {
		t.Error("identical styles should be equal")
	}
	if a.Equals(c) {
		t.Error("different FG should not be equal")
	}
}
