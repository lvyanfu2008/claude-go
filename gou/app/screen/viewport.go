package screen

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/layout"
)

// ScrollbarThumb returns [start, length) in viewport rows [0, vpH) for a proportional thumb.
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

// JoinMessagePaneLinesWithScrollbar pads each line to bodyCols cells and appends one scrollbar
// column when barW==1.
func JoinMessagePaneLinesWithScrollbar(lines []string, bodyCols, vpH, totalH, scrollTop int, barW int) string {
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
		ch := "│"
		if r >= thumbStart && r < thumbStart+thumbLen {
			out = append(out, ln+thumbStyle.Render("┃"))
		} else {
			out = append(out, ln+trackStyle.Render(ch))
		}
	}
	return strings.Join(out, "\n")
}

// ApplyMessagePaneGutter adds uniform 2-space indent (delegates to the config package logic).
func ApplyMessagePaneGutter(block string, cols int) string {
	// Use a simpler approach: prepend "  " to each line.
	if block == "" || cols < 1 {
		return block
	}
	lines := strings.Split(block, "\n")
	for i, ln := range lines {
		lines[i] = "  " + ln
	}
	return strings.Join(lines, "\n")
}
