package markdown

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

// TableToken holds a parsed markdown table's headers, rows, and column alignments.
type TableToken struct {
	Headers []string
	Rows    [][]string
	Aligns  []string
}

// ParseSimpleTable attempts to parse a GFM-style markdown table from raw text.
// It expects at least a header row and a separator line, optionally followed by data rows.
//
//	| Header 1 | Header 2 |
//	|----------|----------|
//	| Cell A1  | Cell B1  |
func ParseSimpleTable(text string) (*TableToken, bool) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) < 2 {
		return nil, false
	}

	// Find the separator line (dashes + pipes pattern)
	sepIdx := -1
	for i, line := range lines {
		if isTableSeparatorLine(line) {
			sepIdx = i
			break
		}
	}
	if sepIdx < 1 {
		return nil, false
	}

	headers := splitTableRow(lines[sepIdx-1])
	if len(headers) == 0 {
		return nil, false
	}

	aligns := parseAligns(lines[sepIdx])

	var rows [][]string
	for _, line := range lines[sepIdx+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	}

	if len(headers) == 0 && len(rows) == 0 {
		return nil, false
	}

	return &TableToken{Headers: headers, Rows: rows, Aligns: aligns}, true
}

// RenderTable renders a TableToken as a Unicode box-drawing table.
// The theme parameter is reserved for future styling use.
func RenderTable(t TableToken, theme lipgloss.Style) string {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return ""
	}

	numCols := len(t.Headers)

	// Ensure aligns slice matches the number of columns
	if len(t.Aligns) == 0 {
		t.Aligns = make([]string, numCols)
		for i := range t.Aligns {
			t.Aligns[i] = "left"
		}
	}
	for len(t.Aligns) < numCols {
		t.Aligns = append(t.Aligns, "left")
	}

	// Calculate maximum column widths across headers and all rows
	colWidths := make([]int, numCols)
	for i, h := range t.Headers {
		colWidths[i] = runeLen(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < numCols {
				w := runeLen(cell)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}
	// Enforce minimum column width (3 characters: "   ")
	for i := range colWidths {
		if colWidths[i] < 3 {
			colWidths[i] = 3
		}
	}

	var b strings.Builder

	// Top border: ┌───┬───┐
	b.WriteString("┌")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┬")
		}
	}
	b.WriteString("┐\n")

	// Header row: │ val │ val │
	b.WriteString("│")
	for i, h := range t.Headers {
		b.WriteString(" " + padCell(h, colWidths[i], t.Aligns[i], true) + " ")
		b.WriteString("│")
	}
	b.WriteString("\n")

	// Header/data separator: ├───┼───┤
	b.WriteString("├")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┼")
		}
	}
	b.WriteString("┤\n")

	// Data rows: │ val │ val │
	for _, row := range t.Rows {
		b.WriteString("│")
		for i := 0; i < numCols; i++ {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			align := "left"
			if i < len(t.Aligns) {
				align = t.Aligns[i]
			}
			b.WriteString(" " + padCell(cell, colWidths[i], align, false) + " ")
			b.WriteString("│")
		}
		b.WriteString("\n")
	}

	// Bottom border: └───┴───┘
	b.WriteString("└")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┴")
		}
	}
	b.WriteString("┘")

	_ = theme // reserved for future styling use
	return b.String()
}

// splitTableRow splits a pipe-delimited table row into trimmed cell values.
// Handles leading and trailing pipes: "| a | b |" -> ["a", "b"]
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	// Strip leading and trailing pipes
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// parseAligns extracts column alignment from a GFM table separator line.
func parseAligns(sepLine string) []string {
	cells := splitTableRow(sepLine)
	aligns := make([]string, 0, len(cells))
	for _, cell := range cells {
		switch {
		case strings.HasPrefix(cell, ":") && strings.HasSuffix(cell, ":"):
			aligns = append(aligns, "center")
		case strings.HasSuffix(cell, ":"):
			aligns = append(aligns, "right")
		default:
			aligns = append(aligns, "left")
		}
	}
	return aligns
}

// padCell pads a cell value to the given width using spaces, according to alignment.
// If the cell value is wider than width, it truncates with an ellipsis character ("…").
// The header parameter is reserved for future use (e.g., bold headers).
func padCell(s string, width int, align string, _ bool) string {
	w := runeLen(s)
	if w > width {
		// Truncate with ellipsis when too wide
		runes := []rune(s)
		if width >= 2 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	pad := width - w
	switch align {
	case "center":
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	case "right":
		return strings.Repeat(" ", pad) + s
	default: // "left"
		return s + strings.Repeat(" ", pad)
	}
}

// runeLen returns the number of Unicode runes in s, used as visual column width.
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
