package config

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// TeardropAsterisk matches TS constants/figures.ts TEARDROP_ASTERISK (Spinner.tsx).
const TeardropAsterisk = "✻"

// SpinnerTickCmd returns a tea.Cmd that fires at 120ms intervals for spinner animation.
func SpinnerTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return SpinnerTickMsg{} })
}
