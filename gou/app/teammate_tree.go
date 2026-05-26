package app

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const idleThreshold = 2 * time.Second

// TeammateTree renders running sub-agent activity lines below the spinner.
// Returns empty string when no sub-agents are running.
func TeammateTree(tasks []*AgentTaskState) string {
	if len(tasks) == 0 {
		return ""
	}

	faint := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	for _, t := range tasks {
		name := t.Name
		if name == "" {
			name = t.AgentType
		}

		activity := activityDesc(t)
		tokenText := formatAgentTokens(t.Progress)

		b.WriteString("  ")
		b.WriteString(bold.Render("@" + name))

		if activity != "" {
			b.WriteString(faint.Render(" · " + activity))
		}

		if tokenText != "" {
			b.WriteString(faint.Render(" · ↑ " + tokenText))
		}

		b.WriteByte('\n')
	}

	return b.String()
}

func activityDesc(t *AgentTaskState) string {
	if t.Progress != nil {
		if t.Progress.LastActivityDesc != "" {
			return t.Progress.LastActivityDesc
		}
		if t.Progress.LastActivity != nil && time.Since(*t.Progress.LastActivity) > idleThreshold {
			return "Idle"
		}
	}
	return ""
}
