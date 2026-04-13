package viewportfold

import "strings"

// Section is one collapsible block (title + body), matching the go-tui demo model.
type Section struct {
	Title     string
	Content   string
	Collapsed bool
}

// ToggleAll flips Collapsed on every section in place (same semantics as go-tui Ctrl+O).
func ToggleAll(sections []Section) {
	for i := range sections {
		sections[i].Collapsed = !sections[i].Collapsed
	}
}

// AppendSections writes each section to b: header line with ▼/▶ icon, then body or folded placeholder.
func AppendSections(b *strings.Builder, sections []Section) {
	for _, sec := range sections {
		b.WriteString("\n\n")
		icon := "▼"
		if sec.Collapsed {
			icon = "▶"
		}
		b.WriteString(icon)
		b.WriteByte(' ')
		b.WriteString(sec.Title)
		if !sec.Collapsed {
			b.WriteString("\n  ")
			b.WriteString(sec.Content)
		} else {
			b.WriteString("\n  [内容已折叠...]")
		}
	}
}
