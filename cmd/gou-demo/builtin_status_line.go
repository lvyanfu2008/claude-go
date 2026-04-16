// Builtin status row mirrors TS PromptInputFooter StatusLine + BuiltinStatusLine.tsx
// (model · Context % · tokens · optional cost · Debug mode).

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"goc/gou/pui"
	"goc/modelenv"
	"goc/types"
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

func formatTokensShort(n int) string {
	if n < 0 {
		n = 0
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.0fM", float64(n)/1e6)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.0fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func estimateMessagesSizeTokens(msgs []types.Message) int {
	n := 0
	for _, msg := range msgs {
		n += len(msg.Content) + len(msg.Attachment) + len(msg.Message)
	}
	return n / 4
}

func defaultContextWindowForModel(modelName string) int {
	// TS getContextWindowForModel is richer; gou-demo uses a conservative default.
	_ = modelName
	if v := strings.TrimSpace(os.Getenv("GOU_DEMO_CONTEXT_WINDOW")); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 {
			return x
		}
	}
	return 200_000
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

// effectiveUsedTokens prefers ccbstream usage totals when any usage line was applied (TS session totals).
func (m *model) effectiveUsedTokens() int {
	if m.store == nil {
		return 0
	}
	if u := m.store.TotalUsageTokens(); u > 0 {
		return u
	}
	return estimateMessagesSizeTokens(m.store.Messages)
}

func (m *model) builtinStatusLineView() string {
	if gouDemoBuiltinStatusLineDisabled() || m.uiScreen == gouDemoScreenTranscript {
		return ""
	}
	modelName := strings.TrimSpace(m.lastMainLoopModel)
	if modelName == "" {
		modelName = modelenv.FirstNonEmpty()
	}
	if modelName == "" {
		modelName = pui.DefaultMainLoopModelForDemo()
	}
	used := m.effectiveUsedTokens()
	win := defaultContextWindowForModel(modelName)
	pct := 0
	if win > 0 {
		pct = int(float64(used) / float64(win) * 100)
		if pct > 999 {
			pct = 999
		}
	}
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
	b.WriteString(sep)
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("Context "))
	b.WriteString(lipgloss.NewStyle().Render(fmt.Sprintf("%d%%", pct)))
	if m.cols >= 60 {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(" (%s/%s)", formatTokensShort(used), formatTokensShort(win))))
	}
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
