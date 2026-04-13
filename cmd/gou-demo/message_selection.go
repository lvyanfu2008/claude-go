package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Message-list selection: viewport-local (row, col) with 0-based pane coords.
// TS parity subset: drag in message pane, Ctrl+C copies when non-empty (terminals that send ctrl+shift+c as ctrl+c).

func orderSelectionEndpoints(ar, ac, fr, fc int) (r1, c1, r2, c2 int) {
	if ar > fr || ar == fr && ac > fc {
		return fr, fc, ar, ac
	}
	return ar, ac, fr, fc
}

func clampSelRect(r1, c1, r2, c2, vpH, bodyCols int) (int, int, int, int, bool) {
	if vpH < 1 || bodyCols < 1 {
		return 0, 0, 0, 0, false
	}
	if r1 < 0 {
		r1 = 0
	}
	if r2 >= vpH {
		r2 = vpH - 1
	}
	if r1 > r2 {
		return 0, 0, 0, 0, false
	}
	bc := bodyCols
	if r1 == r2 {
		if c1 > c2 {
			c1, c2 = c2, c1
		}
		if c1 < 0 {
			c1 = 0
		}
		if c2 >= bc {
			c2 = bc - 1
		}
		if c1 > c2 {
			return 0, 0, 0, 0, false
		}
	} else {
		if c1 < 0 {
			c1 = 0
		}
		if c1 >= bc {
			c1 = bc - 1
		}
		if c2 < 0 {
			c2 = 0
		}
		if c2 >= bc {
			c2 = bc - 1
		}
	}
	return r1, c1, r2, c2, true
}

func (m *model) clearMsgSelection() {
	m.selDragging = false
	m.selHas = false
	m.selAnchorR, m.selAnchorC = 0, 0
	m.selFocusR, m.selFocusC = 0, 0
}

func (m *model) msgSelectionActive() bool {
	return m.selHas
}

// isListViewportScrollKey reports keys that scroll the virtual message list (prompt or transcript pager).
func isListViewportScrollKey(s string) bool {
	switch s {
	case "up", "down", "pgup", "pgdown", "home", "ctrl+home", "end", "ctrl+end",
		"j", "k", "g", "G", "shift+g", "ctrl+u", "ctrl+d", "ctrl+b", "ctrl+f", "b", " ", "ctrl+n", "ctrl+p":
		return true
	default:
		return false
	}
}

func (m *model) cachePaneLinesForSelection(lines []string, bodyCols int) {
	m.lastMsgPaneLines = append([]string(nil), lines...)
	m.lastMsgPaneBodyCols = bodyCols
}

func (m *model) selectionTextForCopy() string {
	if !m.msgSelectionActive() || len(m.lastMsgPaneLines) == 0 {
		return ""
	}
	vp := len(m.lastMsgPaneLines)
	return selectedPlainTextFromPaneLines(m.lastMsgPaneLines, m.lastMsgPaneBodyCols, vp, m.selAnchorR, m.selAnchorC, m.selFocusR, m.selFocusC)
}

func selectedPlainTextFromPaneLines(lines []string, bodyCols, vpH, ar, ac, fr, fc int) string {
	oar, oac, ofr, ofc := orderSelectionEndpoints(ar, ac, fr, fc)
	r1, c1, r2, c2, ok := clampSelRect(oar, oac, ofr, ofc, vpH, bodyCols)
	if !ok {
		return ""
	}
	var b strings.Builder
	for r := r1; r <= r2; r++ {
		if r < 0 || r >= len(lines) {
			continue
		}
		plain := ansi.Strip(lines[r])
		plain = ansi.TruncateWc(plain, bodyCols, "")
		w := ansi.StringWidthWc(plain)
		var seg string
		switch {
		case r == r1 && r == r2:
			seg = ansi.CutWc(plain, c1, c2+1)
		case r == r1:
			seg = ansi.CutWc(plain, c1, w)
		case r == r2:
			seg = ansi.CutWc(plain, 0, c2+1)
		default:
			seg = plain
		}
		seg = strings.TrimRight(seg, " \t")
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(seg)
	}
	return b.String()
}

func applyMsgSelectionVisualHighlight(lines []string, bodyCols, vpH, ar, ac, fr, fc int) []string {
	oar, oac, ofr, ofc := orderSelectionEndpoints(ar, ac, fr, fc)
	r1, c1, r2, c2, ok := clampSelRect(oar, oac, ofr, ofc, vpH, bodyCols)
	if !ok {
		return lines
	}
	hl := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
	out := make([]string, len(lines))
	padTo := func(s string, targetCells int) string {
		n := ansi.StringWidthWc(s)
		if n >= targetCells {
			return s
		}
		return s + strings.Repeat(" ", targetCells-n)
	}
	for i := 0; i < len(lines); i++ {
		ln := lines[i]
		if i < 0 || i >= vpH {
			out[i] = ln
			continue
		}
		plain := ansi.Strip(ln)
		plain = ansi.TruncateWc(plain, bodyCols, "")
		w := ansi.StringWidthWc(plain)
		if i < r1 || i > r2 {
			out[i] = ln
			continue
		}
		var left, mid, right string
		switch {
		case i == r1 && i == r2:
			left = ansi.CutWc(plain, 0, c1)
			mid = ansi.CutWc(plain, c1, c2+1)
			right = ansi.CutWc(plain, c2+1, w)
		case i == r1:
			left = ansi.CutWc(plain, 0, c1)
			mid = ansi.CutWc(plain, c1, w)
		case i == r2:
			mid = ansi.CutWc(plain, 0, c2+1)
			right = ansi.CutWc(plain, c2+1, w)
		default:
			mid = plain
		}
		out[i] = padTo(left+hl.Render(mid)+right, bodyCols)
	}
	return out
}
