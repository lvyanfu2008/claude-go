package ink

import (
	"fmt"
	"image/color"
	"strings"
)

func styleToSGR(s CellStyle) string {
	var codes []string
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	if fg := colorToANSIFG(s.FG); fg != "" {
		codes = append(codes, fg)
	}
	if bg := colorToANSIBG(s.BG); bg != "" {
		codes = append(codes, bg)
	}
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func colorToANSIFG(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("38;2;%d;%d;%d", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func colorToANSIBG(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("48;2;%d;%d;%d", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func sgrReset() string { return "\x1b[0m" }

func cursorTo(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row+1, col+1)
}

func cursorUp(n int) string   { return fmt.Sprintf("\x1b[%dA", n) }
func cursorDown(n int) string { return fmt.Sprintf("\x1b[%dB", n) }
func eraseToEnd() string      { return "\x1b[0K" }
func eraseLine() string       { return "\x1b[2K" }
func eraseDisplay() string    { return "\x1b[2J" }
