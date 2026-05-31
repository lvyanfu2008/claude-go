package message

import (
	"strings"

	"charm.land/lipgloss/v2"
	"goc/gou/layout"
)

// ListMouseWheelStep returns how many terminal rows one wheel notch moves.
func ListMouseWheelStep(vpH int) int {
	if vpH < 1 {
		return 1
	}
	return max(1, vpH/12)
}

// ScrollbarThumb returns [start, length) in viewport rows [0, vpH) for a
// proportional scrollbar thumb.
func ScrollbarThumb(vpH, totalH, scrollTop int) (start, length int) {
	if vpH < 1 {
		return 0, 0
	}
	if totalH <= vpH {
		return 0, vpH
	}
	maxTop := totalH - vpH
	st := scrollTop
	if st < 0 {
		st = 0
	}
	if st > maxTop {
		st = maxTop
	}
	length = max(1, vpH*vpH/max(1, totalH))
	if length > vpH {
		length = vpH
	}
	start = (st * (vpH - length)) / max(1, maxTop)
	if start < 0 {
		start = 0
	}
	if start+length > vpH {
		start = vpH - length
	}
	return start, length
}

// JoinLinesWithScrollbar pads each line to bodyCols cells and appends one
// scrollbar column when barW==1.
func JoinLinesWithScrollbar(lines []string, bodyCols, vpH, totalH, scrollTop int, barW int) string {
	if barW != 1 || vpH < 1 {
		return strings.Join(lines, "\n")
	}
	thumbStart, thumbLen := ScrollbarThumb(vpH, totalH, scrollTop)
	trackStyle := lipgloss.NewStyle().Faint(true)
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	out := make([]string, 0, vpH)
	for r := 0; r < vpH; r++ {
		ln := ""
		if r < len(lines) {
			ln = lines[r]
		}
		pad := bodyCols - layout.VisualWidth(ln)
		if pad > 0 {
			ln += strings.Repeat(" ", pad)
		}
		if r >= thumbStart && r < thumbStart+thumbLen {
			out = append(out, ln+thumbStyle.Render("┃"))
		} else {
			out = append(out, ln+trackStyle.Render("│"))
		}
	}
	return strings.Join(out, "\n")
}
