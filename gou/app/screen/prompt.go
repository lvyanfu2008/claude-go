package screen

// PadStreamRows pads (or truncates) a row slice to exactly h lines.
func PadStreamRows(rows []string, h int) []string {
	for len(rows) < h {
		rows = append(rows, "")
	}
	if len(rows) > h {
		return rows[:h]
	}
	return rows
}
