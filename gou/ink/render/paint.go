package render

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

type DiffEngine struct {
	cursorX, cursorY int
}

func NewDiffEngine() *DiffEngine {
	return &DiffEngine{}
}

func (d *DiffEngine) Generate(prev, curr *Screen) string {
	var buf strings.Builder
	w := curr.Width
	h := curr.Height

	for row := 0; row < h; row++ {
		prevRow := safeRow(prev, row, w)
		currRow := curr.Cells[row]
		if rowsEqual(prevRow, currRow) {
			continue
		}

		d.moveTo(&buf, row, 0)

		var run []TermCell
		var runStyle CellStyle
		inRun := false

		for col := 0; col < w; col++ {
			prevCell := safeCell(prev, row, col)
			currCell := curr.Cells[row][col]

			// Continuation cells for double-width characters — terminal handles these.
			// Emitting spaces here would overwrite the second half of CJK glyphs.
			if currCell.Rune == 0 {
				if inRun {
					d.flushRun(&buf, run, runStyle)
					run = nil
					inRun = false
				}
				continue
			}

			if prevCell.Equals(currCell) {
				if inRun {
					d.flushRun(&buf, run, runStyle)
					run = nil
					inRun = false
				}
				continue
			}

			if !inRun {
				// Position cursor at the changed column if needed
				if d.cursorX != col {
					d.writeCursorTo(&buf, d.cursorY, col)
				}
				runStyle = currCell.Style
				inRun = true
			} else if !currCell.Style.Equals(runStyle) {
				d.flushRun(&buf, run, runStyle)
				run = nil
				runStyle = currCell.Style
			}
			run = append(run, currCell)
		}
		if inRun {
			d.flushRun(&buf, run, runStyle)
		}
		buf.WriteString(EraseToEnd())
	}
	buf.WriteString(SgrReset())
	return buf.String()
}

func (d *DiffEngine) writeCursorTo(buf *strings.Builder, row, col int) {
	d.cursorY = row
	d.cursorX = col
	buf.WriteString(CursorTo(row, col))
}

func (d *DiffEngine) moveTo(buf *strings.Builder, row, col int) {
	buf.WriteString(CursorTo(row, col))
	d.cursorY = row
	d.cursorX = col
}

func (d *DiffEngine) flushRun(buf *strings.Builder, run []TermCell, style CellStyle) {
	if len(run) == 0 {
		return
	}
	sgr := StyleToSGR(style)
	if sgr != "" {
		buf.WriteString(sgr)
	}
	for _, c := range run {
		if c.Rune == 0 {
			buf.WriteByte(' ')
			d.cursorX++
		} else {
			buf.WriteRune(c.Rune)
			d.cursorX += runewidth.RuneWidth(c.Rune)
		}
	}
	buf.WriteString(SgrReset())
}

func rowsEqual(a, b []TermCell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

func safeRow(s *Screen, row, width int) []TermCell {
	if s == nil || row >= len(s.Cells) {
		return make([]TermCell, width)
	}
	return s.Cells[row]
}

func safeCell(s *Screen, row, col int) TermCell {
	if s == nil || row >= len(s.Cells) || col >= len(s.Cells[row]) {
		return TermCell{}
	}
	return s.Cells[row][col]
}
