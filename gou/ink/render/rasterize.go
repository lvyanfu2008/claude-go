package render

import (
	"image/color"
	"github.com/mattn/go-runewidth"

	"goc/gou/ink/vdom"
)

func init() {
	// Box-drawing characters (U+2500+) are single-width in modern terminals.
	// Setting EastAsianWidth=false keeps them single-width while preserving
	// double-width for CJK characters like 你好.
	runewidth.DefaultCondition.EastAsianWidth = false
}

func Rasterize(node *vdom.VNode, screen *Screen) {
	rasterizeNode(node, screen, 0, 0)
}

// rasterizeNode walks the VNode tree and renders each node at its screen position,
// accumulating offsets from parent containers so that Layout.X/Y (which are relative
// to the parent) are translated to absolute screen coordinates.
func rasterizeNode(node *vdom.VNode, screen *Screen, offX, offY int) {
	absX := node.Layout.X + offX
	absY := node.Layout.Y + offY

	switch node.Type {
	case "Text":
		rasterizeTextAt(node, screen, absX, absY)
	case "Box":
		for i := range node.Children {
			rasterizeNode(&node.Children[i], screen, absX, absY)
		}
	case "ScrollBox", "VirtualList":
		visStart := node.Layout.VisibleRange[0]
		visEnd := node.Layout.VisibleRange[1]
		if visEnd > len(node.Children) {
			visEnd = len(node.Children)
		}
		for i := visStart; i < visEnd; i++ {
			rasterizeNode(&node.Children[i], screen, absX, absY)
		}
	}
}

// rasterizeTextAt renders a Text node at the given absolute screen position.
// It is ANSI-aware: embedded SGR escape codes in the content modify the
// current CellStyle, so markdown-rendered bold/italic/color text works correctly.
func rasterizeTextAt(node *vdom.VNode, screen *Screen, offX, offY int) {
	content := node.Props.GetString("content")
	if content == "" {
		return
	}

	baseStyle := CellStyle{
		Bold:      node.Props.GetBool("bold"),
		Dim:       node.Props.GetBool("dim"),
		Italic:    node.Props.GetBool("italic"),
		Underline: node.Props.GetBool("underline"),
	}
	if c, ok := node.Props["color"]; ok {
		baseStyle.FG = toColor(c)
	}
	if c, ok := node.Props["bg"]; ok {
		baseStyle.BG = toColor(c)
	}

	lines := splitLinesANSI(content)
	col := offX
	row := offY

	for _, line := range lines {
		writeANSILine(screen, line, col, row, baseStyle)
		row++
	}
}

// writeANSILine writes a single line of text to the screen, parsing ANSI SGR
// escape codes to dynamically update the character style.
func writeANSILine(screen *Screen, line string, startX, y int, base CellStyle) {
	if y < 0 || y >= screen.Height {
		return
	}
	cur := base
	col := startX
	runes := []rune(line)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Parse SGR sequence until 'm'
			j := i + 2
			for j < len(runes) && runes[j] != 'm' {
				j++
			}
			if j < len(runes) {
				j++ // include 'm'
				cur = applySGRCodes(cur, base, string(runes[i+2:j-1]))
				i = j
				continue
			}
		}
		// Write visible character, accounting for display width (CJK = 2 cells).
		// Double-width characters occupy two terminal columns. Set the second
		// cell to a space so diff comparisons work reliably across redraws.
		rw := runewidth.RuneWidth(r)
		if col >= 0 && col < screen.Width {
			screen.Cells[y][col] = TermCell{Rune: r, Style: cur}
			if rw > 1 && col+1 < screen.Width {
				screen.Cells[y][col+1] = TermCell{Rune: ' ', Style: cur}
			}
		}
		col += rw
		i++
	}
}

// applySGRCodes parses a semicolon-separated list of SGR parameter numbers
// and applies them on top of the base style. A reset (code 0) rolls back to base.
func applySGRCodes(cur, base CellStyle, params string) CellStyle {
	if params == "" || params == "0" {
		return base
	}
	parts := splitSGRParams(params)
	for _, p := range parts {
		switch p {
		case 0:
			cur = base
		case 1:
			cur.Bold = true
			cur.Dim = false
		case 2:
			cur.Dim = true
			cur.Bold = false
		case 3:
			cur.Italic = true
		case 4:
			cur.Underline = true
		case 22:
			cur.Bold = false
			cur.Dim = false
		case 23:
			cur.Italic = false
		case 24:
			cur.Underline = false
		case 39:
			cur.FG = base.FG // default FG
		case 49:
			cur.BG = base.BG // default BG
		}
		// 30-37: standard FG colors
		// 40-47: standard BG colors
		// 38;2;R;G;B: 24-bit FG (handled elsewhere)
		// 48;2;R;G;B: 24-bit BG (handled elsewhere)
	}
	return cur
}

func splitSGRParams(s string) []int {
	var vals []int
	parts := splitStr(s, ";")
	for _, p := range parts {
		n := 0
		for _, r := range p {
			if r >= '0' && r <= '9' {
				n = n*10 + int(r-'0')
			}
		}
		vals = append(vals, n)
	}
	return vals
}

func splitStr(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func toColor(v interface{}) color.Color {
	if c, ok := v.(color.Color); ok {
		return c
	}
	return nil
}

// splitLinesANSI splits a string into lines by '\n'.
func splitLinesANSI(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
