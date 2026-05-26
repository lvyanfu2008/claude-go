package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const maxAgentFooterLines = 6

const (
	agentPlayIcon  = "▶"
	agentPauseIcon = "■"
	agentArrowUp   = "↑"
	agentArrowDown = "↓"
)

// AgentFooterView renders the fixed panel below input area:
// main line + agent lines + bg pills row.
func AgentFooterView(mainTask *AgentTaskState, agentTasks []*AgentTaskState, cols int) string {
	if mainTask == nil && len(agentTasks) == 0 {
		return ""
	}

	faint := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	var b strings.Builder

	// Main line
	if mainTask != nil {
		b.WriteString("  ")
		b.WriteString(statusIcon(mainTask.Status))
		b.WriteString(" main")
		elapsed := formatFooterElapsed(mainTask)
		if elapsed != "" {
			b.WriteString(faint.Render(" · " + elapsed))
		}
		if mainTask.Progress != nil {
			if t := formatAgentTokens(mainTask.Progress); t != "" {
				b.WriteString(faint.Render(" · ↑ " + t))
			}
		}
		if mainTask.Status == "running" {
			b.WriteString(faint.Render(" · x to stop"))
		} else if mainTask.Status == "completed" || mainTask.Status == "stopped" {
			b.WriteString(faint.Render(" · done · x to clear"))
		}
		b.WriteByte('\n')
	}

	// Agent lines (capped)
	shown := 0
	maxShow := maxAgentFooterLines
	if mainTask != nil {
		maxShow = maxAgentFooterLines - 1
	}
	for _, t := range agentTasks {
		if shown >= maxShow {
			break
		}
		b.WriteString("  ")
		b.WriteString(statusIcon(t.Status))
		b.WriteString(" ")

		name := t.Name
		if name == "" {
			name = t.AgentType
		}
		b.WriteString(bold.Render("@" + name))

		// Agent type label (if different from name)
		if t.AgentType != "" && t.AgentType != name && t.AgentType != "generalPurpose" {
			b.WriteString(faint.Render(" · " + t.AgentType))
		}

		// Description
		if t.Description != "" {
			b.WriteString(faint.Render(" · " + t.Description))
		}

		elapsed := formatFooterElapsed(t)
		if elapsed != "" {
			b.WriteString(faint.Render(" · " + elapsed))
		}

		if t.Progress != nil {
			arrow := agentArrowUp
			if t.Progress.LastActivity != nil {
				arrow = agentArrowDown
			}
			if tokenText := formatAgentTokens(t.Progress); tokenText != "" {
				b.WriteString(faint.Render(" · " + arrow + " " + tokenText))
			}
		}

		switch t.Status {
		case "completed":
			b.WriteString(faint.Render(" · done · x to clear"))
		case "failed":
			b.WriteString(red.Render(" · error"))
			b.WriteString(faint.Render(" · x to clear"))
		case "killed":
			b.WriteString(faint.Render(" · stopped · x to clear"))
		case "stopped":
			b.WriteString(faint.Render(" · stopped · x to clear"))
		}

		b.WriteByte('\n')
		shown++
	}

	if shown < len(agentTasks) {
		b.WriteString(faint.Render(fmt.Sprintf("  … +%d more", len(agentTasks)-shown)))
		b.WriteByte('\n')
	}

	return b.String()
}

func statusIcon(status string) string {
	switch status {
	case "running":
		return agentPlayIcon
	default:
		return agentPauseIcon
	}
}

func formatFooterElapsed(t *AgentTaskState) string {
	var d time.Duration
	if t.Status == "running" {
		d = time.Since(t.StartTime)
	} else if t.EndTime != nil {
		d = t.EndTime.Sub(t.StartTime)
	}
	if d <= 0 {
		return ""
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fs", float64(d.Milliseconds())/1000)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := math.Floor(d.Minutes())
	seconds := math.Mod(d.Seconds(), 60)
	return fmt.Sprintf("%.0fm %.0fs", minutes, seconds)
}

func formatAgentTokens(p *AgentTaskProgress) string {
	if p == nil || p.TokenCount <= 0 {
		return ""
	}
	if p.TokenCount < 1000 {
		return fmt.Sprintf("%d tokens", p.TokenCount)
	}
	return fmt.Sprintf("%.1fk tokens", float64(p.TokenCount)/1000)
}
