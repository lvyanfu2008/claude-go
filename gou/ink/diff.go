package ink

import "strings"

type DiffEngine struct {
	cursorX, cursorY int
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

			if prevCell.Equals(currCell) {
				if inRun {
					d.flushRun(&buf, run, runStyle)
					run = nil
					inRun = false
				}
				d.cursorX = col + 1
				continue
			}

			if !inRun {
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
	}
	buf.WriteString(sgrReset())
	return buf.String()
}

func (d *DiffEngine) moveTo(buf *strings.Builder, row, col int) {
	buf.WriteString(cursorTo(row, col))
	d.cursorY = row
	d.cursorX = col
}

func (d *DiffEngine) flushRun(buf *strings.Builder, run []TermCell, style CellStyle) {
	if len(run) == 0 {
		return
	}
	sgr := styleToSGR(style)
	if sgr != "" {
		buf.WriteString(sgr)
	}
	for _, c := range run {
		if c.Rune == 0 {
			buf.WriteByte(' ')
		} else {
			buf.WriteRune(c.Rune)
		}
	}
	buf.WriteString(sgrReset())
	d.cursorX += len(run)
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
