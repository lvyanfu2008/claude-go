package app

import (
	"charm.land/lipgloss/v2"
)

type hooksMenuStyles struct {
	container      lipgloss.Style
	title          lipgloss.Style
	subtitle       lipgloss.Style
	eventItem      lipgloss.Style
	eventCursor    lipgloss.Style
	eventCount     lipgloss.Style
	eventDesc      lipgloss.Style
	matcherItem    lipgloss.Style
	matcherCursor  lipgloss.Style
	matcherSource  lipgloss.Style
	matcherCount   lipgloss.Style
	hookItem       lipgloss.Style
	hookCursor     lipgloss.Style
	hookTypeLabel  lipgloss.Style
	hookSource     lipgloss.Style
	detailLabel    lipgloss.Style
	detailValue    lipgloss.Style
	detailBox      lipgloss.Style
	detailBoxLabel lipgloss.Style
	detailBoxValue lipgloss.Style
	footer         lipgloss.Style
	footerLink     lipgloss.Style
	divider        lipgloss.Style
	separator      string
}

func newHooksMenuStyles(width int) hooksMenuStyles {
	w := max(width-4, 40)
	s := hooksMenuStyles{
		separator: "─",
	}

	s.container = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w)

	s.title = lipgloss.NewStyle().
		Bold(true).
		PaddingTop(1)

	s.subtitle = lipgloss.NewStyle().
		Faint(true)

	s.eventItem = lipgloss.NewStyle().
		Padding(0, 0)

	s.eventCursor = lipgloss.NewStyle().
		Padding(0, 0).
		Reverse(true)

	s.eventCount = lipgloss.NewStyle().
		Faint(true)

	s.eventDesc = lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(2).
		MaxWidth(w - 4)

	s.matcherItem = lipgloss.NewStyle().
		Padding(0, 0)

	s.matcherCursor = lipgloss.NewStyle().
		Padding(0, 0).
		Reverse(true)

	s.matcherSource = lipgloss.NewStyle().
		Faint(true)

	s.matcherCount = lipgloss.NewStyle().
		Faint(true)

	s.hookItem = lipgloss.NewStyle().
		Padding(0, 0)

	s.hookCursor = lipgloss.NewStyle().
		Padding(0, 0).
		Reverse(true)

	s.hookTypeLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))

	s.hookSource = lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(2)

	s.detailLabel = lipgloss.NewStyle().
		Bold(true).
		PaddingTop(1)

	s.detailValue = lipgloss.NewStyle().
		PaddingLeft(2)

	s.detailBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w - 4)

	s.detailBoxLabel = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	s.detailBoxValue = lipgloss.NewStyle().
		PaddingTop(1)

	s.footer = lipgloss.NewStyle().
		Faint(true).
		PaddingTop(1)

	s.footerLink = lipgloss.NewStyle().
		Faint(true).
		PaddingTop(1).
		Foreground(lipgloss.Color("39"))

	s.divider = lipgloss.NewStyle().
		Faint(true)

	return s
}
