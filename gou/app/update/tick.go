package update

import (
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"goc/gou/app/config"
	state "goc/gou/app/state"
)

func (d *Dispatcher) handleSpinnerTick(_ config.SpinnerTickMsg) (tea.Model, tea.Cmd) {
	if !d.deps.GetQuery().Busy {
		return d.deps.Model(), nil
	}
	d.deps.GetQuery().SpinnerFrame++
	return d.deps.Model(), config.SpinnerTickCmd()
}

func (d *Dispatcher) handleToolSummaryDelayTick(_ config.ToolSummaryDelayTickMsg) (tea.Model, tea.Cmd) {
	delay := gouDemoToolUseSummaryDelay()
	if delay <= 0 {
		return d.deps.Model(), nil
	}
	if d.deps.GetScreen().Mode == state.ScreenPrompt && d.deps.AnyToolSummaryDelayPending() {
		d.deps.RebuildHeightCache()
	}

	return d.deps.Model(), tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return config.ToolSummaryDelayTickMsg{} })
}

// gouDemoToolUseSummaryDelay returns how long to show full Grep/Glob/Read chrome before merged summary lines (prompt only).
// Empty/unset env defaults to 2s; 0 disables. Negative or invalid values are treated as 0.
func gouDemoToolUseSummaryDelay() time.Duration {
	v := strings.TrimSpace(os.Getenv("GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS"))
	if v == "" {
		return 2 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}
