package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const (
	agentPlayIcon  = "▶" // ▶
	agentPauseIcon = "■" // ■
	agentArrowUp   = "↑" // ↑
	agentArrowDown = "↓" // ↓
)

// agentCoordinatorView renders the agent coordinator panel matching TS CoordinatorTaskPanel.
// Returns empty string when no tasks are visible.
func (m *model) agentCoordinatorView() string {
	if m.Agent == nil || m.Agent.Tasks == nil {
		return ""
	}
	tasks := m.Agent.Tasks.(*agentTaskStore).VisibleTasks()
	if len(tasks) == 0 {
		return ""
	}

	faint := lipgloss.NewStyle().Faint(true)
	bold := lipgloss.NewStyle().Bold(true)

	var b strings.Builder
	// Main row
	b.WriteString("  ")
	b.WriteString(agentPauseIcon)
	b.WriteString(" main\n")

	for _, t := range tasks {
		isRunning := t.Status == "running"
		sep := agentPlayIcon
		if !isRunning {
			sep = agentPauseIcon
		}

		elapsed := formatAgentElapsed(t)
		tokenText := formatAgentTokens(t.Progress)

		// Arrow direction: ↓ when last activity was receiving, ↑ otherwise
		arrow := agentArrowUp
		if t.Progress != nil && t.Progress.LastActivity != nil {
			arrow = agentArrowDown
		}

		// Description: summary > last activity desc > static description
		desc := t.Description
		if t.Progress != nil && t.Progress.Summary != "" {
			desc = t.Progress.Summary
		} else if t.Progress != nil && t.Progress.LastActivityDesc != "" {
			desc = t.Progress.LastActivityDesc
		}

		name := t.Name
		if name == "" {
			name = t.AgentType
		}

		// Build line: "  ▶ name: desc ▶ 12.3s · ↓ 4.2k tokens"
		b.WriteString("  ")
		b.WriteString(sep)
		b.WriteString(" ")
		b.WriteString(bold.Render(name))
		b.WriteString(": ")
		b.WriteString(desc)
		b.WriteString(" ")
		b.WriteString(sep)
		b.WriteString(" ")
		b.WriteString(elapsed)
		if tokenText != "" {
			b.WriteString(" · ") // ·
			b.WriteString(arrow)
			b.WriteString(" ")
			b.WriteString(tokenText)
		}
		if !isRunning {
			b.WriteString(faint.Render(" · x to clear"))
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func formatAgentElapsed(t *AgentTaskState) string {
	isRunning := t.Status == "running"
	elapsed := time.Duration(0)
	if isRunning {
		elapsed = time.Since(t.StartTime)
	} else if t.EndTime != nil {
		elapsed = t.EndTime.Sub(t.StartTime)
	}

	if elapsed <= 0 {
		return "0s"
	}
	if elapsed < time.Second {
		return fmt.Sprintf("%.1fs", float64(elapsed.Milliseconds())/1000)
	}
	if elapsed < time.Minute {
		return fmt.Sprintf("%.1fs", elapsed.Seconds())
	}
	minutes := math.Floor(elapsed.Minutes())
	seconds := math.Mod(elapsed.Seconds(), 60)
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
