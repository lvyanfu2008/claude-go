package ink

import "image/color"

type CellStyle struct {
	FG, BG                            color.Color
	Bold, Dim, Italic, Underline      bool
}

func (s CellStyle) Equals(o CellStyle) bool {
	return s.FG == o.FG && s.BG == o.BG &&
		s.Bold == o.Bold && s.Dim == o.Dim &&
		s.Italic == o.Italic && s.Underline == o.Underline
}

type TermCell struct {
	Rune  rune
	Style CellStyle
}

func (c TermCell) Equals(o TermCell) bool {
	return c.Rune == o.Rune && c.Style.Equals(o.Style)
}

type Screen struct {
	Width, Height int
	Cells         [][]TermCell
}

func NewScreen(w, h int) *Screen {
	cells := make([][]TermCell, h)
	for i := range cells {
		cells[i] = make([]TermCell, w)
	}
	return &Screen{Width: w, Height: h, Cells: cells}
}

func (s *Screen) Resize(w, h int) {
	newCells := make([][]TermCell, h)
	for i := range newCells {
		newCells[i] = make([]TermCell, w)
		if i < len(s.Cells) {
			copy(newCells[i], s.Cells[i])
		}
	}
	s.Cells = newCells
	s.Width = w
	s.Height = h
}

func (s *Screen) Clear() {
	for y := range s.Cells {
		for x := range s.Cells[y] {
			s.Cells[y][x] = TermCell{}
		}
	}
}

func (s *Screen) Put(x, y int, r rune, style CellStyle) {
	if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
		return
	}
	s.Cells[y][x] = TermCell{Rune: r, Style: style}
}

func (s *Screen) PutString(x, y int, str string, style CellStyle) {
	for i, r := range str {
		s.Put(x+i, y, r, style)
	}
}
