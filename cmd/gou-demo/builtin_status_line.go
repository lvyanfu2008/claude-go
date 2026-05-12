// Builtin status row mirrors TS PromptInputFooter StatusLine + BuiltinStatusLine.tsx
// (model · Context % · tokens · optional cost · Debug mode).

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"goc/modelenv"
)

func gouDemoBuiltinStatusLineDisabled() bool {
	return gouDemoEnvTruthy("GOU_DEMO_NO_BUILTIN_STATUS")
}

func gouDemoDebugModeFooter() bool {
	return gouDemoEnvTruthy("GOU_DEMO_DEBUG") || gouDemoEnvTruthy("CLAUDE_CODE_DEBUG")
}

func shortModelDisplay(modelName string) string {
	parts := strings.Fields(strings.TrimSpace(modelName))
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "gou-demo"
}

func sessionCostUSDFromEnv() float64 {
	v := strings.TrimSpace(os.Getenv("GOU_DEMO_SESSION_COST_USD"))
	if v == "" {
		return 0
	}
	x, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return x
}

func formatCostUSD(x float64) string {
	if x < 0.0001 {
		return "$0"
	}
	return fmt.Sprintf("$%.4f", x)
}

func (m *model) builtinStatusLineView() string {
	if gouDemoBuiltinStatusLineDisabled() || m.uiScreen == gouDemoScreenTranscript {
		return ""
	}
	modelName := modelenv.EffectiveMainLoopModel()
	sep := lipgloss.NewStyle().Faint(true).Render(" │ ")
	var b strings.Builder

	if m.queryBusy {
		verb := strings.TrimSpace(m.spinnerVerb)
		if verb == "" {
			verb = "Flowing"
		}
		frames := []string{"…", ".", "..", "..."}
		sfx := frames[m.spinnerFrame%len(frames)]
		spinner := lipgloss.NewStyle().Bold(true).Render(teardropAsterisk + " " + verb + sfx)
		b.WriteString(spinner)
		b.WriteByte('\n')
	}

	b.WriteString(lipgloss.NewStyle().Render(shortModelDisplay(modelName)))
	if c := sessionCostUSDFromEnv(); c > 0 {
		b.WriteString(sep)
		b.WriteString(lipgloss.NewStyle().Render(formatCostUSD(c)))
	}
	if gouDemoDebugModeFooter() {
		b.WriteString(sep)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Debug mode"))
	}
	return lipgloss.NewStyle().MaxWidth(m.cols).Render(b.String())
}
