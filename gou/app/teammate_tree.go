// TeammateTree renders running sub-agent activity below the spinner with box-drawing tree structure.
// Matches TS TeammateSpinnerTree + TeammateSpinnerLine rendering.

package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	maxTeammateSubRows = 3
	teammateIdleThresh = 2 * time.Second
)

// TeammateTree renders running sub-agent activity lines with tree structure.
// Returns empty string when no sub-agents are running.
func TeammateTree(tasks []*AgentTaskState) string {
	if len(tasks) == 0 {
		return ""
	}

	faint := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	for i, t := range tasks {
		isLast := i == len(tasks)-1

		// Box-drawing branch character
		branch := "├─"
		if isLast {
			branch = "└─"
		}
		if len(tasks) == 1 {
			branch = "┌─"
		}

		name := t.Name
		if name == "" {
			name = t.AgentType
		}

		activity := teammateActivityText(t)
		// Fallback title: agent type + description when no activity yet
		title := teammateTitle(t, name)

		// Stats: tool uses + tokens
		var stats []string
		if t.Progress != nil {
			if t.Progress.ToolUseCount > 0 {
				noun := "tool uses"
				if t.Progress.ToolUseCount == 1 {
					noun = "tool use"
				}
				stats = append(stats, fmt.Sprintf("%d %s", t.Progress.ToolUseCount, noun))
			}
			if t.Progress.TokenCount > 0 {
				stats = append(stats, formatAgentTokens(t.Progress))
			}
		}

		// Main agent line
		b.WriteString("  ")
		b.WriteString(branch)
		b.WriteString(" ")
		b.WriteString(bold.Render("@" + name))

		if activity != "" {
			b.WriteString(faint.Render(" · " + activity))
		} else if title != "" {
			b.WriteString(faint.Render(" · " + title))
		}

		if len(stats) > 0 {
			b.WriteString(faint.Render(" · " + strings.Join(stats, " · ")))
		}

		b.WriteByte('\n')

		// Sub-rows: recent tool call activities
		subRows := teammateSubRows(t, isLast)
		for _, row := range subRows {
			b.WriteString(row)
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// teammateTitle returns a static label for the agent when there's no activity yet.
func teammateTitle(t *AgentTaskState, name string) string {
	var parts []string
	if t.AgentType != "" && t.AgentType != name && t.AgentType != "generalPurpose" {
		parts = append(parts, t.AgentType)
	}
	if t.Description != "" {
		parts = append(parts, t.Description)
	}
	return strings.Join(parts, " · ")
}

// teammateActivityText returns the activity description with fallback.
func teammateActivityText(t *AgentTaskState) string {
	if t.Progress == nil {
		return ""
	}

	// 1. Most recent activity from RecentActivities
	if len(t.Progress.RecentActivities) > 0 {
		last := t.Progress.RecentActivities[len(t.Progress.RecentActivities)-1]
		if last != "" {
			return last
		}
	}

	// 2. Last activity description
	if t.Progress.LastActivityDesc != "" {
		return t.Progress.LastActivityDesc
	}

	// 3. Idle detection
	if t.Progress.LastActivity != nil && time.Since(*t.Progress.LastActivity) > teammateIdleThresh {
		idle := time.Since(*t.Progress.LastActivity)
		if idle < time.Minute {
			return fmt.Sprintf("Idle for %.0fs", idle.Seconds())
		}
		return fmt.Sprintf("Idle for %.0fm", idle.Minutes())
	}

	return ""
}

// teammateSubRows returns tool call activity sub-rows (up to maxTeammateSubRows).
func teammateSubRows(t *AgentTaskState, isLast bool) []string {
	if t.Progress == nil || len(t.Progress.RecentActivities) == 0 {
		return nil
	}

	activities := t.Progress.RecentActivities
	if len(activities) > maxTeammateSubRows {
		activities = activities[len(activities)-maxTeammateSubRows:]
	}

	faint := lipgloss.NewStyle().Faint(true)
	pipe := faint.Render("│")
	blank := " "

	var rows []string
	for _, act := range activities {
		if act == "" {
			continue
		}

		var prefix string
		if isLast {
			prefix = blank + " " // no pipe for last agent
		} else {
			prefix = pipe + " " // continuation pipe for non-last agents
		}

		branch := faint.Render("⎿  ")
		rows = append(rows, "  "+prefix+branch+act)
	}

	return rows
}
