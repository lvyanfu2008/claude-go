// SpinnerRow renders the query-busy spinner line: glyph + verb + elapsed + tokens + thinking.

package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// SpinnerRow renders the query-busy spinner line: glyph + verb + elapsed + tokens + thinking.
func SpinnerRow(verb string, frame int, startedAt time.Time, tokenCount int, thinking bool, cols int) string {
	if verb == "" {
		verb = "Working"
	}

	frames := []string{"…", ".", "..", "..."}
	sfx := frames[frame%len(frames)]

	bold := lipgloss.NewStyle().Bold(true)
	faint := lipgloss.NewStyle().Faint(true)

	var b strings.Builder
	b.WriteString(bold.Render(teardropAsterisk + " " + verb + sfx))

	elapsed := formatSpinnerElapsed(startedAt)
	if elapsed != "" {
		b.WriteString(faint.Render(" · " + elapsed))
	}

	if tokenCount > 0 {
		b.WriteString(faint.Render(" · " + formatSpinnerTokens(tokenCount) + " tokens"))
	}

	if thinking {
		b.WriteString(faint.Render(" · thinking…"))
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
