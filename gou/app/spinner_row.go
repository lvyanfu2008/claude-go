// SpinnerRow renders the query-busy spinner line matching TS SpinnerAnimationRow format:
// ✶ Verb… (elapsed · ↓ tokens · thinking)

package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const spinnerStuckThreshold = 3 * time.Second

// SpinnerRow renders the query-busy spinner line matching TS SpinnerAnimationRow format.
func SpinnerRow(verb string, frame int, startedAt time.Time, tokenCount int, thinking bool, lastActivity time.Time, cols int) string {
	if verb == "" {
		verb = "Working"
	}

	// Slow star glyph animation (changes every ~720ms at 120ms tick)
	starGlyphs := []string{"✶", "✷", "✸", "✹"}
	glyph := starGlyphs[(frame/6)%len(starGlyphs)]

	// Stuck detection: red after 3s of inactivity (matching TS stuck detection)
	var glyphStyle lipgloss.Style
	if !lastActivity.IsZero() && time.Since(lastActivity) > spinnerStuckThreshold {
		glyphStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true) // red
	} else {
		glyphStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // dark red
	}
	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	b.WriteString(glyphStyle.Render(glyph))
	b.WriteString(" ")
	b.WriteString(verb)
	b.WriteString("…") // … fixed

	var stats []string

	elapsed := formatSpinnerElapsed(startedAt)
	if elapsed != "" {
		stats = append(stats, elapsed)
	}

	if tokenCount > 0 {
		stats = append(stats, "↓ "+formatSpinnerTokens(tokenCount)) // ↓
	}

	if thinking {
		stats = append(stats, "thinking")
	}

	if len(stats) > 0 {
		b.WriteString(faint.Render(" (" + strings.Join(stats, " · ") + ")")) // ·
	}

	return b.String()
}

func formatSpinnerElapsed(startedAt time.Time) string {
	if startedAt.IsZero() {
		return ""
	}
	d := time.Since(startedAt)
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	minutes := math.Floor(d.Minutes())
	seconds := math.Mod(d.Seconds(), 60)
	return fmt.Sprintf("%.0fm %.0fs", minutes, seconds)
}

func formatSpinnerTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}
