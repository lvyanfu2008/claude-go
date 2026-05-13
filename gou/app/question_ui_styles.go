package app

import (
	"charm.land/lipgloss/v2"
)

type questionStyles struct {
	container       lipgloss.Style
	navBar          lipgloss.Style
	navTab          lipgloss.Style
	navTabActive    lipgloss.Style
	navTabAnswered  lipgloss.Style
	navTabCurrent   lipgloss.Style
	navSubmit       lipgloss.Style
	navSubmitActive lipgloss.Style
	title           lipgloss.Style
	option          lipgloss.Style
	optionSelected  lipgloss.Style
	optionCursor    lipgloss.Style
	optionDesc      lipgloss.Style
	optionLabel     lipgloss.Style
	footer          lipgloss.Style
	footerBold      lipgloss.Style
	divider         lipgloss.Style
	reviewTitle     lipgloss.Style
	reviewQA        lipgloss.Style
	reviewAnswer    lipgloss.Style
	reviewWarning   lipgloss.Style
	separator       string
}

func newQuestionStyles(width int) questionStyles {
	w := max(width-4, 40)
	s := questionStyles{
		separator: "─",
	}
	s.container = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(w)

	s.navBar = lipgloss.NewStyle().
		PaddingBottom(0)

	s.navTab = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("245"))

	s.navTabActive = lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("237"))

	s.navTabAnswered = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("120"))

	s.navTabCurrent = lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("39"))

	s.navSubmit = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("245"))

	s.navSubmitActive = lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("237"))

	s.title = lipgloss.NewStyle().
		Bold(true).
		PaddingTop(1).
		PaddingBottom(0)

	s.option = lipgloss.NewStyle().
		Padding(0, 0)

	s.optionSelected = lipgloss.NewStyle().
		Padding(0, 0).
		Bold(true).
		Foreground(lipgloss.Color("39"))

	s.optionCursor = lipgloss.NewStyle().
		Padding(0, 0).
		Reverse(true)

	s.optionDesc = lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(4).
		MaxWidth(w - 4)

	s.optionLabel = lipgloss.NewStyle().
		Padding(0, 0)

	s.footer = lipgloss.NewStyle().
		Faint(true).
		PaddingTop(1)

	s.footerBold = lipgloss.NewStyle().
		PaddingTop(1)

	s.divider = lipgloss.NewStyle().
		Faint(true)

	s.reviewTitle = lipgloss.NewStyle().
		Bold(true).
		PaddingTop(1).
		PaddingBottom(1)

	s.reviewQA = lipgloss.NewStyle().
		Bold(true).
		PaddingTop(1)

	s.reviewAnswer = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("39"))

	s.reviewWarning = lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(lipgloss.Color("214"))

	return s
}
